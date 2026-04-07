package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

const (
	// CookieName is the name of the auth cookie.
	CookieName   = "walens_auth"
	cookieMaxAge = 60 * 60 * 24 * 7 // 1 week
	cookieScope  = "walens-auth"
)

// BuildCookieValue creates a signed cookie value from credentials.
func BuildCookieValue(username, password string) string {
	data := username + ":" + password
	encodedData := base64.URLEncoding.EncodeToString([]byte(data))
	h := hmac.New(sha256.New, []byte(data))
	_, _ = h.Write([]byte(encodedData + ":" + cookieScope))
	sig := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return encodedData + "." + sig
}

// ValidateCookieValue checks if a cookie value is valid for the given credentials.
func (c Config) ValidateCookieValue(cookieValue string) error {
	return c.validateCookieValue(cookieValue)
}

// validateCookieValue checks if a cookie value matches the configured credentials.
func (c Config) validateCookieValue(cookieValue string) error {
	parts := strings.SplitN(cookieValue, ".", 2)
	if len(parts) != 2 {
		return ErrInvalidCredentials
	}

	expectedSig := parts[1]
	data, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrInvalidCredentials
	}

	// Verify signature using the configured credentials as the key
	h := hmac.New(sha256.New, []byte(c.Username+":"+c.Password))
	_, _ = h.Write([]byte(parts[0] + ":" + cookieScope))
	sig := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(expectedSig), []byte(sig)) != 1 {
		return ErrInvalidCredentials
	}

	decodedParts := strings.SplitN(string(data), ":", 2)
	if len(decodedParts) != 2 {
		return ErrInvalidCredentials
	}

	return c.validateCredentials(decodedParts[0], decodedParts[1])
}

// CookieOptions holds options for cookie creation.
type CookieOptions struct {
	Secure   bool
	SameSite http.SameSite
	Path     string
}

// DefaultCookieOptions returns default cookie options suitable for most deployments.
func DefaultCookieOptions(r *http.Request, basePath string) CookieOptions {
	secure := r.TLS != nil
	if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		secure = true
	}
	path := strings.TrimSpace(basePath)
	if path == "" {
		path = "/"
	}
	return CookieOptions{
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Path:     path,
	}
}

// NewAuthCookie creates a new auth cookie with the given value.
func NewAuthCookie(value string, opts CookieOptions) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     opts.Path,
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	}
}

// ClearAuthCookie returns a cookie that clears the auth cookie.
func ClearAuthCookie(basePath string) *http.Cookie {
	path := strings.TrimSpace(basePath)
	if path == "" {
		path = "/"
	}
	return &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     path,
		MaxAge:   0,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// SetAuthCookie sets the auth cookie on the response.
func SetAuthCookie(w http.ResponseWriter, r *http.Request, basePath, value string) {
	opts := DefaultCookieOptions(r, basePath)
	cookie := NewAuthCookie(value, opts)
	http.SetCookie(w, cookie)
}

// ClearAuthCookieHandler clears the auth cookie and redirects to login.
func ClearAuthCookieHandler(w http.ResponseWriter, r *http.Request, basePath string) {
	cookie := ClearAuthCookie(basePath)
	http.SetCookie(w, cookie)
}
