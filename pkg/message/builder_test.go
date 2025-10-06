package message

import (
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/target"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	if builder == nil {
		t.Fatal("NewBuilder() returned nil")
	}
	if builder.message == nil {
		t.Fatal("Builder message is nil")
	}
	if builder.message.ID == "" {
		t.Error("Message ID should be generated")
	}
	if builder.message.Format != FormatText {
		t.Errorf("Default format = %v, want %v", builder.message.Format, FormatText)
	}
	if builder.message.Priority != PriorityNormal {
		t.Errorf("Default priority = %v, want %v", builder.message.Priority, PriorityNormal)
	}
	if builder.message.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if builder.message.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
	if builder.message.Variables == nil {
		t.Error("Variables should be initialized")
	}
	if builder.message.PlatformData == nil {
		t.Error("PlatformData should be initialized")
	}
}

func TestNewMessage(t *testing.T) {
	builder := NewMessage()
	if builder == nil {
		t.Fatal("NewMessage() returned nil")
	}
	if builder.message == nil {
		t.Fatal("Builder message is nil")
	}
}

func TestBuilder_SetID(t *testing.T) {
	builder := NewBuilder()
	id := "test-id-123"
	result := builder.SetID(id)

	if result != builder {
		t.Error("SetID should return builder for chaining")
	}
	if builder.message.ID != id {
		t.Errorf("ID = %v, want %v", builder.message.ID, id)
	}
}

func TestBuilder_SetTitle(t *testing.T) {
	builder := NewBuilder()
	title := "Test Title"
	result := builder.SetTitle(title)

	if result != builder {
		t.Error("SetTitle should return builder for chaining")
	}
	if builder.message.Title != title {
		t.Errorf("Title = %v, want %v", builder.message.Title, title)
	}
}

func TestBuilder_SetBody(t *testing.T) {
	builder := NewBuilder()
	body := "Test Body"
	result := builder.SetBody(body)

	if result != builder {
		t.Error("SetBody should return builder for chaining")
	}
	if builder.message.Body != body {
		t.Errorf("Body = %v, want %v", builder.message.Body, body)
	}
}

func TestBuilder_SetFormat(t *testing.T) {
	builder := NewBuilder()
	format := FormatHTML
	result := builder.SetFormat(format)

	if result != builder {
		t.Error("SetFormat should return builder for chaining")
	}
	if builder.message.Format != format {
		t.Errorf("Format = %v, want %v", builder.message.Format, format)
	}
}

func TestBuilder_SetPriority(t *testing.T) {
	builder := NewBuilder()
	priority := PriorityHigh
	result := builder.SetPriority(priority)

	if result != builder {
		t.Error("SetPriority should return builder for chaining")
	}
	if builder.message.Priority != priority {
		t.Errorf("Priority = %v, want %v", builder.message.Priority, priority)
	}
}

func TestBuilder_AddTarget(t *testing.T) {
	builder := NewBuilder()
	tgt := target.NewEmail("test@example.com")
	result := builder.AddTarget(tgt)

	if result != builder {
		t.Error("AddTarget should return builder for chaining")
	}
	if len(builder.message.Targets) != 1 {
		t.Errorf("Targets length = %v, want 1", len(builder.message.Targets))
	}
	if builder.message.Targets[0] != tgt {
		t.Error("Target not added correctly")
	}
}

func TestBuilder_AddTargets(t *testing.T) {
	builder := NewBuilder()
	targets := []target.Target{
		target.NewEmail("test1@example.com"),
		target.NewEmail("test2@example.com"),
	}
	result := builder.AddTargets(targets)

	if result != builder {
		t.Error("AddTargets should return builder for chaining")
	}
	if len(builder.message.Targets) != 2 {
		t.Errorf("Targets length = %v, want 2", len(builder.message.Targets))
	}
}

func TestBuilder_SetTargets(t *testing.T) {
	builder := NewBuilder()
	// Add initial target
	builder.AddTarget(target.NewEmail("old@example.com"))

	// Set new targets (should replace)
	targets := []target.Target{
		target.NewEmail("new1@example.com"),
		target.NewEmail("new2@example.com"),
	}
	result := builder.SetTargets(targets)

	if result != builder {
		t.Error("SetTargets should return builder for chaining")
	}
	if len(builder.message.Targets) != 2 {
		t.Errorf("Targets length = %v, want 2", len(builder.message.Targets))
	}
}

func TestBuilder_AddMetadata(t *testing.T) {
	builder := NewBuilder()
	key := "test_key"
	value := "test_value"
	result := builder.AddMetadata(key, value)

	if result != builder {
		t.Error("AddMetadata should return builder for chaining")
	}
	if builder.message.Metadata[key] != value {
		t.Errorf("Metadata[%s] = %v, want %v", key, builder.message.Metadata[key], value)
	}
}

func TestBuilder_SetMetadata(t *testing.T) {
	builder := NewBuilder()
	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}
	result := builder.SetMetadata(metadata)

	if result != builder {
		t.Error("SetMetadata should return builder for chaining")
	}
	if len(builder.message.Metadata) != 2 {
		t.Errorf("Metadata length = %d, want 2", len(builder.message.Metadata))
	}
	if builder.message.Metadata["key1"] != "value1" {
		t.Error("Metadata key1 not set correctly")
	}
}

func TestBuilder_AddVariable(t *testing.T) {
	builder := NewBuilder()
	key := "username"
	value := "john"
	result := builder.AddVariable(key, value)

	if result != builder {
		t.Error("AddVariable should return builder for chaining")
	}
	if builder.message.Variables[key] != value {
		t.Errorf("Variables[%s] = %v, want %v", key, builder.message.Variables[key], value)
	}
}

func TestBuilder_SetVariables(t *testing.T) {
	builder := NewBuilder()
	variables := map[string]interface{}{
		"name": "John",
		"age":  30,
	}
	result := builder.SetVariables(variables)

	if result != builder {
		t.Error("SetVariables should return builder for chaining")
	}
	if len(builder.message.Variables) != 2 {
		t.Errorf("Variables length = %d, want 2", len(builder.message.Variables))
	}
}

func TestBuilder_ScheduleAt(t *testing.T) {
	builder := NewBuilder()
	scheduled := time.Now().Add(1 * time.Hour)
	result := builder.ScheduleAt(scheduled)

	if result != builder {
		t.Error("ScheduleAt should return builder for chaining")
	}
	if builder.message.ScheduledAt == nil {
		t.Fatal("ScheduledAt should be set")
	}
	if !builder.message.ScheduledAt.Equal(scheduled) {
		t.Errorf("ScheduledAt = %v, want %v", builder.message.ScheduledAt, scheduled)
	}
}

func TestBuilder_AddPlatformData(t *testing.T) {
	builder := NewBuilder()
	platform := "email"
	data := map[string]interface{}{"subject": "Test"}
	result := builder.AddPlatformData(platform, data)

	if result != builder {
		t.Error("AddPlatformData should return builder for chaining")
	}
	if builder.message.PlatformData[platform] == nil {
		t.Error("PlatformData not set")
	}
}

func TestBuilder_SetPlatformData(t *testing.T) {
	builder := NewBuilder()
	platformData := map[string]interface{}{
		"email": map[string]string{"subject": "Test"},
	}
	result := builder.SetPlatformData(platformData)

	if result != builder {
		t.Error("SetPlatformData should return builder for chaining")
	}
	if len(builder.message.PlatformData) != 1 {
		t.Errorf("PlatformData length = %d, want 1", len(builder.message.PlatformData))
	}
}

func TestBuilder_Build(t *testing.T) {
	builder := NewBuilder()
	builder.SetTitle("Test").SetBody("Body")

	msg := builder.Build()
	if msg == nil {
		t.Fatal("Build() returned nil")
	}
	if msg.Title != "Test" {
		t.Errorf("Title = %v, want Test", msg.Title)
	}
	if msg.Body != "Body" {
		t.Errorf("Body = %v, want Body", msg.Body)
	}
}

func TestBuilder_Chaining(t *testing.T) {
	// Test method chaining
	msg := NewBuilder().
		SetID("chain-test").
		SetTitle("Chain Title").
		SetBody("Chain Body").
		SetFormat(FormatHTML).
		SetPriority(PriorityHigh).
		AddTarget(target.NewEmail("chain@example.com")).
		AddMetadata("key", "value").
		AddVariable("var", "val").
		Build()

	if msg.ID != "chain-test" {
		t.Errorf("ID = %v, want chain-test", msg.ID)
	}
	if msg.Title != "Chain Title" {
		t.Errorf("Title = %v, want Chain Title", msg.Title)
	}
	if msg.Body != "Chain Body" {
		t.Errorf("Body = %v, want Chain Body", msg.Body)
	}
	if msg.Format != FormatHTML {
		t.Errorf("Format = %v, want %v", msg.Format, FormatHTML)
	}
	if msg.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want %v", msg.Priority, PriorityHigh)
	}
	if len(msg.Targets) != 1 {
		t.Errorf("Targets length = %v, want 1", len(msg.Targets))
	}
	if msg.Metadata["key"] != "value" {
		t.Error("Metadata not set correctly")
	}
	if msg.Variables["var"] != "val" {
		t.Error("Variables not set correctly")
	}
}
