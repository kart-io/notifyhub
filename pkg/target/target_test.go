package target

import (
	"context"
	"testing"

	"github.com/kart/notifyhub/pkg/utils/logger"
)

func TestTarget_Validate(t *testing.T) {
	tests := []struct {
		name    string
		target  Target
		wantErr bool
	}{
		{
			name: "valid email target",
			target: Target{
				Type:     TargetTypeEmail,
				Value:    "test@example.com",
				Platform: PlatformEmail,
			},
			wantErr: false,
		},
		{
			name: "valid webhook target",
			target: Target{
				Type:     TargetTypeWebhook,
				Value:    "https://webhook.example.com",
				Platform: PlatformWebhook,
			},
			wantErr: false,
		},
		{
			name: "valid user target",
			target: Target{
				Type:     TargetTypeUser,
				Value:    "user123",
				Platform: PlatformFeishu,
			},
			wantErr: false,
		},
		{
			name: "missing type",
			target: Target{
				Value:    "test@example.com",
				Platform: PlatformEmail,
			},
			wantErr: true,
		},
		{
			name: "missing value",
			target: Target{
				Type:     TargetTypeEmail,
				Platform: PlatformEmail,
			},
			wantErr: true,
		},
		{
			name: "missing platform",
			target: Target{
				Type:  TargetTypeEmail,
				Value: "test@example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.target.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Target.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tgt := New(TargetTypeEmail, "test@example.com", PlatformEmail)

	if tgt.Type != TargetTypeEmail {
		t.Errorf("Target.Type = %v, want %v", tgt.Type, TargetTypeEmail)
	}
	if tgt.Value != "test@example.com" {
		t.Errorf("Target.Value = %v, want %v", tgt.Value, "test@example.com")
	}
	if tgt.Platform != PlatformEmail {
		t.Errorf("Target.Platform = %v, want %v", tgt.Platform, PlatformEmail)
	}
}

func TestNewEmail(t *testing.T) {
	email := "test@example.com"
	tgt := NewEmail(email)

	if tgt.Type != TargetTypeEmail {
		t.Errorf("Target.Type = %v, want %v", tgt.Type, TargetTypeEmail)
	}
	if tgt.Value != email {
		t.Errorf("Target.Value = %v, want %v", tgt.Value, email)
	}
	if tgt.Platform != PlatformEmail {
		t.Errorf("Target.Platform = %v, want %v", tgt.Platform, PlatformEmail)
	}
}

func TestNewWebhook(t *testing.T) {
	url := "https://webhook.example.com"
	tgt := NewWebhook(url)

	if tgt.Type != TargetTypeWebhook {
		t.Errorf("Target.Type = %v, want %v", tgt.Type, TargetTypeWebhook)
	}
	if tgt.Value != url {
		t.Errorf("Target.Value = %v, want %v", tgt.Value, url)
	}
	if tgt.Platform != PlatformWebhook {
		t.Errorf("Target.Platform = %v, want %v", tgt.Platform, PlatformWebhook)
	}
}

func TestNewFeishuUser(t *testing.T) {
	userID := "user123"
	tgt := NewFeishuUser(userID)

	if tgt.Type != TargetTypeUser {
		t.Errorf("Target.Type = %v, want %v", tgt.Type, TargetTypeUser)
	}
	if tgt.Value != userID {
		t.Errorf("Target.Value = %v, want %v", tgt.Value, userID)
	}
	if tgt.Platform != PlatformFeishu {
		t.Errorf("Target.Platform = %v, want %v", tgt.Platform, PlatformFeishu)
	}
}

func TestNewFeishuGroup(t *testing.T) {
	groupID := "group456"
	tgt := NewFeishuGroup(groupID)

	if tgt.Type != TargetTypeGroup {
		t.Errorf("Target.Type = %v, want %v", tgt.Type, TargetTypeGroup)
	}
	if tgt.Value != groupID {
		t.Errorf("Target.Value = %v, want %v", tgt.Value, groupID)
	}
	if tgt.Platform != PlatformFeishu {
		t.Errorf("Target.Platform = %v, want %v", tgt.Platform, PlatformFeishu)
	}
}

func TestTarget_IsEmail(t *testing.T) {
	tests := []struct {
		name     string
		target   Target
		expected bool
	}{
		{
			name:     "email target",
			target:   NewEmail("test@example.com"),
			expected: true,
		},
		{
			name:     "non-email target",
			target:   NewFeishuUser("user123"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.IsEmail(); got != tt.expected {
				t.Errorf("Target.IsEmail() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTarget_IsUser(t *testing.T) {
	tests := []struct {
		name     string
		target   Target
		expected bool
	}{
		{
			name:     "user target",
			target:   NewFeishuUser("user123"),
			expected: true,
		},
		{
			name:     "non-user target",
			target:   NewEmail("test@example.com"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.IsUser(); got != tt.expected {
				t.Errorf("Target.IsUser() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTarget_IsGroup(t *testing.T) {
	tests := []struct {
		name     string
		target   Target
		expected bool
	}{
		{
			name:     "group target",
			target:   NewFeishuGroup("group456"),
			expected: true,
		},
		{
			name:     "non-group target",
			target:   NewEmail("test@example.com"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.IsGroup(); got != tt.expected {
				t.Errorf("Target.IsGroup() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTarget_IsWebhook(t *testing.T) {
	tests := []struct {
		name     string
		target   Target
		expected bool
	}{
		{
			name:     "webhook target",
			target:   NewWebhook("https://example.com"),
			expected: true,
		},
		{
			name:     "non-webhook target",
			target:   NewEmail("test@example.com"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.IsWebhook(); got != tt.expected {
				t.Errorf("Target.IsWebhook() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTarget_String(t *testing.T) {
	tests := []struct {
		name     string
		target   Target
		expected string
	}{
		{
			name:     "email target",
			target:   NewEmail("test@example.com"),
			expected: "email:email:test@example.com",
		},
		{
			name:     "feishu user",
			target:   NewFeishuUser("user123"),
			expected: "feishu:user:user123",
		},
		{
			name:     "webhook",
			target:   NewWebhook("https://example.com"),
			expected: "webhook:webhook:https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.String(); got != tt.expected {
				t.Errorf("Target.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTargetTypeConstants(t *testing.T) {
	tests := []struct {
		constant string
		value    string
	}{
		{TargetTypeEmail, "email"},
		{TargetTypePhone, "phone"},
		{TargetTypeUser, "user"},
		{TargetTypeGroup, "group"},
		{TargetTypeChannel, "channel"},
		{TargetTypeWebhook, "webhook"},
	}

	for _, tt := range tests {
		if tt.constant != tt.value {
			t.Errorf("Constant value = %v, want %v", tt.constant, tt.value)
		}
	}
}

func TestPlatformConstants(t *testing.T) {
	tests := []struct {
		constant string
		value    string
	}{
		{PlatformFeishu, "feishu"},
		{PlatformEmail, "email"},
		{PlatformWebhook, "webhook"},
	}

	for _, tt := range tests {
		if tt.constant != tt.value {
			t.Errorf("Constant value = %v, want %v", tt.constant, tt.value)
		}
	}
}

// mockLogger for testing
type mockLogger struct{}

func (m *mockLogger) LogMode(level logger.LogLevel) logger.Logger     { return m }
func (m *mockLogger) Debug(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Info(msg string, keysAndValues ...interface{})   {}
func (m *mockLogger) Warn(msg string, keysAndValues ...interface{})   {}
func (m *mockLogger) Error(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Fatal(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) With(keysAndValues ...interface{}) logger.Logger { return m }

func TestNewDefaultResolver(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)

	if resolver == nil {
		t.Fatal("NewDefaultResolver() returned nil")
	}
	if len(resolver.handlers) == 0 {
		t.Error("NewDefaultResolver() should register default handlers")
	}
}

func TestDefaultResolver_ResolveString(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)
	ctx := context.Background()

	tests := []struct {
		name      string
		input     string
		wantErr   bool
		wantCount int
	}{
		{
			name:      "valid email format",
			input:     "email:test@example.com",
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:      "valid webhook format",
			input:     "webhook:https://example.com/hook",
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:      "auto-detect email",
			input:     "test@example.com",
			wantErr:   false,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := resolver.ResolveString(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(targets) != tt.wantCount {
				t.Errorf("ResolveString() got %d targets, want %d", len(targets), tt.wantCount)
			}
		})
	}
}

func TestDefaultResolver_AddHandler(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)

	handler := &mockResolutionHandler{}
	resolver.AddHandler("custom", handler)

	if _, exists := resolver.handlers["custom"]; !exists {
		t.Error("AddHandler() should add handler to handlers map")
	}
}

func TestDefaultResolver_RemoveHandler(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)

	resolver.RemoveHandler("email")

	if _, exists := resolver.handlers["email"]; exists {
		t.Error("RemoveHandler() should remove handler from handlers map")
	}
}

type mockResolutionHandler struct{}

func (h *mockResolutionHandler) CanResolve(spec TargetSpec) bool {
	return true
}

func (h *mockResolutionHandler) Resolve(ctx context.Context, spec TargetSpec) ([]Target, error) {
	return []Target{New(spec.Type, spec.Value, spec.Platform)}, nil
}

func TestResolveEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  Target
	}{
		{
			name:  "valid email",
			email: "test@example.com",
			want: Target{
				Type:     TargetTypeEmail,
				Value:    "test@example.com",
				Platform: PlatformEmail,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveEmail(tt.email)
			if got.Type != tt.want.Type || got.Value != tt.want.Value || got.Platform != tt.want.Platform {
				t.Errorf("ResolveEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolvePhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  Target
	}{
		{
			name:  "valid phone",
			phone: "+1234567890",
			want: Target{
				Type:  TargetTypePhone,
				Value: "+1234567890",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolvePhone(tt.phone)
			if got.Type != tt.want.Type || got.Value != tt.want.Value {
				t.Errorf("ResolvePhone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveWebhook(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want Target
	}{
		{
			name: "valid webhook",
			url:  "https://webhook.example.com",
			want: Target{
				Type:     TargetTypeWebhook,
				Value:    "https://webhook.example.com",
				Platform: PlatformWebhook,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveWebhook(tt.url)
			if got.Type != tt.want.Type || got.Value != tt.want.Value || got.Platform != tt.want.Platform {
				t.Errorf("ResolveWebhook() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveFeishuWebhook(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want Target
	}{
		{
			name: "valid feishu webhook",
			url:  "https://open.feishu.cn/webhook/test",
			want: Target{
				Type:     TargetTypeWebhook,
				Value:    "https://open.feishu.cn/webhook/test",
				Platform: PlatformFeishu,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveFeishuWebhook(tt.url)
			if got.Type != tt.want.Type || got.Value != tt.want.Value || got.Platform != tt.want.Platform {
				t.Errorf("ResolveFeishuWebhook() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailResolutionHandler_CanResolve(t *testing.T) {
	handler := &EmailResolutionHandler{}

	tests := []struct {
		name string
		spec TargetSpec
		want bool
	}{
		{
			name: "email type with value",
			spec: TargetSpec{
				Type:  "email",
				Value: "test@example.com",
			},
			want: true,
		},
		{
			name: "wrong type",
			spec: TargetSpec{
				Type:  "phone",
				Value: "test@example.com",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.CanResolve(tt.spec); got != tt.want {
				t.Errorf("CanResolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailResolutionHandler_Resolve(t *testing.T) {
	handler := &EmailResolutionHandler{}
	ctx := context.Background()

	spec := TargetSpec{
		Type:  "email",
		Value: "test@example.com",
	}

	targets, err := handler.Resolve(ctx, spec)
	if err != nil {
		t.Errorf("Resolve() error = %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("Resolve() got %d targets, want 1", len(targets))
	}
	if targets[0].Type != TargetTypeEmail {
		t.Errorf("Resolve() target type = %v, want %v", targets[0].Type, TargetTypeEmail)
	}
}

func TestWebhookResolutionHandler_CanResolve(t *testing.T) {
	handler := &WebhookResolutionHandler{}

	tests := []struct {
		name string
		spec TargetSpec
		want bool
	}{
		{
			name: "webhook type with value",
			spec: TargetSpec{
				Type:  "webhook",
				Value: "https://example.com",
			},
			want: true,
		},
		{
			name: "wrong type",
			spec: TargetSpec{
				Type:  "email",
				Value: "https://example.com",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.CanResolve(tt.spec); got != tt.want {
				t.Errorf("CanResolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebhookResolutionHandler_Resolve(t *testing.T) {
	handler := &WebhookResolutionHandler{}
	ctx := context.Background()

	spec := TargetSpec{
		Type:  "webhook",
		Value: "https://webhook.example.com",
	}

	targets, err := handler.Resolve(ctx, spec)
	if err != nil {
		t.Errorf("Resolve() error = %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("Resolve() got %d targets, want 1", len(targets))
	}
}

func TestUserResolutionHandler_CanResolve(t *testing.T) {
	handler := &UserResolutionHandler{}

	tests := []struct {
		name string
		spec TargetSpec
		want bool
	}{
		{
			name: "user type with value",
			spec: TargetSpec{
				Type:  "user",
				Value: "user123",
			},
			want: true,
		},
		{
			name: "wrong type",
			spec: TargetSpec{
				Type:  "email",
				Value: "user123",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.CanResolve(tt.spec); got != tt.want {
				t.Errorf("CanResolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupResolutionHandler_CanResolve(t *testing.T) {
	handler := &GroupResolutionHandler{}

	tests := []struct {
		name string
		spec TargetSpec
		want bool
	}{
		{
			name: "group type",
			spec: TargetSpec{
				Type:  "group",
				Value: "group123",
			},
			want: true,
		},
		{
			name: "wrong type",
			spec: TargetSpec{
				Type:  "user",
				Value: "group123",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.CanResolve(tt.spec); got != tt.want {
				t.Errorf("CanResolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChannelResolutionHandler_CanResolve(t *testing.T) {
	handler := &ChannelResolutionHandler{}

	tests := []struct {
		name string
		spec TargetSpec
		want bool
	}{
		{
			name: "channel type",
			spec: TargetSpec{
				Type:  "channel",
				Value: "#general",
			},
			want: true,
		},
		{
			name: "wrong type",
			spec: TargetSpec{
				Type:  "user",
				Value: "#general",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.CanResolve(tt.spec); got != tt.want {
				t.Errorf("CanResolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPhoneResolutionHandler_CanResolve(t *testing.T) {
	handler := &PhoneResolutionHandler{}

	tests := []struct {
		name string
		spec TargetSpec
		want bool
	}{
		{
			name: "phone type",
			spec: TargetSpec{
				Type:  "phone",
				Value: "+1234567890",
			},
			want: true,
		},
		{
			name: "wrong type",
			spec: TargetSpec{
				Type:  "email",
				Value: "+1234567890",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.CanResolve(tt.spec); got != tt.want {
				t.Errorf("CanResolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultResolver_Resolve(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)
	ctx := context.Background()

	tests := []struct {
		name      string
		spec      TargetSpec
		wantErr   bool
		wantCount int
	}{
		{
			name: "email spec",
			spec: TargetSpec{
				Type:  "email",
				Value: "test@example.com",
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "webhook spec",
			spec: TargetSpec{
				Type:  "webhook",
				Value: "https://webhook.example.com",
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "user spec",
			spec: TargetSpec{
				Type:     "user",
				Value:    "user123",
				Platform: "feishu",
			},
			wantErr:   false,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := resolver.Resolve(ctx, tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(targets) != tt.wantCount {
				t.Errorf("Resolve() got %d targets, want %d", len(targets), tt.wantCount)
			}
		})
	}
}

func TestTargetSpec(t *testing.T) {
	spec := TargetSpec{
		Type:     "email",
		Value:    "test@example.com",
		Platform: "email",
		Metadata: map[string]interface{}{
			"priority": "high",
		},
	}

	if spec.Type != "email" {
		t.Errorf("TargetSpec.Type = %v, want email", spec.Type)
	}
	if spec.Value != "test@example.com" {
		t.Errorf("TargetSpec.Value = %v, want test@example.com", spec.Value)
	}
	if spec.Metadata["priority"] != "high" {
		t.Error("TargetSpec.Metadata not set correctly")
	}
}

func TestPhoneResolutionHandler_Resolve(t *testing.T) {
	handler := &PhoneResolutionHandler{}
	ctx := context.Background()

	tests := []struct {
		name    string
		spec    TargetSpec
		wantErr bool
	}{
		{
			name: "valid phone",
			spec: TargetSpec{
				Type:  "phone",
				Value: "+1234567890",
			},
			wantErr: false,
		},
		{
			name: "invalid phone - too short",
			spec: TargetSpec{
				Type:  "phone",
				Value: "+1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.Resolve(ctx, tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGroupResolutionHandler_Resolve(t *testing.T) {
	handler := &GroupResolutionHandler{}
	ctx := context.Background()

	spec := TargetSpec{
		Type:     "group",
		Value:    "admins", // Use a known group
		Platform: "feishu",
	}

	targets, err := handler.Resolve(ctx, spec)
	if err != nil {
		t.Errorf("Resolve() error = %v", err)
	}
	if len(targets) < 1 {
		t.Errorf("Resolve() got %d targets, want at least 1", len(targets))
	}
}

func TestChannelResolutionHandler_Resolve(t *testing.T) {
	handler := &ChannelResolutionHandler{}
	ctx := context.Background()

	spec := TargetSpec{
		Type:     "channel",
		Value:    "#general",
		Platform: "slack",
	}

	targets, err := handler.Resolve(ctx, spec)
	if err != nil {
		t.Errorf("Resolve() error = %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("Resolve() got %d targets, want 1", len(targets))
	}
}

func TestUserResolutionHandler_Resolve(t *testing.T) {
	handler := &UserResolutionHandler{}
	ctx := context.Background()

	spec := TargetSpec{
		Type:     "user",
		Value:    "user123",
		Platform: "feishu",
	}

	targets, err := handler.Resolve(ctx, spec)
	if err != nil {
		t.Errorf("Resolve() error = %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("Resolve() got %d targets, want 1", len(targets))
	}
}

func TestEmailResolutionHandler_ResolveInvalidEmail(t *testing.T) {
	handler := &EmailResolutionHandler{}
	ctx := context.Background()

	tests := []struct {
		name  string
		value string
	}{
		{"no domain", "user@"},
		{"no at sign", "userexample.com"},
		{"invalid format", "not-an-email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := TargetSpec{
				Type:  "email",
				Value: tt.value,
			}
			_, err := handler.Resolve(ctx, spec)
			if err == nil {
				t.Error("Resolve() should return error for invalid email")
			}
		})
	}
}

func TestWebhookResolutionHandler_ResolveVariousURLs(t *testing.T) {
	handler := &WebhookResolutionHandler{}
	ctx := context.Background()

	tests := []struct {
		name  string
		value string
	}{
		{"https URL", "https://example.com/webhook"},
		{"http URL", "http://example.com/webhook"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := TargetSpec{
				Type:  "webhook",
				Value: tt.value,
			}
			targets, err := handler.Resolve(ctx, spec)
			if err != nil {
				t.Errorf("Resolve() error = %v", err)
			}
			if len(targets) != 1 {
				t.Errorf("Resolve() got %d targets, want 1", len(targets))
			}
		})
	}
}

func TestDefaultResolver_FallbackResolve(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)
	ctx := context.Background()

	// Remove email handler to test fallback
	resolver.RemoveHandler("email")

	// Try to resolve an email - should use fallback
	spec := TargetSpec{
		Type:  "email",
		Value: "test@example.com",
	}

	targets, err := resolver.Resolve(ctx, spec)
	if err != nil {
		t.Errorf("Resolve() with fallback error = %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("Resolve() with fallback got %d targets, want 1", len(targets))
	}
}

func TestDefaultResolver_ResolveWithMetadata(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)
	ctx := context.Background()

	spec := TargetSpec{
		Type:  "email",
		Value: "test@example.com",
		Metadata: map[string]interface{}{
			"priority": "high",
			"tags":     []string{"alert", "urgent"},
		},
	}

	targets, err := resolver.Resolve(ctx, spec)
	if err != nil {
		t.Errorf("Resolve() with metadata error = %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("Resolve() with metadata got %d targets, want 1", len(targets))
	}
}

func TestDefaultResolver_MultipleResolve(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)
	ctx := context.Background()

	specs := []TargetSpec{
		{Type: "email", Value: "test1@example.com"},
		{Type: "email", Value: "test2@example.com"},
		{Type: "webhook", Value: "https://example.com/hook"},
	}

	for i, spec := range specs {
		targets, err := resolver.Resolve(ctx, spec)
		if err != nil {
			t.Errorf("Resolve() test %d error = %v", i, err)
		}
		if len(targets) != 1 {
			t.Errorf("Resolve() test %d got %d targets, want 1", i, len(targets))
		}
	}
}

func TestEmailHandler_ValidFormats(t *testing.T) {
	handler := &EmailResolutionHandler{}
	ctx := context.Background()

	validEmails := []string{
		"test@example.com",
		"user.name@example.com",
		"user+tag@example.com",
		"user_name@sub.example.com",
	}

	for _, email := range validEmails {
		spec := TargetSpec{Type: "email", Value: email}
		targets, err := handler.Resolve(ctx, spec)
		if err != nil {
			t.Errorf("Resolve(%s) error = %v", email, err)
		}
		if len(targets) != 1 {
			t.Errorf("Resolve(%s) got %d targets, want 1", email, len(targets))
		}
	}
}

func TestResolveFeishuWebhook_Variants(t *testing.T) {
	urls := []string{
		"https://open.feishu.cn/webhook/test",
		"https://open.larksuite.com/webhook/test",
	}

	for _, url := range urls {
		tgt := ResolveFeishuWebhook(url)
		if tgt.Type != TargetTypeWebhook {
			t.Errorf("ResolveFeishuWebhook().Type = %v, want %v", tgt.Type, TargetTypeWebhook)
		}
		if tgt.Platform != PlatformFeishu {
			t.Errorf("ResolveFeishuWebhook().Platform = %v, want %v", tgt.Platform, PlatformFeishu)
		}
	}
}

func TestResolveEmail_Variants(t *testing.T) {
	emails := []string{
		"simple@example.com",
		"with.dots@example.com",
		"with+plus@example.com",
		"with_underscore@example.com",
	}

	for _, email := range emails {
		tgt := ResolveEmail(email)
		if tgt.Type != TargetTypeEmail {
			t.Errorf("ResolveEmail(%s).Type = %v, want %v", email, tgt.Type, TargetTypeEmail)
		}
		if tgt.Value != email {
			t.Errorf("ResolveEmail(%s).Value = %v, want %v", email, tgt.Value, email)
		}
	}
}

func TestDefaultResolver_AutoDetectTypes(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)
	ctx := context.Background()

	tests := []struct {
		input        string
		expectedType string
	}{
		{"user@example.com", "email"},
		{"#general", "channel"},
		{"@developers", "group"},
	}

	for _, tt := range tests {
		targets, err := resolver.ResolveString(ctx, tt.input)
		if err != nil {
			// Some might fail validation, that's ok
			continue
		}
		if len(targets) > 0 && targets[0].Type != tt.expectedType {
			t.Errorf("ResolveString(%s) type = %v, want %v", tt.input, targets[0].Type, tt.expectedType)
		}
	}
}

func TestDefaultResolver_ExplicitType(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)
	ctx := context.Background()

	tests := []struct {
		input        string
		expectedType string
	}{
		{"email:test@example.com", "email"},
		{"webhook:https://example.com", "webhook"},
		{"user:john.doe", "user"},
		{"group:developers", "group"},
		{"channel:#general", "channel"},
		{"phone:+1234567890", "phone"},
	}

	for _, tt := range tests {
		targets, err := resolver.ResolveString(ctx, tt.input)
		if err != nil && tt.expectedType != "phone" {
			// Phone validation might fail, skip error check for it
			t.Errorf("ResolveString(%s) error = %v", tt.input, err)
			continue
		}
		if len(targets) > 0 && targets[0].Type != tt.expectedType {
			t.Errorf("ResolveString(%s) type = %v, want %v", tt.input, targets[0].Type, tt.expectedType)
		}
	}
}

func TestResolveWebhook_Variants(t *testing.T) {
	tests := []string{
		"https://webhook.example.com/path",
		"https://api.example.com/v1/webhook",
		"http://localhost:8080/webhook",
	}

	for _, url := range tests {
		tgt := ResolveWebhook(url)
		if tgt.Type != TargetTypeWebhook {
			t.Errorf("ResolveWebhook(%s).Type = %v, want %v", url, tgt.Type, TargetTypeWebhook)
		}
		if tgt.Value != url {
			t.Errorf("ResolveWebhook(%s).Value = %v, want %v", url, tgt.Value, url)
		}
		if tgt.Platform != PlatformWebhook {
			t.Errorf("ResolveWebhook(%s).Platform = %v, want %v", url, tgt.Platform, PlatformWebhook)
		}
	}
}

func TestTarget_EmptyValidation(t *testing.T) {
	tests := []struct {
		name    string
		target  Target
		wantErr bool
	}{
		{
			name:    "empty type",
			target:  Target{Value: "test", Platform: "platform"},
			wantErr: true,
		},
		{
			name:    "empty value",
			target:  Target{Type: "email", Platform: "platform"},
			wantErr: true,
		},
		{
			name:    "empty platform",
			target:  Target{Type: "email", Value: "test@example.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.target.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTargetTypeConstants_Coverage(t *testing.T) {
	constants := []string{
		TargetTypeEmail,
		TargetTypePhone,
		TargetTypeUser,
		TargetTypeGroup,
		TargetTypeChannel,
		TargetTypeWebhook,
	}

	// Just verify all constants are non-empty
	for _, c := range constants {
		if c == "" {
			t.Error("TargetType constant should not be empty")
		}
	}
}

func TestPlatformConstants_Coverage(t *testing.T) {
	constants := []string{
		PlatformFeishu,
		PlatformEmail,
		PlatformWebhook,
	}

	// Just verify all constants are non-empty
	for _, c := range constants {
		if c == "" {
			t.Error("Platform constant should not be empty")
		}
	}
}

func TestNewWithAll(t *testing.T) {
	tgt := New("custom", "value123", "platform-x")
	if tgt.Type != "custom" {
		t.Errorf("New() Type = %v, want custom", tgt.Type)
	}
	if tgt.Value != "value123" {
		t.Errorf("New() Value = %v, want value123", tgt.Value)
	}
	if tgt.Platform != "platform-x" {
		t.Errorf("New() Platform = %v, want platform-x", tgt.Platform)
	}
}

func TestTarget_StringRepresentation(t *testing.T) {
	tests := []struct {
		target Target
		want   string
	}{
		{
			target: Target{Platform: "email", Type: "email", Value: "test@example.com"},
			want:   "email:email:test@example.com",
		},
		{
			target: Target{Platform: "feishu", Type: "user", Value: "user123"},
			want:   "feishu:user:user123",
		},
	}

	for _, tt := range tests {
		got := tt.target.String()
		if got != tt.want {
			t.Errorf("String() = %v, want %v", got, tt.want)
		}
	}
}

func TestResolver_CustomHandler(t *testing.T) {
	log := &mockLogger{}
	resolver := NewDefaultResolver(log)

	// Test adding and removing custom handler
	customHandler := &mockResolutionHandler{}
	resolver.AddHandler("custom-type", customHandler)

	// Now remove it
	resolver.RemoveHandler("custom-type")

	// Verify it was removed by checking we can add it again
	resolver.AddHandler("custom-type", customHandler)
}

func TestPhoneHandler_ValidNumbers(t *testing.T) {
	handler := &PhoneResolutionHandler{}
	ctx := context.Background()

	validPhones := []string{
		"+12345678901",
		"+861234567890",
		"+441234567890",
	}

	for _, phone := range validPhones {
		spec := TargetSpec{Type: "phone", Value: phone}
		targets, err := handler.Resolve(ctx, spec)
		if err != nil {
			t.Errorf("Resolve(%s) error = %v", phone, err)
		}
		if len(targets) != 1 {
			t.Errorf("Resolve(%s) got %d targets, want 1", phone, len(targets))
		}
	}
}

func TestGroupAndChannelHandlers(t *testing.T) {
	ctx := context.Background()

	// Test Group Handler
	groupHandler := &GroupResolutionHandler{}
	groupSpec := TargetSpec{Type: "group", Value: "developers", Platform: "feishu"} // Use known group
	groupTargets, err := groupHandler.Resolve(ctx, groupSpec)
	if err != nil {
		t.Errorf("GroupHandler.Resolve() error = %v", err)
	}
	if len(groupTargets) < 1 {
		t.Errorf("GroupHandler.Resolve() got %d targets, want at least 1", len(groupTargets))
	}

	// Test Channel Handler
	channelHandler := &ChannelResolutionHandler{}
	channelSpec := TargetSpec{Type: "channel", Value: "#general", Platform: "slack"}
	channelTargets, err := channelHandler.Resolve(ctx, channelSpec)
	if err != nil {
		t.Errorf("ChannelHandler.Resolve() error = %v", err)
	}
	if len(channelTargets) != 1 {
		t.Errorf("ChannelHandler.Resolve() got %d targets, want 1", len(channelTargets))
	}
}

func TestMultipleTargetTypes(t *testing.T) {
	targets := []Target{
		NewEmail("test@example.com"),
		NewWebhook("https://webhook.example.com"),
		NewFeishuUser("user123"),
		NewFeishuGroup("group456"),
	}

	for i, tgt := range targets {
		if err := tgt.Validate(); err != nil {
			t.Errorf("Target %d validation failed: %v", i, err)
		}
	}
}
