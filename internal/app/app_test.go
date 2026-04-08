package app

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/walens/walens/internal/auth"
	"github.com/walens/walens/internal/config"
	"github.com/walens/walens/internal/db"
	"github.com/walens/walens/internal/queue"
	"github.com/walens/walens/internal/runner"
	"github.com/walens/walens/internal/scheduler"
	"github.com/walens/walens/internal/services/configs"

	_ "modernc.org/sqlite"
)

const testCookieSecret = "test-cookie-secret"

func TestAuthConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.AuthConfig
		wantErr bool
	}{
		{
			name: "auth disabled with empty credentials is valid",
			cfg: config.AuthConfig{
				Enabled:  false,
				Username: "",
				Password: "",
			},
			wantErr: false,
		},
		{
			name: "auth disabled with credentials is valid",
			cfg: config.AuthConfig{
				Enabled:  false,
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name: "auth enabled with valid credentials is valid",
			cfg: config.AuthConfig{
				Enabled:      true,
				Username:     "user",
				Password:     "pass",
				CookieSecret: testCookieSecret,
			},
			wantErr: false,
		},
		{
			name: "auth enabled with empty username is invalid",
			cfg: config.AuthConfig{
				Enabled:      true,
				Username:     "",
				Password:     "pass",
				CookieSecret: testCookieSecret,
			},
			wantErr: true,
		},
		{
			name: "auth enabled with empty password is invalid",
			cfg: config.AuthConfig{
				Enabled:      true,
				Username:     "user",
				Password:     "",
				CookieSecret: testCookieSecret,
			},
			wantErr: true,
		},
		{
			name: "auth enabled with both empty is invalid",
			cfg: config.AuthConfig{
				Enabled:      true,
				Username:     "",
				Password:     "",
				CookieSecret: testCookieSecret,
			},
			wantErr: true,
		},
		{
			name: "auth enabled with empty cookie secret is invalid",
			cfg: config.AuthConfig{
				Enabled:      true,
				Username:     "user",
				Password:     "pass",
				CookieSecret: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthAuthorize(t *testing.T) {
	authCfg := auth.Config{
		Enabled:      true,
		Username:     "testuser",
		Password:     "testpass",
		CookieSecret: testCookieSecret,
	}

	disabledAuthCfg := auth.Config{
		Enabled: false,
	}

	tests := []struct {
		name     string
		authCfg  auth.Config
		setupReq func(r *http.Request)
		wantErr  error
	}{
		{
			name:    "auth disabled allows any request",
			authCfg: disabledAuthCfg,
			setupReq: func(r *http.Request) {
				// No auth
			},
			wantErr: nil,
		},
		{
			name:    "auth enabled rejects missing credentials",
			authCfg: authCfg,
			setupReq: func(r *http.Request) {
				// No auth header, no cookie
			},
			wantErr: auth.ErrMissingCredentials,
		},
		{
			name:    "auth enabled accepts valid basic auth header",
			authCfg: authCfg,
			setupReq: func(r *http.Request) {
				creds := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
				r.Header.Set("Authorization", "Basic "+creds)
			},
			wantErr: nil,
		},
		{
			name:    "auth enabled rejects invalid basic auth header - wrong user",
			authCfg: authCfg,
			setupReq: func(r *http.Request) {
				creds := base64.StdEncoding.EncodeToString([]byte("wronguser:testpass"))
				r.Header.Set("Authorization", "Basic "+creds)
			},
			wantErr: auth.ErrInvalidCredentials,
		},
		{
			name:    "auth enabled rejects invalid basic auth header - wrong pass",
			authCfg: authCfg,
			setupReq: func(r *http.Request) {
				creds := base64.StdEncoding.EncodeToString([]byte("testuser:wrongpass"))
				r.Header.Set("Authorization", "Basic "+creds)
			},
			wantErr: auth.ErrInvalidCredentials,
		},
		{
			name:    "auth enabled rejects invalid basic auth header - malformed",
			authCfg: authCfg,
			setupReq: func(r *http.Request) {
				creds := base64.StdEncoding.EncodeToString([]byte("notvalidbase64"))
				r.Header.Set("Authorization", "Basic "+creds)
			},
			wantErr: auth.ErrInvalidCredentials,
		},
		{
			name:    "auth enabled rejects non-basic auth header",
			authCfg: authCfg,
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer sometoken")
			},
			wantErr: auth.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupReq(req)

			err := tt.authCfg.Authorize(req)
			if err != tt.wantErr {
				t.Errorf("AuthConfig.Authorize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthCookieValidation(t *testing.T) {
	authCfg := auth.Config{
		Enabled:      true,
		Username:     "testuser",
		Password:     "testpass",
		CookieSecret: testCookieSecret,
	}

	// Create a valid cookie value
	validCookieValue, _ := auth.BuildCookieValue(testCookieSecret, "testuser", "testpass")
	wrongUserCookieValue, _ := auth.BuildCookieValue(testCookieSecret, "wronguser", "testpass")
	wrongPassCookieValue, _ := auth.BuildCookieValue(testCookieSecret, "testuser", "wrongpass")

	tests := []struct {
		name      string
		cookieVal string
		wantErr   bool
	}{
		{
			name:      "valid cookie",
			cookieVal: validCookieValue,
			wantErr:   false,
		},
		{
			name:      "cookie with wrong username",
			cookieVal: wrongUserCookieValue,
			wantErr:   true,
		},
		{
			name:      "cookie with wrong password",
			cookieVal: wrongPassCookieValue,
			wantErr:   true,
		},
		{
			name:      "empty cookie",
			cookieVal: "",
			wantErr:   true,
		},
		{
			name:      "malformed cookie",
			cookieVal: "not.valid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authCfg.ValidateCookieValue(tt.cookieVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCookieValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAppWithAuth tests the full HTTP auth flow with a minimal app setup.
func TestAppWithAuth(t *testing.T) {
	// Create a minimal test database
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	// Create minimal app configuration with auth enabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:     "localhost",
			Port:     0, // use random port
			BasePath: "/",
		},
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		Auth: config.AuthConfig{
			Enabled:      true,
			Username:     "testuser",
			Password:     "testpass",
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "error",
	}

	// Create app with minimal overrides for testing
	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
	}

	handler := app.Handler()

	tests := []struct {
		name           string
		method         string
		path           string
		setupReq       func(r *http.Request)
		expectedStatus int
		clearCookie    bool
	}{
		{
			name:   "health endpoint is public",
			method: http.MethodGet,
			path:   "/health",
			setupReq: func(r *http.Request) {
				// No auth
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "missing auth is rejected for protected route",
			method: http.MethodGet,
			path:   "/",
			setupReq: func(r *http.Request) {
				// No auth
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "valid basic auth reaches protected router layer",
			method: http.MethodGet,
			path:   "/",
			setupReq: func(r *http.Request) {
				creds := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
				r.Header.Set("Authorization", "Basic "+creds)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "invalid basic auth is rejected",
			method: http.MethodGet,
			path:   "/",
			setupReq: func(r *http.Request) {
				creds := base64.StdEncoding.EncodeToString([]byte("wronguser:wrongpass"))
				r.Header.Set("Authorization", "Basic "+creds)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "logout clears cookie",
			method: http.MethodPost,
			path:   "/api/logout",
			setupReq: func(r *http.Request) {
				// Set a valid auth cookie
				cookieValue, _ := auth.BuildCookieValue(testCookieSecret, "testuser", "testpass")
				r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: cookieValue})
			},
			expectedStatus: http.StatusNoContent,
			clearCookie:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			tt.setupReq(req)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.clearCookie {
				cookies := rec.Result().Cookies()
				found := false
				for _, c := range cookies {
					if c.Name == auth.CookieName && c.Value == "" && c.MaxAge == 0 {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected auth cookie to be cleared")
				}
			}
		})
	}
}

// TestLoginFlow tests the login form submission and cookie setting.
func TestLoginFlow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:     "localhost",
			Port:     0,
			BasePath: "/",
		},
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		Auth: config.AuthConfig{
			Enabled:      true,
			Username:     "testuser",
			Password:     "testpass",
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "error",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
	}

	handler := app.Handler()

	// Login with valid credentials
	t.Run("valid login sets cookie", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"username": "testuser", "password": "testpass"})
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}

		// Check cookie was set
		cookies := rec.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == auth.CookieName && c.HttpOnly && c.Secure == false && c.Value != "" && c.Path == "/" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected auth cookie to be set")
		}
	})

	// Login with invalid credentials
	t.Run("invalid login returns 401", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"username": "wronguser", "password": "wrongpass"})
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("cookie allows access after login", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"username": "testuser", "password": "testpass"})
		loginReq := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(string(body)))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRec := httptest.NewRecorder()
		handler.ServeHTTP(loginRec, loginReq)

		resp := loginRec.Result()
		var authCookie *http.Cookie
		for _, c := range resp.Cookies() {
			if c.Name == auth.CookieName {
				authCookie = c
				break
			}
		}
		if authCookie == nil {
			t.Fatalf("expected auth cookie to be set")
		}

		homeReq := httptest.NewRequest(http.MethodGet, "/", nil)
		homeReq.AddCookie(authCookie)
		homeRec := httptest.NewRecorder()
		handler.ServeHTTP(homeRec, homeReq)

		if homeRec.Code != http.StatusNotFound {
			t.Fatalf("expected cookie-authenticated request to reach router, got %d", homeRec.Code)
		}
	})
}

// TestAuthDisabledAllowAccess tests that when auth is disabled, all routes are accessible.
func TestAuthDisabledAllowAccess(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:     "localhost",
			Port:     0,
			BasePath: "/",
		},
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		Auth: config.AuthConfig{
			Enabled:  false,
			Username: "",
			Password: "",
		},
		DataDir:  tmpDir,
		LogLevel: "error",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
	}

	handler := app.Handler()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "missing frontend route stays 404 when auth disabled",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "health accessible without auth when disabled",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

// TestInvalidHeaderDoesNotFallbackToCookie tests that an invalid Authorization header
// does not fall back to validating a cookie.
func TestInvalidHeaderDoesNotFallbackToCookie(t *testing.T) {
	authCfg := auth.Config{
		Enabled:      true,
		Username:     "testuser",
		Password:     "testpass",
		CookieSecret: testCookieSecret,
	}

	// Create a valid cookie
	validCookie, _ := auth.BuildCookieValue(testCookieSecret, "testuser", "testpass")

	// Make request with invalid Basic auth header AND valid cookie
	// The auth should fail because the header is checked first and is invalid
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	creds := base64.StdEncoding.EncodeToString([]byte("invalid:credentials"))
	req.Header.Set("Authorization", "Basic "+creds)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: validCookie})

	err := authCfg.Authorize(req)
	if err != auth.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials when header is invalid, got %v", err)
	}
}

// TestCookieAllowsAccessAfterInvalidHeader tests that a valid cookie allows access
// when there's no Authorization header.
func TestCookieAllowsAccessAfterNoHeader(t *testing.T) {
	authCfg := auth.Config{
		Enabled:      true,
		Username:     "testuser",
		Password:     "testpass",
		CookieSecret: testCookieSecret,
	}

	// Create a valid cookie
	validCookie, _ := auth.BuildCookieValue(testCookieSecret, "testuser", "testpass")

	// Make request with valid cookie but no Authorization header
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: validCookie})

	err := authCfg.Authorize(req)
	if err != nil {
		t.Errorf("expected no error with valid cookie, got %v", err)
	}
}

// TestBasePathRoutes tests that routes work correctly with a non-root base path.
func TestBasePathRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:     "localhost",
			Port:     0,
			BasePath: "/walens",
		},
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		Auth: config.AuthConfig{
			Enabled:      true,
			Username:     "testuser",
			Password:     "testpass",
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "error",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
	}

	handler := app.Handler()

	tests := []struct {
		name           string
		method         string
		path           string
		setupReq       func(r *http.Request)
		expectedStatus int
	}{
		{
			name:   "health at /walens/health is public",
			method: http.MethodGet,
			path:   "/walens/health",
			setupReq: func(r *http.Request) {
				// No auth
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "protected route at /walens requires auth",
			method: http.MethodGet,
			path:   "/walens",
			setupReq: func(r *http.Request) {
				// No auth
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "valid auth at /walens reaches router layer",
			method: http.MethodGet,
			path:   "/walens",
			setupReq: func(r *http.Request) {
				creds := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
				r.Header.Set("Authorization", "Basic "+creds)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "login endpoint under base path is public",
			method: http.MethodPost,
			path:   "/walens/api/login",
			setupReq: func(r *http.Request) {
				body := `{"username":"testuser","password":"testpass"}`
				r.Body = io.NopCloser(strings.NewReader(body))
				r.Header.Set("Content-Type", "application/json")
				r.ContentLength = int64(len(body))
			},
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			tt.setupReq(req)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

// TestInitDBAppliesPersistedConfig tests that initDB loads persisted config and
// applies it back to the active app config.
func TestInitDBAppliesPersistedConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	// Run migrations first
	if err := db.RunMigrations(testDB); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Insert a custom persisted config
	customCfg := &configs.PersistedConfig{
		Server: configs.PersistedServerConfig{
			BasePath: "/custom-path",
		},
		DataDir:  "/custom/data",
		LogLevel: "debug",
	}
	customBytes, _ := json.Marshal(customCfg)
	_, err = testDB.Exec(`UPDATE configs SET value = ?, updated_at = ? WHERE id = 1`, string(customBytes), 1000)
	if err != nil {
		t.Fatalf("insert custom config: %v", err)
	}

	// Create bootstrap config
	bootstrapCfg := &config.Config{
		Server: config.ServerConfig{
			Host:     "localhost",
			Port:     9999,
			BasePath: "/",
		},
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		Auth: config.AuthConfig{
			Enabled:      false,
			CookieSecret: "test-secret",
		},
		DataDir:  "./default-data",
		LogLevel: "info",
	}

	// Create app and call initDB
	app := &App{
		config: bootstrapCfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: runner.New(slog.Default()),
	}

	// Set up a minimal scheduler for initDB
	app.scheduler = scheduler.New(slog.Default())
	app.runner.SetQueue(app.queue)

	if err := app.initDB(); err != nil {
		t.Fatalf("initDB failed: %v", err)
	}

	// Verify persisted config was applied to active config
	if app.config.Server.BasePath != "/custom-path" {
		t.Errorf("expected BasePath '/custom-path', got: %q", app.config.Server.BasePath)
	}
	if app.config.DataDir != "/custom/data" {
		t.Errorf("expected DataDir '/custom/data', got: %q", app.config.DataDir)
	}
	if app.config.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got: %q", app.config.LogLevel)
	}

	// Verify bootstrap-only fields are preserved
	if app.config.Server.Host != "localhost" {
		t.Errorf("expected Host 'localhost' to be preserved, got: %q", app.config.Server.Host)
	}
	if app.config.Server.Port != 9999 {
		t.Errorf("expected Port 9999 to be preserved, got: %d", app.config.Server.Port)
	}
	if app.config.Database.Path != dbPath {
		t.Errorf("expected Database.Path to be preserved, got: %q", app.config.Database.Path)
	}
}

// TestInitDBInjectsDefaultsForEmptyRow tests that initDB injects defaults
// when the configs row is empty.
func TestInitDBInjectsDefaultsForEmptyRow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	// Run migrations (this creates configs table with empty '{}' row)
	if err := db.RunMigrations(testDB); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Create bootstrap config with specific defaults
	bootstrapCfg := &config.Config{
		Server: config.ServerConfig{
			Host:     "0.0.0.0",
			Port:     8080,
			BasePath: "/default",
		},
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		Auth: config.AuthConfig{
			Enabled:      false,
			CookieSecret: "test-secret",
		},
		DataDir:  "./default-data",
		LogLevel: "info",
	}

	// Create app and call initDB
	app := &App{
		config: bootstrapCfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: runner.New(slog.Default()),
	}

	// Set up a minimal scheduler for initDB
	app.scheduler = scheduler.New(slog.Default())
	app.runner.SetQueue(app.queue)

	if err := app.initDB(); err != nil {
		t.Fatalf("initDB failed: %v", err)
	}

	// Verify defaults were injected and applied
	if app.config.Server.BasePath != "/default" {
		t.Errorf("expected BasePath '/default', got: %q", app.config.Server.BasePath)
	}
	if app.config.DataDir != "./default-data" {
		t.Errorf("expected DataDir './default-data', got: %q", app.config.DataDir)
	}
	if app.config.LogLevel != "info" {
		t.Errorf("expected LogLevel 'info', got: %q", app.config.LogLevel)
	}

	// Verify the persisted config row was updated with defaults
	var value string
	err = testDB.QueryRow(`SELECT value FROM configs WHERE id = 1`).Scan(&value)
	if err != nil {
		t.Fatalf("query persisted config: %v", err)
	}

	var stored configs.PersistedConfig
	if err := json.Unmarshal([]byte(value), &stored); err != nil {
		t.Fatalf("unmarshal stored config: %v", err)
	}
	if stored.Server.BasePath != "/default" {
		t.Errorf("expected stored BasePath '/default', got: %q", stored.Server.BasePath)
	}
	if stored.DataDir != "./default-data" {
		t.Errorf("expected stored DataDir './default-data', got: %q", stored.DataDir)
	}
}

// TestInitDBPreservesAuthBootstrapOnly tests that auth config is NOT loaded
// from persisted config (it's bootstrap-only).
func TestInitDBPreservesAuthBootstrapOnly(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	// Run migrations
	if err := db.RunMigrations(testDB); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Create bootstrap config
	bootstrapCfg := &config.Config{
		Server: config.ServerConfig{
			Host:     "0.0.0.0",
			Port:     8080,
			BasePath: "/",
		},
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		Auth: config.AuthConfig{
			Enabled:      true,
			Username:     "bootstrap-user",
			Password:     "bootstrap-pass",
			CookieSecret: "bootstrap-secret",
		},
		DataDir:  "./data",
		LogLevel: "info",
	}

	// Create app and call initDB
	app := &App{
		config: bootstrapCfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: runner.New(slog.Default()),
	}

	// Set up a minimal scheduler for initDB
	app.scheduler = scheduler.New(slog.Default())
	app.runner.SetQueue(app.queue)

	if err := app.initDB(); err != nil {
		t.Fatalf("initDB failed: %v", err)
	}

	// Verify auth fields are preserved from bootstrap config, not from persisted config
	if app.config.Auth.Username != "bootstrap-user" {
		t.Errorf("expected Auth.Username 'bootstrap-user', got: %q", app.config.Auth.Username)
	}
	if app.config.Auth.Password != "bootstrap-pass" {
		t.Errorf("expected Auth.Password 'bootstrap-pass', got: %q", app.config.Auth.Password)
	}
	if app.config.Auth.Enabled != true {
		t.Errorf("expected Auth.Enabled true, got: %v", app.config.Auth.Enabled)
	}
}
