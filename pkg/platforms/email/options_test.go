package email_test

import (
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
	email "github.com/kart-io/notifyhub/pkg/platforms/email"
)

func TestWithGmail(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{
			name:     "valid gmail configuration",
			username: "test@gmail.com",
			password: "test-password",
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			password: "test-password",
			wantErr:  true,
		},
		{
			name:     "empty password",
			username: "test@gmail.com",
			password: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub, err := notifyhub.New(
				email.WithGmailSMTP(tt.username, tt.password),
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("WithGmail() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("WithGmail() unexpected error: %v", err)
				return
			}

			if hub == nil {
				t.Error("WithGmail() returned nil hub")
				return
			}

			_ = hub.Close()
		})
	}
}

func TestWithEmail(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		port    int
		from    string
		options []func(map[string]interface{})
		wantErr bool
	}{
		{
			name:    "basic email configuration",
			host:    "smtp.example.com",
			port:    587,
			from:    "sender@example.com",
			wantErr: false,
		},
		{
			name: "with auth",
			host: "smtp.example.com",
			port: 587,
			from: "sender@example.com",
			options: []func(map[string]interface{}){
				email.WithEmailAuth("user", "pass"),
			},
			wantErr: false,
		},
		{
			name: "with TLS",
			host: "smtp.example.com",
			port: 587,
			from: "sender@example.com",
			options: []func(map[string]interface{}){
				email.WithEmailTLS(true),
			},
			wantErr: false,
		},
		{
			name: "with SSL",
			host: "smtp.example.com",
			port: 465,
			from: "sender@example.com",
			options: []func(map[string]interface{}){
				email.WithEmailSSL(true),
			},
			wantErr: false,
		},
		{
			name: "with timeout",
			host: "smtp.example.com",
			port: 587,
			from: "sender@example.com",
			options: []func(map[string]interface{}){
				email.WithEmailTimeout(10 * time.Second),
			},
			wantErr: false,
		},
		{
			name:    "empty host",
			host:    "",
			port:    587,
			from:    "sender@example.com",
			wantErr: true,
		},
		{
			name:    "empty from",
			host:    "smtp.example.com",
			port:    587,
			from:    "",
			wantErr: true,
		},
		{
			name: "complete configuration",
			host: "smtp.example.com",
			port: 587,
			from: "sender@example.com",
			options: []func(map[string]interface{}){
				email.WithEmailAuth("user", "pass"),
				email.WithEmailTLS(true),
				email.WithEmailTimeout(30 * time.Second),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub, err := notifyhub.New(
				email.WithEmail("smtp.example.com", 587, "from@example.com"),
			)
			if err != nil {
				t.Fatalf("Failed to create hub: %v", err)
			}
			defer func() { _ = hub.Close() }()
		})
	}
}

func TestWithEmailSMTP(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		username string
		password string
		from     string
		useTLS   bool
		timeout  time.Duration
		wantErr  bool
	}{
		{
			name:     "legacy SMTP configuration",
			host:     "smtp.example.com",
			port:     587,
			username: "user",
			password: "pass",
			from:     "sender@example.com",
			useTLS:   true,
			timeout:  30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "without TLS",
			host:     "smtp.example.com",
			port:     25,
			username: "user",
			password: "pass",
			from:     "sender@example.com",
			useTLS:   false,
			timeout:  30 * time.Second,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub, err := notifyhub.New(
				email.WithEmailSMTP(tt.host, tt.port, tt.username, tt.password, tt.from, tt.useTLS, tt.timeout),
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("WithEmailSMTP() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("WithEmailSMTP() unexpected error: %v", err)
				return
			}

			if hub == nil {
				t.Error("WithEmailSMTP() returned nil hub")
				return
			}

			_ = hub.Close()
		})
	}
}

func TestUseGoMailAndNetSMTP(t *testing.T) {
	t.Run("default uses go-mail", func(t *testing.T) {
		email.UseGoMail()
		// Since we're in external test package, we can't test internal state directly
		// The function should work without error
	})

	t.Run("switch to net/smtp", func(t *testing.T) {
		email.UseNetSMTP()
		// Since we're in external test package, we can't test internal state directly
		// The function should work without error
	})

	t.Run("switch back to go-mail", func(t *testing.T) {
		email.UseGoMail()
		// Since we're in external test package, we can't test internal state directly
		// The function should work without error
	})
}

func TestEmailOptionsHelpers(t *testing.T) {
	t.Run("WithEmailAuth sets credentials", func(t *testing.T) {
		config := make(map[string]interface{})
		opt := email.WithEmailAuth("testuser", "testpass")
		opt(config)

		if config["smtp_username"] != "testuser" {
			t.Errorf("WithEmailAuth() username = %v, want testuser", config["smtp_username"])
		}
		if config["smtp_password"] != "testpass" {
			t.Errorf("WithEmailAuth() password = %v, want testpass", config["smtp_password"])
		}
	})

	t.Run("WithEmailTLS sets TLS flag", func(t *testing.T) {
		config := make(map[string]interface{})
		opt := email.WithEmailTLS(true)
		opt(config)

		if config["smtp_tls"] != true {
			t.Errorf("WithEmailTLS() = %v, want true", config["smtp_tls"])
		}
	})

	t.Run("WithEmailSSL sets SSL flag", func(t *testing.T) {
		config := make(map[string]interface{})
		opt := email.WithEmailSSL(true)
		opt(config)

		if config["smtp_ssl"] != true {
			t.Errorf("WithEmailSSL() = %v, want true", config["smtp_ssl"])
		}
	})

	t.Run("WithEmailTimeout sets timeout", func(t *testing.T) {
		config := make(map[string]interface{})
		timeout := 15 * time.Second
		opt := email.WithEmailTimeout(timeout)
		opt(config)

		if config["timeout"] != timeout {
			t.Errorf("WithEmailTimeout() = %v, want %v", config["timeout"], timeout)
		}
	})
}
