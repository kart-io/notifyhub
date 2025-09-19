package middleware

import "net/http"

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-API-Key"},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a CORS middleware
func CORS(config *CORSConfig) func(http.HandlerFunc) http.HandlerFunc {
	if config == nil {
		config = DefaultCORSConfig()
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			if len(config.AllowOrigins) > 0 {
				origin := r.Header.Get("Origin")
				if origin != "" && contains(config.AllowOrigins, origin) || contains(config.AllowOrigins, "*") {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}

			if len(config.AllowMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", joinStrings(config.AllowMethods, ", "))
			}

			if len(config.AllowHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", joinStrings(config.AllowHeaders, ", "))
			}

			if len(config.ExposeHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", joinStrings(config.ExposeHeaders, ", "))
			}

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next(w, r)
		}
	}
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func joinStrings(slice []string, sep string) string {
	if len(slice) == 0 {
		return ""
	}
	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += sep + slice[i]
	}
	return result
}
