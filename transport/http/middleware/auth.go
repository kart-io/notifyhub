package middleware

import (
	"net/http"
	"strings"

	"github.com/kart-io/notifyhub/api"
)

// AuthMiddleware provides authentication middleware for HTTP transport
type AuthMiddleware struct {
	hub        *api.NotifyHub
	apiKeys    map[string]bool
	bearerKeys map[string]bool
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(hub *api.NotifyHub) *AuthMiddleware {
	return &AuthMiddleware{
		hub:        hub,
		apiKeys:    make(map[string]bool),
		bearerKeys: make(map[string]bool),
	}
}

// AddAPIKey adds an allowed API key
func (a *AuthMiddleware) AddAPIKey(key string) {
	a.apiKeys[key] = true
}

// AddBearerToken adds an allowed bearer token
func (a *AuthMiddleware) AddBearerToken(token string) {
	a.bearerKeys[token] = true
}

// Middleware returns the HTTP middleware function
func (a *AuthMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check API key in header
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			if a.apiKeys[apiKey] {
				next(w, r)
				return
			}
		}

		// Check bearer token
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if a.bearerKeys[token] {
					next(w, r)
					return
				}
			}
		}

		// Check basic auth (could be added)
		// if username, password, ok := r.BasicAuth(); ok {
		//     if a.validateBasicAuth(username, password) {
		//         next(w, r)
		//         return
		//     }
		// }

		// Unauthorized
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized", "message": "valid authentication required"}`))
	}
}

// NoAuth returns a middleware that allows all requests (for development)
func NoAuth() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return next
	}
}
