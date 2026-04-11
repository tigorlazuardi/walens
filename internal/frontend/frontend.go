package frontend

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/olivere/vite"
)

// spaHandler handles SPA fallback with backend-owned HTML shell and Vite asset integration.
type spaHandler struct {
	basePath      string
	apiBase       string
	devMode       bool
	viteURL       string
	fragment      *vite.Fragment
	staticFS      fs.FS
	assetsHandler http.Handler
}

// NewSPAHandler creates a new SPA handler.
// If devMode is true, it uses olivere/vite HTMLFragment for development with viteURL.
// Otherwise it uses the static file system for production assets.
// If staticFS is nil and devMode is false, returns an error.
func NewSPAHandler(basePath, viteURL string, devMode bool, staticFS fs.FS) (*spaHandler, error) {
	return NewSPAHandlerWithEntry(basePath, viteURL, devMode, staticFS, "src/main.js")
}

// NewSPAHandlerWithEntry creates a new SPA handler with a specific Vite entry point.
// If devMode is true, it uses olivere/vite HTMLFragment for development with viteURL.
// Otherwise it uses the static file system for production assets.
// If staticFS is nil and devMode is false, returns an error.
func NewSPAHandlerWithEntry(basePath, viteURL string, devMode bool, staticFS fs.FS, viteEntry string) (*spaHandler, error) {
	// In production mode, we need a valid filesystem
	if !devMode && staticFS == nil {
		return nil, fmt.Errorf("static FS is required in production mode")
	}

	sh := &spaHandler{
		basePath: basePath,
		apiBase:  path.Join(basePath, "api"),
		devMode:  devMode,
		viteURL:  viteURL,
		staticFS: staticFS,
	}

	// Create Vite fragment for JS/CSS tags
	manifestPath := ".vite/manifest.json"
	if devMode {
		manifestPath = "" // Dev mode doesn't need manifest
	}

	viteConfig := vite.Config{
		FS:              staticFS,
		IsDev:           devMode,
		ViteURL:         viteURL,
		ViteManifest:    manifestPath,
		AssetsURLPrefix: path.Join(basePath, "assets"),
		ViteEntry:       viteEntry,
	}

	frag, err := vite.HTMLFragment(viteConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vite fragment: %w", err)
	}
	sh.fragment = frag

	// Set up assets handler for production mode
	if !devMode && staticFS != nil {
		sh.assetsHandler = http.FileServer(http.FS(staticFS))
	}

	return sh, nil
}

// walensConfig holds the runtime configuration injected into the SPA shell.
type walensConfig struct {
	BasePath string `json:"basePath"`
	APIBase  string `json:"apiBase"`
}

// serveShell renders the backend-owned HTML shell with runtime config injection.
func (sh *spaHandler) serveShell(w http.ResponseWriter, r *http.Request) {
	cfg := walensConfig{
		BasePath: sh.basePath,
		APIBase:  sh.apiBase,
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		http.Error(w, "failed to encode config", http.StatusInternalServerError)
		return
	}

	// Build HTML shell with window.__WALENS__ injection
	// The fragment tags contain the Vite JS/CSS script tags
	htmlTemplate := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Walens</title>
  <script type="application/json" id="__WALENS_CONFIG__">%s</script>
  <script>
    window.__WALENS__ = JSON.parse(document.getElementById('__WALENS_CONFIG__').textContent);
  </script>
  %s
</head>
<body>
  <div id="app"></div>
</body>
</html>`, string(cfgJSON), string(sh.fragment.Tags))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlTemplate))
}

// ServeHTTP handles HTTP requests for SPA fallback and Vite asset serving.
func (sh *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Path

	// Never intercept API routes
	if strings.HasPrefix(reqPath, sh.apiBase) {
		http.NotFound(w, r)
		return
	}

	// Never intercept OpenAPI/docs/schemas routes
	if strings.HasPrefix(reqPath, path.Join(sh.basePath, "openapi")) ||
		strings.HasPrefix(reqPath, path.Join(sh.basePath, "docs")) ||
		strings.HasPrefix(reqPath, path.Join(sh.basePath, "schemas")) {
		http.NotFound(w, r)
		return
	}

	// Handle health route - let it pass through (404 from mux)
	if reqPath == path.Join(sh.basePath, "health") {
		http.NotFound(w, r)
		return
	}

	// Handle login/logout routes
	loginPath := path.Join(sh.apiBase, "login")
	logoutPath := path.Join(sh.apiBase, "logout")
	if reqPath == loginPath || reqPath == logoutPath {
		http.NotFound(w, r)
		return
	}

	// Handle assets in production mode
	if !sh.devMode && sh.assetsHandler != nil {
		assetsPath := path.Join(sh.basePath, "assets")
		if strings.HasPrefix(reqPath, assetsPath) {
			// Strip the basePath prefix for the assets handler
			assetReq := r.WithContext(r.Context())
			assetReq.URL.Path = strings.TrimPrefix(reqPath, sh.basePath)
			sh.assetsHandler.ServeHTTP(w, assetReq)
			return
		}
	}

	// For all SPA routes, serve the HTML shell with config injection
	sh.serveShell(w, r)
}

// SetAssetFS sets the embed.FS for production assets (used with go:embed).
func (sh *spaHandler) SetAssetFS(fs embed.FS) {
	// Not needed with current implementation
}

// DevMode returns whether the handler is in development mode.
func (sh *spaHandler) DevMode() bool {
	return sh.devMode
}

// BasePath returns the configured base path.
func (sh *spaHandler) BasePath() string {
	return sh.basePath
}

// IsAssetPath checks if a request path is for a static asset.
func IsAssetPath(p string) bool {
	if strings.HasPrefix(p, "/assets/") {
		return true
	}
	ext := path.Ext(p)
	return ext == ".js" || ext == ".css" || ext == ".ico" || ext == ".png" ||
		ext == ".jpg" || ext == ".jpeg" || ext == ".svg" || ext == ".woff" ||
		ext == ".woff2" || ext == ".ttf" || ext == ".eot" || ext == ".map" ||
		ext == ".webp" || ext == ".gif" || ext == ".avif"
}

// EscapeForJS escapes a string for safe injection into a JavaScript string literal.
// Deprecated: Use json.Marshal instead for safe injection.
func EscapeForJS(s string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, c := range s {
		switch c {
		case '\\':
			builder.WriteString("\\\\")
		case '"':
			builder.WriteString("\\\"")
		case '\n':
			builder.WriteString("\\n")
		case '\r':
			builder.WriteString("\\r")
		case '\t':
			builder.WriteString("\\t")
		default:
			builder.WriteRune(c)
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

// serveSPAWithConfig serves the SPA shell with runtime configuration injection.
// Deprecated: Use spaHandler.serveShell instead.
func ServeSPAWithConfig(w http.ResponseWriter, r *http.Request, basePath, apiBase, entryPoint string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl := `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Walens</title>
    <script>
      window.__WALENS__ = {
        basePath: %s,
        apiBase: %s,
      };
    </script>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="%s"></script>
  </body>
</html>`

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, tmpl,
		EscapeForJS(basePath),
		EscapeForJS(apiBase),
		EscapeForJS(entryPoint),
	)
}

// SPAFileServer creates an http.Handler that serves SPA fallback from a filesystem.
// Deprecated: Use NewSPAHandler instead.
func SPAFileServer(basePath string, staticFS fs.FS) http.Handler {
	return &spaFileServer{
		basePath: basePath,
		fs:       staticFS,
	}
}

type spaFileServer struct {
	basePath string
	fs       fs.FS
}

func (s *spaFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
