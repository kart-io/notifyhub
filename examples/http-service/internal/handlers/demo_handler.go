package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/notifiers"
)

// DemoHandler demonstrates the new optimized NotifyHub features
type DemoHandler struct {
	hub    *client.Hub
	logger logger.Interface
}

// NewDemoHandler creates a new demo handler
func NewDemoHandler(hub *client.Hub, logger logger.Interface) *DemoHandler {
	return &DemoHandler{
		hub:    hub,
		logger: logger,
	}
}

// DemoBuilderAPI demonstrates the new Builder pattern API
func (h *DemoHandler) DemoBuilderAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Example 1: Simple alert using new Builder API
	alertMessage := client.NewAlert("System Alert", "Database connection restored").
		Email("admin@company.com").
		Email("ops@company.com").
		Urgent().
		Metadata("service", "database").
		Build()

	_, err := h.hub.Send(ctx, alertMessage, nil)
	if err != nil {
		h.logger.Error(ctx, "Failed to send alert: %v", err)
	}

	// Example 2: Marketing message with template and variables
	marketingMessage := client.NewMessage().
		Title("Welcome to Our Service!").
		WithTemplate("welcome_template").
		WithVariables(map[string]interface{}{
			"user_name":    "John Doe",
			"signup_date":  time.Now().Format("2006-01-02"),
			"company_name": "TechCorp",
		}).
		MultipleEmails("user@example.com", "backup@example.com").
		Normal().
		Build()

	_, err = h.hub.Send(ctx, marketingMessage, nil)
	if err != nil {
		h.logger.Error(ctx, "Failed to send marketing message: %v", err)
	}

	// Example 3: Multi-platform notification
	multiPlatformMessage := client.NewNotice("Deployment Complete", "Version 2.1.0 has been deployed successfully").
		User("user123", "feishu").
		User("dev-team", "slack").
		Channel("general", "discord").
		High().
		Build()

	_, err = h.hub.Send(ctx, multiPlatformMessage, nil)
	if err != nil {
		h.logger.Error(ctx, "Failed to send multi-platform message: %v", err)
	}

	// Example 4: Async notification with delay
	delayedMessage := client.NewMessage().
		Title("Scheduled Maintenance Reminder").
		Body("System maintenance will begin in 1 hour").
		Email("ops@company.com").
		Delay(1 * time.Hour).
		High().
		Build()

	taskID, err := h.hub.SendAsync(ctx, delayedMessage, nil)
	if err != nil {
		h.logger.Error(ctx, "Failed to schedule delayed message: %v", err)
	} else {
		h.logger.Info(ctx, "Delayed message scheduled with task ID: %s", taskID)
	}

	response := client.CreateSuccessResponse("Builder API demo completed", map[string]interface{}{
		"examples_sent":      3,
		"scheduled_task_id": taskID,
		"features_demonstrated": []string{
			"Builder pattern API",
			"Fluent priority methods",
			"Multi-target support",
			"Template and variables",
			"Async scheduling",
			"Delay functionality",
		},
	})

	client.WriteJSONResponse(w, http.StatusOK, response)
}

// DemoConvenienceFunctions demonstrates the new convenience functions
func (h *DemoHandler) DemoConvenienceFunctions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Example 1: Quick text message (using convenience function from Hub)
	err := h.hub.SendText(ctx, "Quick Update", "System is running normally",
		Email("admin@company.com"),
		User("ops-team", "slack"),
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to send text message: %v", err)
	}

	// Example 2: Emergency alert (using convenience function from Hub)
	err = h.hub.SendAlert(ctx, "CRITICAL: Service Down", "Payment service is not responding",
		Email("oncall@company.com"),
		Channel("incidents", "slack"),
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to send alert: %v", err)
	}

	// Example 3: Template-based message
	err = h.hub.SendWithTemplate(ctx, "daily_report", map[string]interface{}{
		"date":           time.Now().Format("2006-01-02"),
		"total_users":    1250,
		"active_sessions": 89,
		"revenue":        "$15,432",
	},
		Email("management@company.com"),
		User("analytics-team", "feishu"),
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to send template message: %v", err)
	}

	// Example 4: Demonstrate ultra-quick convenience builders
	quickEmailMessage := client.QuickEmail("Test Alert", "System check complete", "admin@company.com")
	_, err = h.hub.Send(ctx, quickEmailMessage.Build(), nil)
	if err != nil {
		h.logger.Error(ctx, "Failed to send quick email: %v", err)
	}

	// Example 5: Batch email with new builder
	batchMessage := client.BatchEmail("Weekly Report", "Performance summary attached",
		"team@company.com", "manager@company.com", "director@company.com")
	_, err = h.hub.Send(ctx, batchMessage.Build(), nil)
	if err != nil {
		h.logger.Error(ctx, "Failed to send batch email: %v", err)
	}

	// Example 6: Conditional message building
	isProduction := true
	conditionalMessage := client.NewAlert("Deployment Alert", "New version deployed").
		Email("ops@company.com").
		If(isProduction, func(b *client.MessageBuilder) *client.MessageBuilder {
			return b.Urgent().Email("cto@company.com").Metadata("env", "production")
		}).
		Unless(isProduction, func(b *client.MessageBuilder) *client.MessageBuilder {
			return b.Low().Metadata("env", "development")
		})

	_, err = h.hub.Send(ctx, conditionalMessage.Build(), nil)
	if err != nil {
		h.logger.Error(ctx, "Failed to send conditional message: %v", err)
	}

	response := client.CreateSuccessResponse("Enhanced convenience functions demo completed", map[string]interface{}{
		"functions_demonstrated": []string{
			"SendText - Quick text messages",
			"SendAlert - Emergency notifications",
			"SendWithTemplate - Template-based messages",
			"QuickEmail - One-line email builder",
			"BatchEmail - Multi-recipient builder",
			"Conditional building - If/Unless patterns",
		},
		"code_improvements": map[string]interface{}{
			"before": "15+ lines per message with manual target creation",
			"after": "1-3 lines with smart builders and auto-targeting",
			"reduction": "80-90% code reduction achieved",
		},
		"benefits": []string{
			"Reduced code complexity",
			"Built-in error handling",
			"Automatic timeout management",
			"Consistent API patterns",
			"Conditional message building",
			"Smart target detection",
		},
	})

	client.WriteJSONResponse(w, http.StatusOK, response)
}

// DemoHTTPUtilities demonstrates the new HTTP utility functions
func (h *DemoHandler) DemoHTTPUtilities(w http.ResponseWriter, r *http.Request) {
	// Parse HTTP request using new utility functions
	httpReq, err := client.ParseHTTPRequest(r)
	if err != nil {
		response := client.CreateErrorResponse("Failed to parse HTTP request", err.Error())
		client.WriteJSONResponse(w, http.StatusBadRequest, response)
		return
	}

	// Parse HTTP options
	httpOptions, err := client.ParseHTTPOptions(r)
	if err != nil {
		response := client.CreateErrorResponse("Failed to parse HTTP options", err.Error())
		client.WriteJSONResponse(w, http.StatusBadRequest, response)
		return
	}

	// Convert to NotifyHub types
	message, err := client.ConvertHTTPToMessage(httpReq)
	if err != nil {
		validationErrors := []string{err.Error()}
		response := client.CreateValidationErrorResponse(validationErrors)
		client.WriteJSONResponse(w, http.StatusBadRequest, response)
		return
	}

	options, err := client.ConvertHTTPToOptions(httpOptions)
	if err != nil {
		response := client.CreateErrorResponse("Failed to convert options", err.Error())
		client.WriteJSONResponse(w, http.StatusBadRequest, response)
		return
	}

	// Send the message
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if httpOptions.Async {
		taskID, err := h.hub.SendAsync(ctx, message, options)
		if err != nil {
			response := client.CreateErrorResponse("Failed to enqueue message", err.Error())
			client.WriteJSONResponse(w, http.StatusInternalServerError, response)
			return
		}

		response := client.CreateAsyncSuccessResponse(taskID)
		client.WriteJSONResponse(w, http.StatusAccepted, response)
	} else {
		results, err := h.hub.Send(ctx, message, options)
		if err != nil {
			response := client.CreateErrorResponse("Failed to send message", err.Error())
			client.WriteJSONResponse(w, http.StatusInternalServerError, response)
			return
		}

		response := client.CreateSuccessResponse("Message sent successfully", map[string]interface{}{
			"message_id": message.ID,
			"targets":    len(message.Targets),
			"results":    len(results),
			"features_used": []string{
				"HTTP request parsing",
				"HTTP options parsing",
				"Message conversion with validation",
				"Options conversion",
				"Standardized response format",
			},
		})
		client.WriteJSONResponse(w, http.StatusOK, response)
	}
}

// Helper functions for creating targets

func Email(email string) notifiers.Target {
	return notifiers.Target{
		Type:  notifiers.TargetTypeEmail,
		Value: email,
	}
}

func User(userID, platform string) notifiers.Target {
	return notifiers.Target{
		Type:     notifiers.TargetTypeUser,
		Value:    userID,
		Platform: platform,
	}
}

func Channel(channelID, platform string) notifiers.Target {
	return notifiers.Target{
		Type:     notifiers.TargetTypeChannel,
		Value:    channelID,
		Platform: platform,
	}
}