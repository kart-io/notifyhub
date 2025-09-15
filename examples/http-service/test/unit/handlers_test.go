package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/examples/http-service/internal/handlers"
	"github.com/kart-io/notifyhub/examples/http-service/internal/models"
	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/logger/adapters"
)

func TestNotificationHandler_SendNotification(t *testing.T) {
	// Setup
	hub, err := client.New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create hub: %v", err)
	}

	if err := hub.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start hub: %v", err)
	}
	defer hub.Stop()

	testLogger := adapters.NewStdLogAdapter(log.New(os.Stderr, "", log.LstdFlags), logger.Error)
	handler := handlers.NewNotificationHandler(hub, testLogger)

	tests := []struct {
		name           string
		payload        models.NotificationRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "valid notification",
			payload: models.NotificationRequest{
				Title: "Test Notification",
				Body:  "This is a test",
				Targets: []models.TargetRequest{
					{
						Type:  "email",
						Value: "test@example.com",
					},
				},
				Priority: 1,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "missing title",
			payload: models.NotificationRequest{
				Body: "This is a test",
				Targets: []models.TargetRequest{
					{
						Type:  "email",
						Value: "test@example.com",
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "missing body",
			payload: models.NotificationRequest{
				Title: "Test Notification",
				Targets: []models.TargetRequest{
					{
						Type:  "email",
						Value: "test@example.com",
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "missing targets",
			payload: models.NotificationRequest{
				Title:   "Test Notification",
				Body:    "This is a test",
				Targets: []models.TargetRequest{},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid priority",
			payload: models.NotificationRequest{
				Title: "Test Notification",
				Body:  "This is a test",
				Targets: []models.TargetRequest{
					{
						Type:  "email",
						Value: "test@example.com",
					},
				},
				Priority: 10, // Invalid priority
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handler.SendNotification(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check response
			var response interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}

			if tt.expectError {
				// Should be error response
				if errorResp, ok := response.(map[string]interface{}); ok {
					if _, hasError := errorResp["error"]; !hasError {
						t.Error("Expected error in response")
					}
				}
			} else {
				// Should be success response
				if successResp, ok := response.(map[string]interface{}); ok {
					if status, hasStatus := successResp["status"]; !hasStatus || status != "sent" {
						t.Error("Expected successful status in response")
					}
				}
			}
		})
	}
}

func TestNotificationHandler_SendBulkNotifications(t *testing.T) {
	// Setup
	hub, err := client.New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create hub: %v", err)
	}

	if err := hub.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start hub: %v", err)
	}
	defer hub.Stop()

	testLogger := adapters.NewStdLogAdapter(log.New(os.Stderr, "", log.LstdFlags), logger.Error)
	handler := handlers.NewNotificationHandler(hub, testLogger)

	tests := []struct {
		name           string
		payload        models.BulkNotificationRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "valid bulk notifications",
			payload: models.BulkNotificationRequest{
				Notifications: []models.NotificationRequest{
					{
						Title: "Test 1",
						Body:  "Body 1",
						Targets: []models.TargetRequest{
							{Type: "email", Value: "test1@example.com"},
						},
					},
					{
						Title: "Test 2",
						Body:  "Body 2",
						Targets: []models.TargetRequest{
							{Type: "email", Value: "test2@example.com"},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "empty notifications",
			payload: models.BulkNotificationRequest{
				Notifications: []models.NotificationRequest{},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "too many notifications",
			payload: models.BulkNotificationRequest{
				Notifications: make([]models.NotificationRequest, 101), // More than limit
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/bulk", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.SendBulkNotifications(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestNotificationHandler_GetHealth(t *testing.T) {
	// Setup
	hub, err := client.New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create hub: %v", err)
	}

	if err := hub.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start hub: %v", err)
	}
	defer hub.Stop()

	testLogger := adapters.NewStdLogAdapter(log.New(os.Stderr, "", log.LstdFlags), logger.Error)
	handler := handlers.NewNotificationHandler(hub, testLogger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.GetHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response models.HealthCheckResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal health response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", response.Status)
	}

	if response.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if response.Uptime <= 0 {
		t.Error("Expected positive uptime")
	}
}

func TestNotificationHandler_GetMetrics(t *testing.T) {
	// Setup
	hub, err := client.New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create hub: %v", err)
	}

	if err := hub.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start hub: %v", err)
	}
	defer hub.Stop()

	testLogger := adapters.NewStdLogAdapter(log.New(os.Stderr, "", log.LstdFlags), logger.Error)
	handler := handlers.NewNotificationHandler(hub, testLogger)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handler.GetMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response models.MetricsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal metrics response: %v", err)
	}

	if response.LastUpdated.IsZero() {
		t.Error("Expected last updated to be set")
	}
}

func TestNotificationHandler_SendTextNotification(t *testing.T) {
	// Setup
	hub, err := client.New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create hub: %v", err)
	}

	if err := hub.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start hub: %v", err)
	}
	defer hub.Stop()

	testLogger := adapters.NewStdLogAdapter(log.New(os.Stderr, "", log.LstdFlags), logger.Error)
	handler := handlers.NewNotificationHandler(hub, testLogger)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "valid text notification",
			queryParams:    "title=Test&body=Hello&target=test@example.com",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "missing title",
			queryParams:    "body=Hello&target=test@example.com",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "missing body",
			queryParams:    "title=Test&target=test@example.com",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "missing target",
			queryParams:    "title=Test&body=Hello",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/notifications/text"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()

			handler.SendTextNotification(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// Benchmark tests
func BenchmarkNotificationHandler_SendNotification(b *testing.B) {
	// Setup
	hub, err := client.New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}

	if err := hub.Start(context.Background()); err != nil {
		b.Fatalf("Failed to start hub: %v", err)
	}
	defer hub.Stop()

	testLogger := adapters.NewStdLogAdapter(log.New(os.Stderr, "", log.LstdFlags), logger.Error)
	handler := handlers.NewNotificationHandler(hub, testLogger)

	payload := models.NotificationRequest{
		Title: "Benchmark Test",
		Body:  "This is a benchmark test",
		Targets: []models.TargetRequest{
			{
				Type:  "email",
				Value: "benchmark@example.com",
			},
		},
		Priority: 1,
	}

	payloadBytes, _ := json.Marshal(payload)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications",
				bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.SendNotification(rr, req)

			if rr.Code != http.StatusOK {
				b.Errorf("Expected status 200, got %d", rr.Code)
			}
		}
	})
}

func BenchmarkNotificationHandler_GetHealth(b *testing.B) {
	// Setup
	hub, err := client.New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}

	if err := hub.Start(context.Background()); err != nil {
		b.Fatalf("Failed to start hub: %v", err)
	}
	defer hub.Stop()

	testLogger := adapters.NewStdLogAdapter(log.New(os.Stderr, "", log.LstdFlags), logger.Error)
	handler := handlers.NewNotificationHandler(hub, testLogger)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rr := httptest.NewRecorder()
			handler.GetHealth(rr, req)

			if rr.Code != http.StatusOK {
				b.Errorf("Expected status 200, got %d", rr.Code)
			}
		}
	})
}