// Package target provides target factory functionality for NotifyHub
package target

// NewEmailTarget creates a new email target
func NewEmailTarget(email string) Target {
	return Target{
		Type:     TargetTypeEmail,
		Value:    email,
		Platform: PlatformEmail,
	}
}

// NewPhoneTarget creates a new phone/SMS target
func NewPhoneTarget(phone string) Target {
	return Target{
		Type:     TargetTypePhone,
		Value:    phone,
		Platform: PlatformSMS,
	}
}

// NewFeishuUserTarget creates a new Feishu user target
func NewFeishuUserTarget(userID string) Target {
	return Target{
		Type:     TargetTypeUser,
		Value:    userID,
		Platform: PlatformFeishu,
	}
}

// NewFeishuGroupTarget creates a new Feishu group target
func NewFeishuGroupTarget(groupID string) Target {
	return Target{
		Type:     TargetTypeGroup,
		Value:    groupID,
		Platform: PlatformFeishu,
	}
}

// NewWebhookTarget creates a new webhook target
func NewWebhookTarget(url string) Target {
	return Target{
		Type:     TargetTypeWebhook,
		Value:    url,
		Platform: PlatformWebhook,
	}
}
