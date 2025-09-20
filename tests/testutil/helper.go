package testutil

import (
	"time"

	"github.com/kart-io/notifyhub/api"
	"github.com/kart-io/notifyhub/tests/mocks"
)

// RegisterMockTransports registers mock transports for common platforms
// This is a convenience function for tests that need mock transports
func RegisterMockTransports(hub *api.Client, delay time.Duration) map[string]*mocks.MockTransport {
	platforms := []string{"test", "email", "feishu", "slack", "sms"}
	mockTransports := make(map[string]*mocks.MockTransport)

	for _, platform := range platforms {
		mockTransport := mocks.NewMockTransport(platform)
		if delay > 0 {
			mockTransport.SetDelay(delay)
		}
		hub.RegisterTransport(mockTransport)
		mockTransports[platform] = mockTransport
	}

	return mockTransports
}

// RegisterMockTransport registers a single mock transport
func RegisterMockTransport(hub *api.Client, platform string, delay time.Duration) *mocks.MockTransport {
	mockTransport := mocks.NewMockTransport(platform)
	if delay > 0 {
		mockTransport.SetDelay(delay)
	}
	hub.RegisterTransport(mockTransport)
	return mockTransport
}
