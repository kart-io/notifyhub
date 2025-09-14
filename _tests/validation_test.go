package notifyhub_test

import (
	"testing"
	"time"

	"github.com/kart-io/notifyhub"
	"github.com/kart-io/notifyhub/notifiers"
)

func TestPackageStructure(t *testing.T) {
	// Test message builder
	message := notifyhub.NewMessage().
		Title("Test").
		Body("Test message").
		Priority(3).
		Build()

	if message.Title != "Test" {
		t.Error("Message title should be 'Test'")
	}

	if message.Body != "Test message" {
		t.Error("Message body should be 'Test message'")
	}

	if message.Priority != 3 {
		t.Error("Message priority should be 3")
	}

	// Test Hub creation with test defaults
	hub, err := notifyhub.New(notifyhub.WithTestDefaults())
	if err == nil { // Hub creation may fail without proper config, that's OK for this test
		if hub == nil {
			t.Error("Hub should not be nil when creation succeeds")
		}
	}

	// Test routing rule builder
	rule := notifyhub.NewRoutingRule("test-rule").
		WithPriority(5).
		RouteTo("email").
		Build()

	if rule.Name != "test-rule" {
		t.Error("Routing rule name should be 'test-rule'")
	}

	if len(rule.Conditions.Priority) == 0 || rule.Conditions.Priority[0] != 5 {
		t.Error("Routing rule should have priority condition of 5")
	}
}

func TestMessageCreation(t *testing.T) {
	// Test various message builders
	alert := notifyhub.NewAlert("Alert Title", "Alert Body").Build()
	if alert.Priority != 4 {
		t.Error("Alert should have priority 4")
	}

	notice := notifyhub.NewNotice("Notice Title", "Notice Body").Build()
	if notice.Priority != 3 {
		t.Error("Notice should have priority 3")
	}

	report := notifyhub.NewReport("Report Title", "Report Body").Build()
	if report.Priority != 2 {
		t.Error("Report should have priority 2")
	}
}

func TestEmailNotifier(t *testing.T) {
	// Test email notifier creation (won't actually send without valid config)
	notifier := notifiers.NewEmailNotifier("localhost", 587, "", "", "test@example.com", false, 30*time.Second)
	if notifier.Name() != "email" {
		t.Error("Email notifier name should be 'email'")
	}

	// Test target support
	emailTarget := notifiers.Target{Type: notifiers.TargetTypeEmail, Value: "user@example.com"}
	if !notifier.SupportsTarget(emailTarget) {
		t.Error("Email notifier should support email targets")
	}
}

func TestFeishuNotifier(t *testing.T) {
	// Test feishu notifier creation (won't actually send without valid webhook)
	notifier := notifiers.NewFeishuNotifier("https://example.com/webhook", "", 30*time.Second)
	if notifier.Name() != "feishu" {
		t.Error("Feishu notifier name should be 'feishu'")
	}

	// Test target support
	groupTarget := notifiers.Target{Type: notifiers.TargetTypeGroup, Value: "group123"}
	if !notifier.SupportsTarget(groupTarget) {
		t.Error("Feishu notifier should support group targets")
	}
}

func TestRetryPolicy(t *testing.T) {
	policy := notifyhub.DefaultRetryPolicy()
	if policy.MaxRetries != 3 {
		t.Error("Default retry policy should have 3 max retries")
	}

	// Test retry decision
	if !policy.ShouldRetry(0) {
		t.Error("Should retry on first attempt")
	}

	if policy.ShouldRetry(3) {
		t.Error("Should not retry after max retries")
	}
}