package config

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cfg := New()
	if cfg == nil {
		t.Fatal("Expected config to be created, got nil")
	}
}

func TestWithDefaults(t *testing.T) {
	cfg := New(WithDefaults())
	if cfg == nil {
		t.Fatal("Expected config to be created with defaults, got nil")
	}
}

func TestWithMockNotifier(t *testing.T) {
	cfg := New(WithMockNotifier("test"))
	if cfg == nil {
		t.Fatal("Expected config to be created with mock notifier, got nil")
	}

	if cfg.mockNotifier == nil {
		t.Fatal("Expected mock notifier to be configured")
	}

	if cfg.mockNotifier.Name != "test" {
		t.Errorf("Expected mock notifier name to be 'test', got %s", cfg.mockNotifier.Name)
	}
}

func TestWithQueue(t *testing.T) {
	cfg := New(WithQueue("memory", 100, 4))
	if cfg == nil {
		t.Fatal("Expected config to be created with queue, got nil")
	}

	if cfg.queue == nil {
		t.Fatal("Expected queue to be configured")
	}

	if cfg.queue.Type != "memory" {
		t.Errorf("Expected queue type to be 'memory', got %s", cfg.queue.Type)
	}

	if cfg.queue.BufferSize != 100 {
		t.Errorf("Expected buffer size to be 100, got %d", cfg.queue.BufferSize)
	}

	if cfg.queue.Workers != 4 {
		t.Errorf("Expected workers to be 4, got %d", cfg.queue.Workers)
	}
}

func TestWithFeishu(t *testing.T) {
	cfg := New(WithFeishu("https://example.com/webhook", "secret"))
	if cfg == nil {
		t.Fatal("Expected config to be created with Feishu, got nil")
	}

	if cfg.feishu == nil {
		t.Fatal("Expected Feishu to be configured")
	}

	if cfg.feishu.WebhookURL != "https://example.com/webhook" {
		t.Errorf("Expected webhook URL to be 'https://example.com/webhook', got %s", cfg.feishu.WebhookURL)
	}

	if cfg.feishu.Secret != "secret" {
		t.Errorf("Expected secret to be 'secret', got %s", cfg.feishu.Secret)
	}
}

func TestWithEmail(t *testing.T) {
	cfg := New(WithEmail("smtp.example.com", 587, "user", "pass", "from@example.com"))
	if cfg == nil {
		t.Fatal("Expected config to be created with Email, got nil")
	}

	if cfg.email == nil {
		t.Fatal("Expected Email to be configured")
	}

	if cfg.email.Host != "smtp.example.com" {
		t.Errorf("Expected SMTP host to be 'smtp.example.com', got %s", cfg.email.Host)
	}

	if cfg.email.Port != 587 {
		t.Errorf("Expected SMTP port to be 587, got %d", cfg.email.Port)
	}
}

func TestWithSilentLogger(t *testing.T) {
	cfg := New(WithSilentLogger())
	if cfg == nil {
		t.Fatal("Expected config to be created with silent logger, got nil")
	}
}

func TestWithTestDefaults(t *testing.T) {
	cfg := New(WithTestDefaults())
	if cfg == nil {
		t.Fatal("Expected config to be created with test defaults, got nil")
	}
}

func TestWithMockNotifierDelay(t *testing.T) {
	delay := 100 * time.Millisecond
	cfg := New(
		WithMockNotifier("test"),
		WithMockNotifierDelay(delay),
	)

	if cfg.mockNotifier == nil {
		t.Fatal("Expected mock notifier to be configured")
	}

	if cfg.mockNotifier.Delay != delay {
		t.Errorf("Expected delay to be %v, got %v", delay, cfg.mockNotifier.Delay)
	}
}

func TestWithMockNotifierFailure(t *testing.T) {
	cfg := New(
		WithMockNotifier("test"),
		WithMockNotifierFailure(),
	)

	if cfg.mockNotifier == nil {
		t.Fatal("Expected mock notifier to be configured")
	}

	if !cfg.mockNotifier.ShouldFail {
		t.Error("Expected mock notifier to be configured to fail")
	}
}

func TestMultipleOptions(t *testing.T) {
	cfg := New(
		WithMockNotifier("test"),
		WithQueue("memory", 200, 8),
		WithSilentLogger(),
	)

	if cfg == nil {
		t.Fatal("Expected config to be created with multiple options, got nil")
	}

	if cfg.mockNotifier == nil {
		t.Fatal("Expected mock notifier to be configured")
	}

	if cfg.queue == nil {
		t.Fatal("Expected queue to be configured")
	}

	if cfg.mockNotifier.Name != "test" {
		t.Errorf("Expected mock notifier name to be 'test', got %s", cfg.mockNotifier.Name)
	}

	if cfg.queue.BufferSize != 200 {
		t.Errorf("Expected buffer size to be 200, got %d", cfg.queue.BufferSize)
	}
}
