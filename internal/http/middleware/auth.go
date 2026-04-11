package middleware

import (
	"net/http"
	"strings"

	"github.com/walens/walens/internal/auth"
)

// NewAuth returns HTTP middleware that enforces bootstrap auth.
func NewAuth(cfg auth.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := cfg.Authorize(r); err != nil {
				if !isPublicRoute(cfg, r) {
					w.Header().Set("WWW-Authenticate", `Basic realm="Walens"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isPublicRoute(cfg auth.Config, r *http.Request) bool {
	path := r.URL.Path
	basePath := strings.TrimSuffix(cfg.BasePath, "/")
	if basePath != "" && basePath != "/" {
		path = strings.TrimPrefix(path, basePath)
	}

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
