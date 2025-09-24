package email

import (
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

var (
	registerOnce     sync.Once
	useGoMailLibrary = true // Use modern go-mail by default
)

// UseNetSMTP switches to use deprecated net/smtp library
// This is for backward compatibility only. Prefer go-mail library.
func UseNetSMTP() {
	useGoMailLibrary = false
}

// UseGoMail switches to use modern go-mail library (default)
func UseGoMail() {
	useGoMailLibrary = true
}

// ensureRegistered ensures the Email platform is registered with NotifyHub
func ensureRegistered() {
	registerOnce.Do(func() {
		var creator func(map[string]interface{}) (platform.ExternalSender, error)

		if useGoMailLibrary {
			creator = NewEmailSenderGoMail
		} else {
			creator = NewEmailSender
		}

		_ = notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
			Name:    "email",
			Creator: creator,
			DefaultOpts: func() map[string]interface{} {
				return map[string]interface{}{
					"timeout":   30 * time.Second,
					"smtp_tls":  true,
					"smtp_ssl":  false,
					"smtp_port": 587,
				}
			},
			Validator: func(config map[string]interface{}) error {
				if _, ok := config["smtp_host"].(string); !ok {
					return fmt.Errorf("smtp_host is required")
				}
				if config["smtp_host"].(string) == "" {
					return fmt.Errorf("smtp_host cannot be empty")
				}

				if _, ok := config["smtp_port"].(int); !ok {
					return fmt.Errorf("smtp_port is required")
				}

				if _, ok := config["smtp_from"].(string); !ok {
					return fmt.Errorf("smtp_from is required")
				}
				if config["smtp_from"].(string) == "" {
					return fmt.Errorf("smtp_from cannot be empty")
				}

				return nil
			},
		})
	})
}

// WithEmail creates a HubOption for Email platform with SMTP configuration
// This function automatically registers the Email platform if not already registered
func WithEmail(smtpHost string, smtpPort int, smtpFrom string, options ...func(map[string]interface{})) notifyhub.HubOption {
	ensureRegistered()

	config := map[string]interface{}{
		"smtp_host": smtpHost,
		"smtp_port": smtpPort,
		"smtp_from": smtpFrom,
		"timeout":   30 * time.Second,
		"smtp_tls":  true,
		"smtp_ssl":  false,
	}

	// Apply additional options
	for _, opt := range options {
		opt(config)
	}

	return notifyhub.WithCustomPlatform("email", config)
}

// WithEmailAuth adds SMTP authentication credentials
func WithEmailAuth(username, password string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["smtp_username"] = username
		config["smtp_password"] = password
	}
}

// WithEmailTLS enables or disables TLS for SMTP connection
func WithEmailTLS(useTLS bool) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["smtp_tls"] = useTLS
	}
}

// WithEmailSSL enables or disables SSL for SMTP connection
func WithEmailSSL(useSSL bool) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["smtp_ssl"] = useSSL
	}
}

// WithEmailTimeout sets the timeout for email operations
func WithEmailTimeout(timeout time.Duration) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["timeout"] = timeout
	}
}

// Legacy compatibility functions (these will call the new platform-specific ones)

// WithEmailSMTP is a legacy function that configures Email with SMTP settings
// Deprecated: Use WithEmail with additional options instead
func WithEmailSMTP(host string, port int, username, password, from string, useTLS bool, timeout time.Duration) notifyhub.HubOption {
	return WithEmail(host, port, from,
		WithEmailAuth(username, password),
		WithEmailTLS(useTLS),
		WithEmailTimeout(timeout),
	)
}

// WithGmail is a helper function to create a Gmail email sender
// This function automatically registers the Email platform if not already registered
// and uses the go-mail library
func WithGmailSMTP(username, password string) notifyhub.HubOption {
	return WithEmail("smtp.gmail.com", 587, username,
		WithEmailAuth(username, password),
		WithEmailTLS(true),
	)
}

// With163SMTP is a helper function to create a 163 email sender
// This function automatically registers the Email platform if not already registered
// and uses the go-mail library
// 163 requires SSL on port 465 or 994
func With163SMTP(username, password string) notifyhub.HubOption {
	return WithEmail("smtp.163.com", 465, username,
		WithEmailAuth(username, password),
		WithEmailSSL(true),
		WithEmailTLS(false),
	)
}
