package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

// ErrAuthDisabled is returned when auth validation is attempted but auth is disabled.
var ErrAuthDisabled = errors.New("auth is disabled")

// ErrInvalidCredentials is returned when provided credentials do not match.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrMissingCredentials is returned when no credentials are provided.
var ErrMissingCredentials = errors.New("missing credentials")

// ErrAuthEnabledButNotConfigured is returned when auth is enabled but username or password is empty.
var ErrAuthEnabledButNotConfigured = errors.New("auth is enabled but username or password is not configured")

// Config holds authentication configuration.
type Config struct {
	Enabled  bool
	Username string
	Password string
	BasePath string // e.g., "/" or "/walens"
}

// Validate checks that if auth is enabled, username and password are non-empty.
func (c Config) Validate() error {
	if c.Enabled {
		if c.Username == "" || c.Password == "" {
			return ErrAuthEnabledButNotConfigured
		}
	}
	return nil
}

// ValidateCredentials validates a provided username and password pair.
func (c Config) ValidateCredentials(username, password string) error {
	return c.validateCredentials(username, password)
}

// Authorize validates credentials against the configured username and password.
// It returns nil on success, or an error on failure.
// The order of checking is:
// 1. Authorization header (Basic auth)
// 2. Auth cookie
// If auth is disabled, it returns nil without checking credentials.
func (c Config) Authorize(r *http.Request) error {
	if !c.Enabled {
		return nil
	}

	// Try Authorization header first
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Basic ") {
			return c.validateBasicHeader(authHeader)
		}
		// If header is present but not Basic, it's invalid
		return ErrInvalidCredentials
	}

	// Try cookie
	if cookie, err := r.Cookie(CookieName); err == nil && cookie != nil {
		return c.validateCookieValue(cookie.Value)
	}

	return ErrMissingCredentials
}

// validateBasicHeader validates a Basic auth header value.
func (c Config) validateBasicHeader(header string) error {
	encoded := strings.TrimPrefix(header, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ErrInvalidCredentials
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return ErrInvalidCredentials
	}
	return c.validateCredentials(parts[0], parts[1])
}

// validateCredentials checks username and password using constant-time comparison.
func (c Config) validateCredentials(username, password string) error {
	if subtle.ConstantTimeCompare([]byte(username), []byte(c.Username)) != 1 {
		return ErrInvalidCredentials
	}
	if subtle.ConstantTimeCompare([]byte(password), []byte(c.Password)) != 1 {
		return ErrInvalidCredentials
	}
	return nil
}

// Middleware returns an HTTP middleware that enforces authentication.
// It does not protect login and static asset paths.
func (c Config) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := c.Authorize(r); err != nil {
				// Only send 401 for protected routes; allow login page to render
				if !c.isPublicRoute(r) {
					w.Header().Set("WWW-Authenticate", `Basic realm="Walens"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isPublicRoute returns true if the route does not require authentication.
// It strips the base path before checking.
func (c Config) isPublicRoute(r *http.Request) bool {
	path := r.URL.Path
	// Strip base path prefix if present
	basePath := strings.TrimSuffix(c.BasePath, "/")
	if basePath != "" && basePath != "/" {
		path = strings.TrimPrefix(path, basePath)
	}
	// Public routes: login page, health check, docs, openapi
	publicPrefixes := []string{
		"/login",
		"/api/login",
		"/api/logout",
		"/health",
		"/docs",
		"/openapi",
	}
	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
