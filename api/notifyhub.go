package api

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/analysis"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/core/hub"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/middleware"
	"github.com/kart-io/notifyhub/targeting"
	"github.com/kart-io/notifyhub/transport/email"
	"github.com/kart-io/notifyhub/transport/feishu"
)

// NotifyHub is the main API entry point for the notification system
type NotifyHub struct {
	hub      *hub.Hub
	resolver *targeting.Resolver
	analyzer *analysis.Analyzer
	config   *config.Config
	logger   logger.Interface
}

// Options for NotifyHub configuration
type Options struct {
	Logger logger.Interface
}

// New creates a new NotifyHub instance with the given configuration
func New(cfg *config.Config, opts *Options) (*NotifyHub, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if opts == nil {
		opts = &Options{}
	}

	// Create unified logger from configuration or options
	var unifiedLogger logger.Interface
	if opts.Logger != nil {
		unifiedLogger = opts.Logger
	} else {
		unifiedLogger = createUnifiedLogger(cfg)
	}

	// Create hub with unified logger
	hubOpts := &hub.Options{
		Logger: unifiedLogger,
	}
	h := hub.NewHub(hubOpts)

	// Register transports based on configuration
	if err := registerTransports(h, cfg); err != nil {
		return nil, fmt.Errorf("failed to register transports: %w", err)
	}

	// Add middleware with unified logger
	addMiddleware(h, cfg, unifiedLogger)

	// Create target resolver
	resolver := targeting.NewResolver()
	setupTargetResolver(resolver, cfg)

	// Create analyzer
	analyzer := analysis.NewAnalyzer()
	setupAnalyzer(analyzer)

	return &NotifyHub{
		hub:      h,
		resolver: resolver,
		analyzer: analyzer,
		config:   cfg,
		logger:   unifiedLogger,
	}, nil
}

// registerTransports registers platform transports based on configuration
func registerTransports(h *hub.Hub, cfg *config.Config) error {
	// Register Feishu transport if configured
	if feishuCfg := cfg.Feishu(); feishuCfg != nil {
		transport := feishu.NewTransport(&feishu.Config{
			WebhookURL: feishuCfg.WebhookURL,
			Secret:     feishuCfg.Secret,
			Timeout:    feishuCfg.Timeout,
		})
		h.RegisterTransport(transport)
	}

	// Register Email transport if configured
	if emailCfg := cfg.Email(); emailCfg != nil {
		transport := email.NewTransport(&email.Config{
			Host:     emailCfg.Host,
			Port:     emailCfg.Port,
			Username: emailCfg.Username,
			Password: emailCfg.Password,
			From:     emailCfg.From,
			UseTLS:   emailCfg.UseTLS,
			Timeout:  emailCfg.Timeout,
		})
		h.RegisterTransport(transport)
	}

	return nil
}

// addMiddleware adds middleware to the hub
func addMiddleware(h *hub.Hub, cfg *config.Config, unifiedLogger logger.Interface) {
	// Use the unified logger directly - no adapter needed
	loggerForMiddleware := unifiedLogger

	// Add routing middleware if routing rules are configured
	if routingRules := cfg.Routing(); len(routingRules) > 0 {
		routingMiddleware := middleware.NewRoutingMiddleware(routingRules, loggerForMiddleware)
		h.AddMiddleware(routingMiddleware)
	}

	// Add rate limiting middleware
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(loggerForMiddleware)
	// Configure rate limits based on configuration (could be added to config)
	rateLimitMiddleware.SetLimit("feishu", 10.0, 50) // 10 req/sec, burst 50
	rateLimitMiddleware.SetLimit("email", 5.0, 20)   // 5 req/sec, burst 20
	h.AddMiddleware(rateLimitMiddleware)

	// Add retry middleware
	retryMiddleware := middleware.NewRetryMiddleware(3, 1*time.Second, loggerForMiddleware)
	h.AddMiddleware(retryMiddleware)
}

// createUnifiedLogger creates a unified logger from the configuration
func createUnifiedLogger(cfg *config.Config) logger.Interface {
	// Get the base logger from configuration
	baseLogger := cfg.Logger()

	// If no logger provided, use the default from logger package
	if baseLogger == nil {
		return logger.Default
	}

	// If it's already a logger.Interface, return it directly
	// if l, ok := baseLogger.(logger.Interface); ok {
	// 	return l
	// }

	// If it's an unknown type, use default logger
	// In a real implementation, you might want to add more type checking here
	return logger.Default
}

// setupTargetResolver sets up the target resolver
func setupTargetResolver(resolver *targeting.Resolver, cfg *config.Config) {
	// Register static provider
	staticProvider := targeting.NewStaticProvider()

	// Add default target groups (these could come from configuration)
	defaultTargets := []sending.Target{
		sending.NewTarget(sending.TargetTypeEmail, "admin@company.com", "email"),
		sending.NewTarget(sending.TargetTypeGroup, "alerts", "feishu"),
	}
	staticProvider.AddTargetGroup("default", defaultTargets)

	resolver.RegisterProvider("static", staticProvider)

	// Register direct provider
	directProvider := targeting.NewDirectProvider()
	resolver.RegisterProvider("direct", directProvider)
}

// setupAnalyzer sets up the analyzer with default rules
func setupAnalyzer(analyzer *analysis.Analyzer) {
	// Add analysis rules
	analyzer.AddRule(analysis.NewHighFailureRateRule(0.3)) // Alert if >30% failure rate
	analyzer.AddRule(analysis.NewPlatformFailureRule())
}

// Send sends a message to the specified targets
func (nh *NotifyHub) Send(ctx context.Context, msg *message.Message, targets []sending.Target) (*sending.SendingResults, error) {
	if nh.logger != nil {
		nh.logger.Info(ctx, "sending notification", "message_id", msg.ID, "targets", len(targets))
	}

	// Send through hub
	results, err := nh.hub.Send(ctx, msg, targets)
	if err != nil {
		if nh.logger != nil {
			nh.logger.Error(ctx, "failed to send notification", "error", err)
		}
		return nil, err
	}

	// Analyze results
	go func() {
		if report, analyzeErr := nh.analyzer.AnalyzeResults(context.Background(), results); analyzeErr == nil {
			if len(report.Findings) > 0 && nh.logger != nil {
				nh.logger.Info(context.Background(), "analysis findings", "count", len(report.Findings))
			}
		}
	}()

	return results, nil
}

// SendToTargetExpressions sends a message to targets resolved from expressions
func (nh *NotifyHub) SendToTargetExpressions(ctx context.Context, msg *message.Message, expressions []string) (*sending.SendingResults, error) {
	// Resolve target expressions
	targets, err := nh.resolver.ResolveTargets(ctx, expressions)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve targets: %w", err)
	}

	return nh.Send(ctx, msg, targets)
}

// NewMessage creates a new message builder using the new architecture
func (nh *NotifyHub) NewMessage() *message.Message {
	return message.NewMessage()
}

// GetMetrics returns current metrics
func (nh *NotifyHub) GetMetrics() *analysis.MetricsSnapshot {
	return nh.analyzer.GetMetrics()
}

// GetTransports returns list of registered transport names
func (nh *NotifyHub) GetTransports() []string {
	return nh.hub.ListTransports()
}

// Shutdown gracefully shuts down the NotifyHub
func (nh *NotifyHub) Shutdown(ctx context.Context) error {
	if nh.logger != nil {
		nh.logger.Info(context.Background(), "shutting down NotifyHub")
	}

	return nh.hub.Shutdown(ctx)
}

// IsShutdown returns true if the hub is shut down
func (nh *NotifyHub) IsShutdown() bool {
	return nh.hub.IsShutdown()
}
