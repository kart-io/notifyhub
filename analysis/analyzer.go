package analysis

import (
	"context"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/core/sending"
)

// Analyzer provides analysis of sending results and patterns
type Analyzer struct {
	metrics      *Metrics
	patterns     *PatternDetector
	mutex        sync.RWMutex
	enabledRules []Rule
}

// Rule defines an analysis rule
type Rule interface {
	Analyze(ctx context.Context, results *sending.SendingResults) (*Finding, error)
	Name() string
	Enabled() bool
}

// Finding represents an analysis finding
type Finding struct {
	RuleName    string                 `json:"rule_name"`
	Severity    Severity               `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// Severity levels for findings
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// NewAnalyzer creates a new analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		metrics:      NewMetrics(),
		patterns:     NewPatternDetector(),
		enabledRules: make([]Rule, 0),
	}
}

// AddRule adds an analysis rule
func (a *Analyzer) AddRule(rule Rule) {
	if rule.Enabled() {
		a.mutex.Lock()
		a.enabledRules = append(a.enabledRules, rule)
		a.mutex.Unlock()
	}
}

// AnalyzeResults analyzes sending results
func (a *Analyzer) AnalyzeResults(ctx context.Context, results *sending.SendingResults) (*AnalysisReport, error) {
	report := &AnalysisReport{
		Timestamp: time.Now(),
		Findings:  make([]*Finding, 0),
		Metrics:   a.metrics.CalculateMetrics(results),
	}

	// Update metrics
	a.metrics.UpdateFromResults(results)

	// Run analysis rules
	a.mutex.RLock()
	rules := make([]Rule, len(a.enabledRules))
	copy(rules, a.enabledRules)
	a.mutex.RUnlock()

	for _, rule := range rules {
		finding, err := rule.Analyze(ctx, results)
		if err != nil {
			// Log error but continue with other rules
			continue
		}
		if finding != nil {
			report.Findings = append(report.Findings, finding)
		}
	}

	// Detect patterns
	patterns := a.patterns.DetectPatterns(results)
	for _, pattern := range patterns {
		finding := &Finding{
			RuleName:    "pattern_detector",
			Severity:    pattern.Severity,
			Title:       pattern.Title,
			Description: pattern.Description,
			Metadata:    pattern.Metadata,
			Timestamp:   time.Now(),
		}
		report.Findings = append(report.Findings, finding)
	}

	return report, nil
}

// GetMetrics returns current metrics
func (a *Analyzer) GetMetrics() *MetricsSnapshot {
	return a.metrics.GetSnapshot()
}

// AnalysisReport contains the results of analysis
type AnalysisReport struct {
	Timestamp time.Time        `json:"timestamp"`
	Findings  []*Finding       `json:"findings"`
	Metrics   *MetricsSnapshot `json:"metrics"`
}

// HighFailureRateRule detects high failure rates
type HighFailureRateRule struct {
	threshold float64
}

// NewHighFailureRateRule creates a new high failure rate rule
func NewHighFailureRateRule(threshold float64) *HighFailureRateRule {
	return &HighFailureRateRule{threshold: threshold}
}

// Analyze analyzes for high failure rates
func (r *HighFailureRateRule) Analyze(ctx context.Context, results *sending.SendingResults) (*Finding, error) {
	if results.Total == 0 {
		return nil, nil
	}

	failureRate := float64(results.Failed) / float64(results.Total)
	if failureRate > r.threshold {
		return &Finding{
			RuleName:    r.Name(),
			Severity:    SeverityWarning,
			Title:       "High Failure Rate Detected",
			Description: "The message delivery failure rate is above the threshold",
			Metadata: map[string]interface{}{
				"failure_rate": failureRate,
				"threshold":    r.threshold,
				"total":        results.Total,
				"failed":       results.Failed,
			},
			Timestamp: time.Now(),
		}, nil
	}

	return nil, nil
}

// Name returns the rule name
func (r *HighFailureRateRule) Name() string {
	return "high_failure_rate"
}

// Enabled returns if the rule is enabled
func (r *HighFailureRateRule) Enabled() bool {
	return true
}

// PlatformFailureRule detects platform-specific failures
type PlatformFailureRule struct{}

// NewPlatformFailureRule creates a new platform failure rule
func NewPlatformFailureRule() *PlatformFailureRule {
	return &PlatformFailureRule{}
}

// Analyze analyzes for platform-specific failures
func (r *PlatformFailureRule) Analyze(ctx context.Context, results *sending.SendingResults) (*Finding, error) {
	platformFailures := make(map[string]int)
	platformTotals := make(map[string]int)

	for _, result := range results.Results {
		platform := result.Target.Platform
		platformTotals[platform]++
		if result.IsFailed() {
			platformFailures[platform]++
		}
	}

	for platform, failures := range platformFailures {
		total := platformTotals[platform]
		if total > 0 && failures == total {
			return &Finding{
				RuleName:    r.Name(),
				Severity:    SeverityError,
				Title:       "Complete Platform Failure",
				Description: "All messages failed for a specific platform",
				Metadata: map[string]interface{}{
					"platform": platform,
					"failures": failures,
					"total":    total,
				},
				Timestamp: time.Now(),
			}, nil
		}
	}

	return nil, nil
}

// Name returns the rule name
func (r *PlatformFailureRule) Name() string {
	return "platform_failure"
}

// Enabled returns if the rule is enabled
func (r *PlatformFailureRule) Enabled() bool {
	return true
}
