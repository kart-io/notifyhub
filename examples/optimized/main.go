package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers")

func main() {
	ctx := context.Background()

	// Create hub with enhanced configuration
	hub, err := client.New(
		config.WithFeishu("https://httpbin.org/post", ""),
		config.WithEmail("localhost", 587, "test", "test", "test@example.com"),
		config.WithQueue("memory", 1000, 2),
	)
	if err != nil {
		log.Fatalf("Failed to create hub: %v", err)
	}

	// Validate configuration before starting
	if err := hub.ValidateAndReport(ctx); err != nil {
		log.Printf("Configuration validation failed: %v", err)
	}

	// Setup middleware
	hub.SetupDefaultMiddleware()

	// Start the hub
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("Failed to start hub: %v", err)
	}
	defer hub.Stop()

	// ================================
	// Example 1: Quick Send API
	// ================================
	fmt.Println("=== Example 1: Quick Send API ===")

	// Simple email send
	err = hub.Email(ctx, "Test Email", "Hello from NotifyHub!", "user@example.com")
	if err != nil {
		log.Printf("Email send error: %v", err)
	}

	// Quick send with target strings
	err = hub.QuickSend(ctx, "Quick Alert", "System is down!",
		"email:admin@example.com",
		"group:ops-team@feishu",
		"user:john@feishu")
	if err != nil {
		log.Printf("Quick send error: %v", err)
	}

	// ================================
	// Example 2: Batch Operations with TargetList
	// ================================
	fmt.Println("\n=== Example 2: Batch Operations ===")

	// Create a target list
	targetList := client.NewTargetList().
		AddEmails("user1@example.com", "user2@example.com").
		AddFeishuGroups("dev-team", "ops-team").
		AddFeishuUsers("alice", "bob")

	fmt.Printf("Target list created: %s\n", targetList.String())

	// Send to target list
	message := client.NewAlert("Batch Alert", "This is a batch notification").Build()
	results, err := hub.SendToTargetList(ctx, message, targetList, nil)
	if err != nil {
		log.Printf("Batch send error: %v", err)
	} else {
		fmt.Printf("Batch send completed: %d results\n", len(results))
	}

	// Enhanced batch builder
	batch := hub.NewEnhancedBatch().
		WithDefaultTargetList(targetList).
		AddText("Daily Report", "System running normally").
		AddAlert("Critical Alert", "Database connection lost").
		AddText("Maintenance Notice", "Scheduled maintenance at 2 AM")

	batchResults, err := batch.SendAll(ctx)
	if err != nil {
		log.Printf("Enhanced batch send error: %v", err)
	} else {
		fmt.Printf("Enhanced batch completed: %d results\n", len(batchResults))
	}

	// ================================
	// Example 3: Enhanced Error Handling
	// ================================
	fmt.Println("\n=== Example 3: Enhanced Error Handling ===")

	// Create an invalid message to trigger validation errors
	invalidMessage := client.NewMessage().Title("").Body("").Build()
	_, err = hub.Send(ctx, invalidMessage, nil)
	if err != nil {
		if client.IsValidationError(err) {
			fmt.Printf("Validation error detected: %v\n", err)
			suggestions := client.GetErrorSuggestions(err)
			if len(suggestions) > 0 {
				fmt.Printf("Suggestions: %v\n", suggestions)
			}
		}
	}

	// Error collector example
	collector := client.NewErrorCollector()

	// Simulate multiple operations with errors
	collector.Add(client.NewValidationError("TEST", "Test error 1", "details", "suggestion"))
	collector.Add(client.NewNetworkError("NETWORK", "Test error 2", nil, true))

	if collector.HasErrors() {
		fmt.Printf("Error summary: %s\n", collector.Summary())
	}

	// ================================
	// Example 4: Streaming Results
	// ================================
	fmt.Println("\n=== Example 4: Streaming Results ===")

	// Create multiple messages for streaming
	messages := []*notifiers.Message{
		client.NewNotice("Stream Message 1", "First streaming message").Build(),
		client.NewNotice("Stream Message 2", "Second streaming message").Build(),
		client.NewNotice("Stream Message 3", "Third streaming message").Build(),
	}

	// Send with streaming
	streamOptions := &client.StreamingOptions{
		BufferSize:    50,
		BatchSize:     2,
		ConcurrentOps: 2,
	}

	stream := hub.SendBatchStream(ctx, messages, nil, streamOptions)

	// Collect results from stream
	streamCollector := client.NewStreamCollector()
	go streamCollector.Collect(stream)

	// Wait for completion
	stream.Wait()

	// Get summary
	summary := streamCollector.Summary()
	fmt.Printf("Stream summary: %s\n", summary)

	// ================================
	// Example 5: Template Management
	// ================================
	fmt.Println("\n=== Example 5: Template Management ===")

	// Register a template with metadata
	templateMetadata := &client.TemplateMetadata{
		Name:        "welcome",
		Description: "Welcome message template",
		Version:     "1.0.0",
		Author:      "NotifyHub",
		Tags:        []string{"welcome", "user", "onboarding"},
		Variables: []client.VariableInfo{
			{
				Name:        "username",
				Type:        "string",
				Required:    true,
				Description: "User's display name",
			},
			{
				Name:        "company",
				Type:        "string",
				Required:    false,
				DefaultValue: "NotifyHub",
				Description: "Company name",
			},
		},
		Platforms:    []string{"email", "feishu"},
		MessageTypes: []string{"notice", "welcome"},
	}

	templateContent := `TITLE: Welcome to {{.company}}, {{.username}}!
Hello {{.username}},

Welcome to {{.company}}! We're excited to have you on board.

Best regards,
The {{.company}} Team`

	err = hub.RegisterTemplate("welcome", templateContent, templateMetadata)
	if err != nil {
		log.Printf("Template registration error: %v", err)
	} else {
		fmt.Println("Template registered successfully")
	}

	// Use the template
	templateVars := map[string]interface{}{
		"username": "John Doe",
		"company":  "Acme Corp",
	}

	targetListForTemplate := client.NewTargetList().AddEmails("john.doe@example.com")

	templateResults, err := hub.SendWithTemplateManager(ctx, "welcome", templateVars,
		targetListForTemplate.Build(), nil)
	if err != nil {
		log.Printf("Template send error: %v", err)
	} else {
		fmt.Printf("Template message sent: %d results\n", len(templateResults))
	}

	// Get template statistics
	stats, err := hub.Templates().GetTemplateStats("welcome")
	if err != nil {
		log.Printf("Template stats error: %v", err)
	} else {
		fmt.Printf("Template stats: %+v\n", stats)
	}

	// ================================
	// Example 6: Middleware Usage
	// ================================
	fmt.Println("\n=== Example 6: Custom Middleware ===")

	// Add custom middleware
	hub.UseSendMiddlewareFunc("custom_logger", func(ctx context.Context, message *notifiers.Message, options *client.Options, next client.SendFunc) ([]*notifiers.SendResult, error) {
		fmt.Printf("🔧 Custom middleware: Processing message '%s'\n", message.Title)
		start := time.Now()
		results, err := next(ctx, message, options)
		fmt.Printf("🔧 Custom middleware: Completed in %v\n", time.Since(start))
		return results, err
	})

	// Send a message that will go through middleware
	middlewareMessage := client.NewNotice("Middleware Test", "Testing middleware functionality").
		Email("test@example.com").
		Build()

	_, err = hub.Send(ctx, middlewareMessage, nil)
	if err != nil {
		log.Printf("Middleware test error: %v", err)
	}

	// Show middleware chain info
	if middlewareChain := hub.GetMiddlewareChain(); middlewareChain != nil {
		fmt.Printf("Middleware chain: %d middlewares - %v\n",
			middlewareChain.Count(), middlewareChain.Names())
	}

	// ================================
	// Example 7: Configuration Validation
	// ================================
	fmt.Println("\n=== Example 7: Configuration Validation ===")

	// Run strict validation
	validationResult := hub.ValidateConfigurationStrict(ctx)
	fmt.Printf("Validation result: Valid=%t, Errors=%d, Warnings=%d\n",
		validationResult.Valid, len(validationResult.Errors), len(validationResult.Warnings))

	if len(validationResult.Warnings) > 0 {
		fmt.Println("Validation warnings:")
		for _, warning := range validationResult.Warnings {
			fmt.Printf("  - [%s] %s: %s\n", warning.Component, warning.Field, warning.Message)
		}
	}

	fmt.Println("\n🎉 All optimized examples completed successfully!")

	// Show final metrics
	metrics := hub.GetMetrics()
	if metrics != nil {
		fmt.Printf("\n📊 Final metrics: %+v\n", metrics)
	}
}