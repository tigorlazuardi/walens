package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
)

const (
	// CookieName is the name of the auth cookie.
	CookieName   = "walens_auth"
	cookieMaxAge = 60 * 60 * 24 * 7 // 1 week
	cookieScope  = "walens-auth"
)

var errInvalidCookieValue = errors.New("invalid cookie value")

// BuildCookieValue creates an encrypted cookie value from credentials.
func BuildCookieValue(secret, username, password string) (string, error) {
	plaintext := []byte(username + ":" + password)

	block, err := aes.NewCipher(deriveCookieKey(secret))
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, []byte(cookieScope))
	payload := append(nonce, ciphertext...)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// ValidateCookieValue checks if a cookie value is valid for the given credentials.
func (c Config) ValidateCookieValue(cookieValue string) error {
	return c.validateCookieValue(cookieValue)
}

// validateCookieValue checks if a cookie value matches the configured credentials.
func (c Config) validateCookieValue(cookieValue string) error {
	plaintext, err := decryptCookieValue(c.CookieSecret, cookieValue)
	if err != nil {
		return ErrInvalidCredentials
	}

	decodedParts := strings.SplitN(string(plaintext), ":", 2)
	if len(decodedParts) != 2 {
		return ErrInvalidCredentials
	}

	return c.validateCredentials(decodedParts[0], decodedParts[1])
}

func decryptCookieValue(secret, cookieValue string) ([]byte, error) {
	payload, err := base64.RawURLEncoding.DecodeString(cookieValue)
	if err != nil {
		return nil, errInvalidCookieValue
	}

	block, err := aes.NewCipher(deriveCookieKey(secret))
	if err != nil {
		return nil, errInvalidCookieValue
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errInvalidCookieValue
	}

	if len(payload) < aead.NonceSize() {
		return nil, errInvalidCookieValue
	}

	nonce := payload[:aead.NonceSize()]
	ciphertext := payload[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, []byte(cookieScope))
	if err != nil {
		return nil, errInvalidCookieValue
	}

	return plaintext, nil
}

func deriveCookieKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

// CookieOptions holds options for cookie creation.
type CookieOptions struct {
	Secure   bool
	SameSite http.SameSite
	Path     string
}

// DefaultCookieOptions returns default cookie options from explicit config.
func DefaultCookieOptions(basePath string, secure bool) CookieOptions {
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
func SetAuthCookie(w http.ResponseWriter, basePath string, secure bool, value string) {
	opts := DefaultCookieOptions(basePath, secure)
	cookie := NewAuthCookie(value, opts)
	http.SetCookie(w, cookie)
}

// ClearAuthCookieHandler clears the auth cookie.
func ClearAuthCookieHandler(w http.ResponseWriter, r *http.Request, basePath string) {
	cookie := ClearAuthCookie(basePath)
	http.SetCookie(w, cookie)
}
