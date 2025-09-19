package sending

// TargetType represents the type of notification target
type TargetType string

const (
	TargetTypeEmail   TargetType = "email"
	TargetTypeUser    TargetType = "user"
	TargetTypeGroup   TargetType = "group"
	TargetTypeChannel TargetType = "channel"
	TargetTypeWebhook TargetType = "webhook"
	TargetTypeSMS     TargetType = "sms"
	TargetTypeOther   TargetType = "other"
)

// Target represents a notification destination
type Target struct {
	Type     TargetType        `json:"type"`
	Value    string            `json:"value"`
	Platform string            `json:"platform"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewTarget creates a new target
func NewTarget(targetType TargetType, value, platform string) Target {
	return Target{
		Type:     targetType,
		Value:    value,
		Platform: platform,
		Metadata: make(map[string]string),
	}
}

// AddMetadata adds metadata to the target
func (t *Target) AddMetadata(key, value string) *Target {
	if t.Metadata == nil {
		t.Metadata = make(map[string]string)
	}
	t.Metadata[key] = value
	return t
}

// GetMetadata retrieves metadata value
func (t *Target) GetMetadata(key string) (string, bool) {
	if t.Metadata == nil {
		return "", false
	}
	value, exists := t.Metadata[key]
	return value, exists
}

// String returns a string representation of the target
func (t Target) String() string {
	return t.Platform + ":" + string(t.Type) + ":" + t.Value
}

// Validate checks if the target is valid
func (t *Target) Validate() error {
	if t.Type == "" {
		return ErrInvalidTargetType
	}
	if t.Value == "" {
		return ErrEmptyTargetValue
	}
	if t.Platform == "" {
		return ErrEmptyPlatform
	}
	return nil
}

// GetPlatform returns the target platform
func (t *Target) GetPlatform() string {
	return t.Platform
}

// GetValue returns the target value
func (t *Target) GetValue() string {
	return t.Value
}

// GetType returns the target type
func (t *Target) GetType() TargetType {
	return t.Type
}
