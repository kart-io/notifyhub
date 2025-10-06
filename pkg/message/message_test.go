package message

import (
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/target"
)

func TestMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		msg     *Message
		wantErr bool
	}{
		{
			name: "valid message with title",
			msg: &Message{
				ID:    "msg-123",
				Title: "Test Title",
				Targets: []target.Target{
					target.NewEmail("test@example.com"),
				},
			},
			wantErr: false,
		},
		{
			name: "valid message with body",
			msg: &Message{
				ID:   "msg-123",
				Body: "Test body content",
				Targets: []target.Target{
					target.NewEmail("test@example.com"),
				},
			},
			wantErr: false,
		},
		{
			name: "valid message with both",
			msg: &Message{
				ID:    "msg-123",
				Title: "Test Title",
				Body:  "Test body",
				Targets: []target.Target{
					target.NewEmail("test@example.com"),
				},
			},
			wantErr: false,
		},
		{
			name: "missing title and body",
			msg: &Message{
				ID: "msg-123",
				Targets: []target.Target{
					target.NewEmail("test@example.com"),
				},
			},
			wantErr: true,
		},
		{
			name: "missing targets",
			msg: &Message{
				ID:    "msg-123",
				Title: "Test",
				Body:  "Test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	msg := New()

	if msg.ID == "" {
		t.Error("New() should generate an ID")
	}
	if msg.Format != FormatText {
		t.Errorf("Message.Format = %v, want %v", msg.Format, FormatText)
	}
	if msg.Priority != PriorityNormal {
		t.Errorf("Message.Priority = %v, want %v", msg.Priority, PriorityNormal)
	}
	if msg.CreatedAt.IsZero() {
		t.Error("Message.CreatedAt should not be zero")
	}
	if len(msg.Targets) != 0 {
		t.Errorf("Message.Targets length = %v, want %v", len(msg.Targets), 0)
	}
}

func TestMessage_SetMethods(t *testing.T) {
	msg := New()

	// Test SetTitle
	msg.SetTitle("Test Title")
	if msg.Title != "Test Title" {
		t.Errorf("SetTitle() failed, got %v", msg.Title)
	}

	// Test SetBody
	msg.SetBody("Test Body")
	if msg.Body != "Test Body" {
		t.Errorf("SetBody() failed, got %v", msg.Body)
	}

	// Test SetFormat
	msg.SetFormat(FormatMarkdown)
	if msg.Format != FormatMarkdown {
		t.Errorf("SetFormat() failed, got %v", msg.Format)
	}

	// Test SetPriority
	msg.SetPriority(PriorityHigh)
	if msg.Priority != PriorityHigh {
		t.Errorf("SetPriority() failed, got %v", msg.Priority)
	}

	// Test AddTarget
	tgt := target.NewEmail("test@example.com")
	msg.AddTarget(tgt)
	if len(msg.Targets) != 1 {
		t.Errorf("AddTarget() failed, got %v targets", len(msg.Targets))
	}

	// Test SetTargets
	targets := []target.Target{
		target.NewEmail("test1@example.com"),
		target.NewEmail("test2@example.com"),
	}
	msg.SetTargets(targets)
	if len(msg.Targets) != 2 {
		t.Errorf("SetTargets() failed, got %v targets", len(msg.Targets))
	}
}

func TestMessage_SetMetadata(t *testing.T) {
	msg := New()

	msg.SetMetadata("key1", "value1")
	msg.SetMetadata("key2", 123)

	if len(msg.Metadata) != 2 {
		t.Errorf("Metadata length = %v, want %v", len(msg.Metadata), 2)
	}
	if msg.Metadata["key1"] != "value1" {
		t.Errorf("Metadata[key1] = %v, want %v", msg.Metadata["key1"], "value1")
	}
	if msg.Metadata["key2"] != 123 {
		t.Errorf("Metadata[key2] = %v, want %v", msg.Metadata["key2"], 123)
	}
}

func TestMessage_SetVariable(t *testing.T) {
	msg := New()

	msg.SetVariable("Name", "John")
	msg.SetVariable("Age", 30)

	if len(msg.Variables) != 2 {
		t.Errorf("Variables length = %v, want %v", len(msg.Variables), 2)
	}
	if msg.Variables["Name"] != "John" {
		t.Errorf("Variables[Name] = %v, want %v", msg.Variables["Name"], "John")
	}
	if msg.Variables["Age"] != 30 {
		t.Errorf("Variables[Age] = %v, want %v", msg.Variables["Age"], 30)
	}
}

func TestMessage_ScheduleAt(t *testing.T) {
	msg := New()
	scheduleTime := time.Now().Add(1 * time.Hour)

	msg.ScheduleAt(scheduleTime)

	if msg.ScheduledAt == nil {
		t.Fatal("ScheduledAt should not be nil")
	}
	if !msg.ScheduledAt.Equal(scheduleTime) {
		t.Errorf("ScheduledAt = %v, want %v", msg.ScheduledAt, scheduleTime)
	}
}

func TestMessage_IsScheduled(t *testing.T) {
	tests := []struct {
		name     string
		msg      *Message
		expected bool
	}{
		{
			name: "scheduled in future",
			msg: func() *Message {
				m := New()
				future := time.Now().Add(1 * time.Hour)
				m.ScheduleAt(future)
				return m
			}(),
			expected: true,
		},
		{
			name: "scheduled in past",
			msg: func() *Message {
				m := New()
				past := time.Now().Add(-1 * time.Hour)
				m.ScheduleAt(past)
				return m
			}(),
			expected: false,
		},
		{
			name:     "not scheduled",
			msg:      New(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.IsScheduled(); got != tt.expected {
				t.Errorf("IsScheduled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormat_String(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatText, "text"},
		{FormatMarkdown, "markdown"},
		{FormatHTML, "html"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := string(tt.format); got != tt.want {
				t.Errorf("Format.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriority_Values(t *testing.T) {
	tests := []struct {
		priority Priority
		want     int
	}{
		{PriorityLow, 0},
		{PriorityNormal, 1},
		{PriorityHigh, 2},
		{PriorityUrgent, 3},
	}

	for _, tt := range tests {
		if int(tt.priority) != tt.want {
			t.Errorf("Priority value = %v, want %v", tt.priority, tt.want)
		}
	}
}

func TestMessage_SetPlatformData(t *testing.T) {
	msg := New()

	msg.SetPlatformData("email", map[string]string{"cc": "cc@example.com"})
	msg.SetPlatformData("feishu", map[string]string{"chat_id": "oc_123"})

	if len(msg.PlatformData) != 2 {
		t.Errorf("PlatformData length = %v, want %v", len(msg.PlatformData), 2)
	}
	if msg.PlatformData["email"] == nil {
		t.Error("PlatformData[email] should not be nil")
	}
	if msg.PlatformData["feishu"] == nil {
		t.Error("PlatformData[feishu] should not be nil")
	}
}
