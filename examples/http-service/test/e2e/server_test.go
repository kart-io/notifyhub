package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/examples/http-service/internal/models"
)

const (
	testServerURL = "http://localhost:8081"
	testAPIKey    = "test-api-key-12345"
)

var serverCmd *exec.Cmd

// TestMain sets up and tears down the test server
func TestMain(m *testing.M) {
	// Start test server
	if err := startTestServer(); err != nil {
		fmt.Printf("Failed to start test server: %v\n", err)
		os.Exit(1)
	}

	// Wait for server to be ready
	if err := waitForServer(testServerURL + "/health"); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		stopTestServer()
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Stop test server
	stopTestServer()

	os.Exit(code)
}

func startTestServer() error {
	// Set environment variables for test
	os.Setenv("PORT", "8081")
	os.Setenv("API_KEY", testAPIKey)
	os.Setenv("LOG_LEVEL", "error") // Reduce noise during tests
	os.Setenv("RATE_LIMIT_PER_MINUTE", "1000") // High limit for tests

	// Build and start server
	serverCmd = exec.Command("go", "run", "./cmd/server.go")
	serverCmd.Dir = "../../" // From test/e2e to project root

	// Capture output for debugging
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	return serverCmd.Start()
}

func stopTestServer() {
	if serverCmd != nil && serverCmd.Process != nil {
		serverCmd.Process.Signal(os.Interrupt)
		serverCmd.Wait()
	}
}

func waitForServer(url string) error {
	client := &http.Client{Timeout: 1 * time.Second}

	for i := 0; i < 30; i++ { // Wait up to 30 seconds
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("server failed to start within 30 seconds")
}

func TestE2E_HealthCheck(t *testing.T) {
	resp, err := http.Get(testServerURL + "/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health models.HealthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Errorf("Failed to decode health response: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}

	if health.Uptime <= 0 {
		t.Error("Expected positive uptime")
	}
}

func TestE2E_Metrics(t *testing.T) {
	resp, err := http.Get(testServerURL + "/metrics")
	if err != nil {
		t.Fatalf("Failed to call metrics endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var metrics models.MetricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		t.Errorf("Failed to decode metrics response: %v", err)
	}

	if metrics.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be set")
	}
}

func TestE2E_SendNotification_WithAuth(t *testing.T) {
	payload := models.NotificationRequest{
		Title: "E2E Test Notification",
		Body:  "This is an end-to-end test",
		Targets: []models.TargetRequest{
			{
				Type:  "email",
				Value: "e2e-test@example.com",
			},
		},
		Priority: 1,
	}

	jsonPayload, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, testServerURL+"/api/v1/notifications", bytes.NewBuffer(jsonPayload))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var response models.NotificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Status != "sent" {
		t.Errorf("Expected status 'sent', got %s", response.Status)
	}

	if response.ID == "" {
		t.Error("Expected response ID to be set")
	}
}

func TestE2E_SendNotification_WithoutAuth(t *testing.T) {
	payload := models.NotificationRequest{
		Title: "Unauthorized Test",
		Body:  "This should fail",
		Targets: []models.TargetRequest{
			{
				Type:  "email",
				Value: "unauthorized@example.com",
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(testServerURL+"/api/v1/notifications", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestE2E_SendBulkNotifications(t *testing.T) {
	payload := models.BulkNotificationRequest{
		Notifications: []models.NotificationRequest{
			{
				Title: "Bulk Test 1",
				Body:  "First bulk message",
				Targets: []models.TargetRequest{
					{Type: "email", Value: "bulk1@example.com"},
				},
			},
			{
				Title: "Bulk Test 2",
				Body:  "Second bulk message",
				Targets: []models.TargetRequest{
					{Type: "email", Value: "bulk2@example.com"},
				},
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodPost, testServerURL+"/api/v1/notifications/bulk", bytes.NewBuffer(jsonPayload))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send bulk notifications: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	total, ok := response["total"].(float64)
	if !ok || total != 2 {
		t.Errorf("Expected total 2, got %v", response["total"])
	}
}

func TestE2E_SendTextNotification(t *testing.T) {
	url := fmt.Sprintf("%s/api/v1/notifications/text?title=Quick%%20Test&body=Quick%%20message&target=text@example.com", testServerURL)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+testAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send text notification: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var response models.NotificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Status != "sent" {
		t.Errorf("Expected status 'sent', got %s", response.Status)
	}
}

func TestE2E_RateLimit(t *testing.T) {
	// This test would require lowering the rate limit for testing
	// For now, we'll just test that the endpoint responds correctly
	t.Skip("Rate limit testing requires special configuration")
}

func TestE2E_InvalidJSON(t *testing.T) {
	invalidJSON := `{"title": "Test", "body": "Test", "targets": [`

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, testServerURL+"/api/v1/notifications", bytes.NewBufferString(invalidJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestE2E_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		payload models.NotificationRequest
	}{
		{
			name: "missing title",
			payload: models.NotificationRequest{
				Body: "Test body",
				Targets: []models.TargetRequest{
					{Type: "email", Value: "test@example.com"},
				},
			},
		},
		{
			name: "missing body",
			payload: models.NotificationRequest{
				Title: "Test title",
				Targets: []models.TargetRequest{
					{Type: "email", Value: "test@example.com"},
				},
			},
		},
		{
			name: "missing targets",
			payload: models.NotificationRequest{
				Title:   "Test title",
				Body:    "Test body",
				Targets: []models.TargetRequest{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPayload, _ := json.Marshal(tt.payload)

			client := &http.Client{Timeout: 30 * time.Second}
			req, err := http.NewRequest(http.MethodPost, testServerURL+"/api/v1/notifications", bytes.NewBuffer(jsonPayload))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+testAPIKey)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("Expected status 400 for %s, got %d", tt.name, resp.StatusCode)
			}
		})
	}
}

func TestE2E_ContentTypeValidation(t *testing.T) {
	payload := `{"title": "Test", "body": "Test", "targets": [{"type": "email", "value": "test@example.com"}]}`

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, testServerURL+"/api/v1/notifications", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "text/plain") // Wrong content type
	req.Header.Set("Authorization", "Bearer "+testAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("Expected status 415, got %d", resp.StatusCode)
	}
}

func TestE2E_CORSHeaders(t *testing.T) {
	resp, err := http.Get(testServerURL + "/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	for header, expected := range expectedHeaders {
		if actual := resp.Header.Get(header); actual != expected {
			t.Errorf("Expected %s to be %s, got %s", header, expected, actual)
		}
	}
}

func TestE2E_SecurityHeaders(t *testing.T) {
	resp, err := http.Get(testServerURL + "/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":   "nosniff",
		"X-Frame-Options":          "DENY",
		"X-XSS-Protection":         "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}

	for header, expected := range expectedHeaders {
		if actual := resp.Header.Get(header); actual != expected {
			t.Errorf("Expected %s to be %s, got %s", header, expected, actual)
		}
	}
}