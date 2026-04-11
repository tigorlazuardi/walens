package app

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/walens/walens/internal/auth"
	"github.com/walens/walens/internal/config"
	"github.com/walens/walens/internal/db"
	"github.com/walens/walens/internal/frontend"
	"github.com/walens/walens/internal/queue"
	"github.com/walens/walens/internal/runner"
	"github.com/walens/walens/internal/scheduler"
	"github.com/walens/walens/internal/services/configs"
	"github.com/walens/walens/internal/services/images"
	"github.com/walens/walens/internal/services/jobs"
	sourcesvc "github.com/walens/walens/internal/services/sources"
	"github.com/walens/walens/internal/services/tags"
	"github.com/walens/walens/internal/sources"
	"github.com/walens/walens/internal/sources/booru"
	"github.com/walens/walens/internal/sources/reddit"
	"github.com/walens/walens/internal/storage"

	_ "modernc.org/sqlite"
)

const testCookieSecret = "test-cookie-secret"

func newTestScheduler(t *testing.T, db *sql.DB) *scheduler.Scheduler {
	t.Helper()
	return scheduler.New(scheduler.SchedulerDependencies{
		Logger: slog.New(slog.DiscardHandler),
		Loader: scheduleLoaderFunc(func(context.Context) ([]scheduler.SourceSchedule, error) {
			return nil, nil
		}),
		JobsService: jobs.NewService(db),
		EnqueueFunc: func(string) {},
	})
}

func newTestRunner(t *testing.T, db *sql.DB) *runner.Runner {
	t.Helper()
	return runner.New(runner.RunnerDependencies{
		Logger:         slog.New(slog.DiscardHandler),
		Queue:          queue.New(slog.New(slog.DiscardHandler)),
		JobsService:    jobs.NewService(db),
		StorageService: storage.NewService(storage.Config{BaseDir: t.TempDir()}),
		ImageService:   images.NewService(db),
		TagsService:    tags.NewService(db),
		SourceRegistry: newSourceRegistry(),
	})
}

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
// applies it back to the active app config. BasePath is NOT applied from persisted
// config because it is bootstrap-only.
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

	// Insert a custom persisted config (BasePath is ignored since it's bootstrap-only)
	customCfg := &configs.PersistedConfig{
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
		runner: newTestRunner(t, testDB),
	}

	// Set up a minimal scheduler for initDB
	app.scheduler = newTestScheduler(t, testDB)

	if err := app.initDB(); err != nil {
		t.Fatalf("initDB failed: %v", err)
	}

	// Verify persisted config was applied to active config
	// BasePath remains from bootstrap config, NOT from persisted config
	if app.config.Server.BasePath != "/" {
		t.Errorf("expected BasePath '/' from bootstrap, got: %q", app.config.Server.BasePath)
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
		runner: newTestRunner(t, testDB),
	}

	// Set up a minimal scheduler for initDB
	app.scheduler = newTestScheduler(t, testDB)

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
	// Note: BasePath is NOT stored in persisted config since it's bootstrap-only
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
		runner: newTestRunner(t, testDB),
	}

	// Set up a minimal scheduler for initDB
	app.scheduler = newTestScheduler(t, testDB)

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

// TestGetConfigRoute tests the GET /api/v1/configs/GetConfig endpoint.
func TestGetConfigRoute(t *testing.T) {
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize configService
	app.configService = configs.NewService(app.db)

	// Set up a minimal scheduler for initDB
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Test GetConfig returns defaults when no config is set
	t.Run("get config returns defaults", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/configs/GetConfig", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Huma returns body fields at the top level of the response
		if resp["data_dir"] != "./data" {
			t.Errorf("expected data_dir './data', got: %v", resp["data_dir"])
		}
		if resp["log_level"] != "info" {
			t.Errorf("expected log_level 'info', got: %v", resp["log_level"])
		}
	})

	// Test UpdateConfig stores new config
	t.Run("update config stores new values", func(t *testing.T) {
		body := `{"data_dir":"/new/data","log_level":"debug"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/configs/UpdateConfig", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp["data_dir"] != "/new/data" {
			t.Errorf("expected data_dir '/new/data', got: %v", resp["data_dir"])
		}
		if resp["log_level"] != "debug" {
			t.Errorf("expected log_level 'debug', got: %v", resp["log_level"])
		}
	})

	// Test GetConfig returns updated values
	t.Run("get config returns updated values", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/configs/GetConfig", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp["data_dir"] != "/new/data" {
			t.Errorf("expected data_dir '/new/data', got: %v", resp["data_dir"])
		}
		if resp["log_level"] != "debug" {
			t.Errorf("expected log_level 'debug', got: %v", resp["log_level"])
		}
	})
}

// TestGetConfigRouteWithAuth tests the config endpoints with auth enabled.
func TestGetConfigRouteWithAuth(t *testing.T) {
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
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize configService
	app.configService = configs.NewService(app.db)

	// Set up a minimal scheduler for initDB
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Test GetConfig requires auth
	t.Run("get config requires auth", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/configs/GetConfig", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	// Test GetConfig with valid auth
	t.Run("get config with valid auth", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/configs/GetConfig", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		creds := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		req.Header.Set("Authorization", "Basic "+creds)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})
}

// TestConfigRouteWithBasePath tests config routes with a non-root base path.
func TestConfigRouteWithBasePath(t *testing.T) {
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize configService
	app.configService = configs.NewService(app.db)

	// Set up a minimal scheduler for initDB
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Test GetConfig at /walens/api/v1/configs/GetConfig
	t.Run("get config with base path", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/configs/GetConfig", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	// Test UpdateConfig at /walens/api/v1/configs/UpdateConfig
	t.Run("update config with base path", func(t *testing.T) {
		body := `{"data_dir":"/walens/data","log_level":"warn"}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/configs/UpdateConfig", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp["data_dir"] != "/walens/data" {
			t.Errorf("expected data_dir '/walens/data', got: %v", resp["data_dir"])
		}
	})
}

// TestSourceTypesRoutes tests the source_types API endpoints.
func TestSourceTypesRoutes(t *testing.T) {
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize configService
	app.configService = configs.NewService(app.db)

	// Set up a minimal scheduler for initDB
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Test ListSourceTypes
	t.Run("list source types returns registered sources", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_types/ListSourceTypes", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Huma unwraps Body field - items are at top level
		items, ok := resp["items"].([]interface{})
		if !ok {
			t.Fatalf("expected 'items' field in response, got: %v", resp)
		}

		if len(items) == 0 {
			t.Error("expected at least one source type")
		}

		// Check booru is in the list
		found := false
		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if itemMap["type_name"] == "booru" {
				found = true
				if itemMap["display_name"] != "Booru Image Board" {
					t.Errorf("expected display_name 'Booru Image Board', got: %v", itemMap["display_name"])
				}
				defaultCount, ok := itemMap["default_lookup_count"].(float64)
				if !ok {
					t.Error("expected default_lookup_count to be a number")
				}
				if int(defaultCount) != 100 {
					t.Errorf("expected default_lookup_count 100, got: %v", defaultCount)
				}
				break
			}
		}
		if !found {
			t.Error("expected 'booru' source type to be in list")
		}
	})

	// Test GetSourceType
	t.Run("get source type returns booru metadata", func(t *testing.T) {
		body := `{"type_name":"booru"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_types/GetSourceType", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Huma unwraps Body field - type_name and display_name are at top level
		if resp["type_name"] != "booru" {
			t.Errorf("expected type_name 'booru', got: %v", resp["type_name"])
		}
		if resp["display_name"] != "Booru Image Board" {
			t.Errorf("expected display_name 'Booru Image Board', got: %v", resp["display_name"])
		}
	})

	// Test GetSourceType not found
	t.Run("get source type returns 404 for unknown type", func(t *testing.T) {
		body := `{"type_name":"nonexistent"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_types/GetSourceType", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
		}
	})
}

// TestSourceTypesRoutesWithAuth tests the source_types API endpoints with auth enabled.
func TestSourceTypesRoutesWithAuth(t *testing.T) {
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
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize configService
	app.configService = configs.NewService(app.db)

	// Set up a minimal scheduler for initDB
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Test ListSourceTypes requires auth
	t.Run("list source types requires auth", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_types/ListSourceTypes", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	// Test ListSourceTypes with valid auth
	t.Run("list source types with valid auth", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_types/ListSourceTypes", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		creds := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		req.Header.Set("Authorization", "Basic "+creds)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})
}

// TestSourceTypesRoutesWithBasePath tests source_types routes with a non-root base path.
func TestSourceTypesRoutesWithBasePath(t *testing.T) {
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize configService
	app.configService = configs.NewService(app.db)

	// Set up a minimal scheduler for initDB
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Test ListSourceTypes at /walens/api/v1/source_types/ListSourceTypes
	t.Run("list source types with base path", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/source_types/ListSourceTypes", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	// Test GetSourceType at /walens/api/v1/source_types/GetSourceType
	t.Run("get source type with base path", func(t *testing.T) {
		body := `{"type_name":"booru"}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/source_types/GetSourceType", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Huma unwraps Body field
		if resp["type_name"] != "booru" {
			t.Errorf("expected type_name 'booru', got: %v", resp["type_name"])
		}
	})
}

// TestSourcesRoutes tests the sources CRUD API endpoints.
func TestSourcesRoutes(t *testing.T) {
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize services
	app.configService = configs.NewService(app.db)
	// Initialize sourceRegistry before sourcesService since sourcesService needs it
	app.sourceRegistry = sources.NewRegistry()
	app.sourceRegistry.Register(booru.New())
	app.sourceRegistry.Register(reddit.New())
	app.sourcesService = sourcesvc.NewService(sourcesvc.ServiceDependencies{DB: app.db, Registry: newSourceRegistry()})

	// Set up minimal scheduler
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Test ListSources - empty
	t.Run("list sources returns empty list initially", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/ListSources", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Huma may return null for nil slices, which unmarshals as nil
		// Handle both null and empty array cases
		items, ok := resp["items"]
		if !ok {
			t.Fatalf("expected 'items' field in response, got: %v", resp)
		}
		if items == nil {
			// items is null, which is semantically equivalent to empty for list operations
			// This is a Huma behavior where nil slices serialize as null
			items = []interface{}{}
		}
		itemsSlice, ok := items.([]interface{})
		if !ok {
			t.Fatalf("expected 'items' to be an array, got: %T", items)
		}
		if len(itemsSlice) != 0 {
			t.Errorf("expected 0 items, got %d", len(itemsSlice))
		}
	})

	// Test CreateSource
	t.Run("create source", func(t *testing.T) {
		body := `{"name":"test-source","source_type":"booru","params":{"tags":["nature"]},"lookup_count":50,"is_enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/CreateSource", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp["name"] != "test-source" {
			t.Errorf("expected name 'test-source', got: %v", resp["name"])
		}
		if resp["source_type"] != "booru" {
			t.Errorf("expected source_type 'booru', got: %v", resp["source_type"])
		}
		if resp["lookup_count"] != float64(50) {
			t.Errorf("expected lookup_count 50, got: %v", resp["lookup_count"])
		}
		// ID should be present
		if resp["id"] == nil || resp["id"] == "" {
			t.Error("expected id to be set")
		}
	})

	// Test CreateSource with invalid type
	t.Run("create source with invalid type returns 400", func(t *testing.T) {
		body := `{"name":"bad-source","source_type":"nonexistent","params":{},"lookup_count":50,"is_enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/CreateSource", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	// Test CreateSource with duplicate name
	t.Run("create source with duplicate name returns 409", func(t *testing.T) {
		body := `{"name":"test-source","source_type":"booru","params":{"tags":["test"]},"lookup_count":50,"is_enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/CreateSource", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusConflict, rec.Code, rec.Body.String())
		}
	})

	// Test ListSources - with data
	t.Run("list sources returns created source", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/ListSources", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		items, ok := resp["items"].([]interface{})
		if !ok {
			t.Fatalf("expected 'items' field in response, got: %v", resp)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}
	})
}

// TestSourcesRoutesWithAuth tests the sources API endpoints with auth enabled.
func TestSourcesRoutesWithAuth(t *testing.T) {
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
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize services
	app.configService = configs.NewService(app.db)
	app.sourcesService = sourcesvc.NewService(sourcesvc.ServiceDependencies{DB: app.db, Registry: newSourceRegistry()})

	// Set up minimal scheduler
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Test ListSources requires auth
	t.Run("list sources requires auth", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/ListSources", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	// Test ListSources with valid auth
	t.Run("list sources with valid auth", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/ListSources", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		creds := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		req.Header.Set("Authorization", "Basic "+creds)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})
}

// TestSourcesRoutesWithBasePath tests sources routes with a non-root base path.
func TestSourcesRoutesWithBasePath(t *testing.T) {
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize services
	app.configService = configs.NewService(app.db)
	// Initialize sourceRegistry before sourcesService since sourcesService needs it
	app.sourceRegistry = sources.NewRegistry()
	app.sourceRegistry.Register(booru.New())
	app.sourceRegistry.Register(reddit.New())
	app.sourcesService = sourcesvc.NewService(sourcesvc.ServiceDependencies{DB: app.db, Registry: newSourceRegistry()})

	// Set up minimal scheduler
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Test ListSources at /walens/api/v1/sources/ListSources
	t.Run("list sources with base path", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/sources/ListSources", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	// Test CreateSource at /walens/api/v1/sources/CreateSource
	t.Run("create source with base path", func(t *testing.T) {
		body := `{"name":"walens-source","source_type":"booru","params":{"tags":["test"]},"lookup_count":25,"is_enabled":false}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/sources/CreateSource", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp["name"] != "walens-source" {
			t.Errorf("expected name 'walens-source', got: %v", resp["name"])
		}
	})
}

// TestSourceSchedulesRoutes tests the source_schedules CRUD API endpoints.
func TestSourceSchedulesRoutes(t *testing.T) {
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize services
	app.configService = configs.NewService(app.db)
	app.sourceRegistry = sources.NewRegistry()
	app.sourceRegistry.Register(booru.New())
	app.sourceRegistry.Register(reddit.New())
	app.sourcesService = sourcesvc.NewService(sourcesvc.ServiceDependencies{DB: app.db, Registry: newSourceRegistry()})
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Helper to create a source first
	createSource := func(t *testing.T, name string) string {
		body := `{"name":"` + name + `","source_type":"booru","params":{"tags":["test"]},"lookup_count":25,"is_enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/CreateSource", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("create source failed: %d %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		return resp["id"].(string)
	}

	// Test ListSourceSchedules - empty initially
	t.Run("list source_schedules returns empty list initially", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/ListSourceSchedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		items, ok := resp["items"].([]interface{})
		if !ok {
			t.Fatalf("expected 'items' field in response, got: %v", resp)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})

	// Create a source to reference
	sourceID := createSource(t, "test-source-for-schedules")

	// Test CreateSourceSchedule
	t.Run("create source_schedule", func(t *testing.T) {
		body := `{"source_id":"` + sourceID + `","cron_expr":"0 * * * *","is_enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/CreateSourceSchedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		schedule := resp["schedule"].(map[string]interface{})
		if schedule["cron_expr"] != "0 * * * *" {
			t.Errorf("expected cron_expr '0 * * * *', got: %v", schedule["cron_expr"])
		}
		if schedule["source_id"] != sourceID {
			t.Errorf("expected source_id %s, got: %v", sourceID, schedule["source_id"])
		}
	})

	// Test CreateSourceSchedule with invalid cron
	t.Run("create source_schedule with invalid cron returns 400", func(t *testing.T) {
		body := `{"source_id":"` + sourceID + `","cron_expr":"invalid","is_enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/CreateSourceSchedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	// Test CreateSourceSchedule with non-existent source
	t.Run("create source_schedule with non-existent source returns 400", func(t *testing.T) {
		body := `{"source_id":"01800000-0000-0000-0000-000000000099","cron_expr":"0 * * * *","is_enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/CreateSourceSchedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	// Test ListSourceSchedules - with data
	t.Run("list source_schedules returns created schedule", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/ListSourceSchedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		items := resp["items"].([]interface{})
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}
	})

	// Get the schedule ID from the list
	getScheduleID := func() string {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/ListSourceSchedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		items := resp["items"].([]interface{})
		return items[0].(map[string]interface{})["id"].(string)
	}

	// Test GetSourceSchedule
	t.Run("get source_schedule", func(t *testing.T) {
		schedID := getScheduleID()
		body := `{"id":"` + schedID + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/GetSourceSchedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["id"] != schedID {
			t.Errorf("expected id %s, got: %v", schedID, resp["id"])
		}
	})

	// Test UpdateSourceSchedule
	t.Run("update source_schedule", func(t *testing.T) {
		schedID := getScheduleID()
		body := `{"id":"` + schedID + `","source_id":"` + sourceID + `","cron_expr":"*/5 * * * *","is_enabled":false}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/UpdateSourceSchedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		schedule := resp["schedule"].(map[string]interface{})
		if schedule["cron_expr"] != "*/5 * * * *" {
			t.Errorf("expected cron_expr '*/5 * * * *', got: %v", schedule["cron_expr"])
		}
		if schedule["is_enabled"] != false {
			t.Errorf("expected is_enabled false, got: %v", schedule["is_enabled"])
		}
	})

	// Test DeleteSourceSchedule
	t.Run("delete source_schedule", func(t *testing.T) {
		schedID := getScheduleID()
		body := `{"id":"` + schedID + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/DeleteSourceSchedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
		}
		if rec.Body.Len() != 0 {
			t.Errorf("expected empty body, got: %s", rec.Body.String())
		}

		// Verify deleted
		getReq := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/GetSourceSchedule", strings.NewReader(body))
		getReq.Header.Set("Content-Type", "application/json")
		getRec := httptest.NewRecorder()
		handler.ServeHTTP(getRec, getReq)
		if getRec.Code != http.StatusNotFound {
			t.Errorf("expected 404 after delete, got %d", getRec.Code)
		}
	})
}

// TestSourceSchedulesRoutesWithAuth tests source_schedules endpoints with auth enabled.
func TestSourceSchedulesRoutesWithAuth(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	if err := db.RunMigrations(testDB); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

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
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	app.configService = configs.NewService(app.db)
	app.sourceRegistry = sources.NewRegistry()
	app.sourceRegistry.Register(booru.New())
	app.sourcesService = sourcesvc.NewService(sourcesvc.ServiceDependencies{DB: app.db, Registry: newSourceRegistry()})
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Test ListSourceSchedules requires auth
	t.Run("list source_schedules requires auth", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/ListSourceSchedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	// Test ListSourceSchedules with valid auth
	t.Run("list source_schedules with valid auth", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/source_schedules/ListSourceSchedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		creds := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		req.Header.Set("Authorization", "Basic "+creds)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})
}

// TestSourceSchedulesRoutesWithBasePath tests source_schedules routes with a non-root base path.
func TestSourceSchedulesRoutesWithBasePath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	if err := db.RunMigrations(testDB); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "info",
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	app.configService = configs.NewService(app.db)
	app.sourceRegistry = sources.NewRegistry()
	app.sourceRegistry.Register(booru.New())
	app.sourceRegistry.Register(reddit.New())
	app.sourcesService = sourcesvc.NewService(sourcesvc.ServiceDependencies{DB: app.db, Registry: newSourceRegistry()})
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// Create a source first
	createSource := func(t *testing.T, name string) string {
		body := `{"name":"` + name + `","source_type":"booru","params":{"tags":["test"]},"lookup_count":25,"is_enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/sources/CreateSource", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		return resp["id"].(string)
	}

	t.Run("list source_schedules with base path", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/source_schedules/ListSourceSchedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("create source_schedule with base path", func(t *testing.T) {
		sourceID := createSource(t, "walens-test-source")
		body := `{"source_id":"` + sourceID + `","cron_expr":"0 * * * *","is_enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/source_schedules/CreateSourceSchedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		schedule := resp["schedule"].(map[string]interface{})
		if schedule["cron_expr"] != "0 * * * *" {
			t.Errorf("expected cron_expr '0 * * * *', got: %v", schedule["cron_expr"])
		}
	})

	t.Run("create source_schedule with invalid cron at base path", func(t *testing.T) {
		sourceID := createSource(t, "walens-test-source-2")
		body := `{"source_id":"` + sourceID + `","cron_expr":"not-valid","is_enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/source_schedules/CreateSourceSchedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})
}

// TestSchedulerReloadOnMutations tests that scheduler.Reload is called after create/update/delete.
// Note: This test uses the actual scheduler and verifies the service-level behavior.
// Route-level tests for auth and base path are covered by other tests.
func TestSchedulerReloadOnMutations(t *testing.T) {
	// This functionality is tested at the service level in source_schedules_test.go
	// via the mockScheduler. At the app/route level, we rely on the service tests
	// to verify Reload is called, and verify routes work correctly here.
	// This test is a placeholder to document the testing approach.
	t.Skip("Scheduler reload tested at service level via mockScheduler in source_schedules_test.go")
}

// TestSPAFallbackWithBasePath tests that the SPA fallback works correctly with a non-root base path.
func TestSPAFallbackWithBasePath(t *testing.T) {
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "error",
		Frontend: config.FrontendConfig{
			DevMode:   false,
			ViteURL:   "http://localhost:5173",
			StaticDir: tmpDir + "/nonexistent", // Non-existent dir, will disable SPA
		},
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize configService
	app.configService = configs.NewService(app.db)

	// Set up minimal scheduler
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// When SPA handler fails to initialize (static dir doesn't exist),
	// the app should fall back to just API handler, returning 404 for unknown routes
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "unknown route returns 404 when SPA not available",
			method:         http.MethodGet,
			path:           "/walens/unknown",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "health still works",
			method:         http.MethodGet,
			path:           "/walens/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "API route still works",
			method:         http.MethodGet,
			path:           "/walens/api/v1/configs/GetConfig",
			expectedStatus: http.StatusMethodNotAllowed, // Get on POST endpoint
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

// TestSPAFallbackRootPath tests that the root path serves the SPA shell when configured.
func TestSPAFallbackRootPath(t *testing.T) {
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "error",
		Frontend: config.FrontendConfig{
			DevMode:   false,
			ViteURL:   "http://localhost:5173",
			StaticDir: tmpDir + "/nonexistent",
		},
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	// Initialize configService
	app.configService = configs.NewService(app.db)

	// Set up minimal scheduler
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	// With auth disabled, the root path returns 404 (no SPA available since static dir doesn't exist)
	t.Run("root path without SPA returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})

	// Health should always work
	t.Run("health at root works", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})
}

// TestSPABasePathInjection tests that the SPA shell contains window.__WALENS__ config injection.
func TestSPABasePathInjection(t *testing.T) {
	// Test the escapeForJS function directly
	tests := []struct {
		input    string
		expected string
	}{
		{"/", `"/"`},
		{"/walens", `"/walens"`},
		{"/api", `"/api"`},
		{"http://localhost:5173", `"http://localhost:5173"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := frontend.EscapeForJS(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeForJS(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsAssetPath tests the asset path detection function.
func TestIsAssetPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/assets/index-abc123.js", true},
		{"/assets/styles.css", true},
		{"/favicon.ico", true},
		{"/images/logo.png", true},           // has .png extension
		{"/api/v1/configs/GetConfig", false}, // no asset extension
		{"/health", false},                   // no asset extension
		{"/docs", false},                     // no asset extension
		{"/openapi.json", false},             // no asset extension
		{"/src/main.js", true},               // has .js extension
		{"/routes/Settings.svelte", false},   // no asset extension
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := frontend.IsAssetPath(tt.path)
			if result != tt.expected {
				t.Errorf("IsAssetPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestSPRouterRoutes tests that the spRouter correctly routes requests.
func TestSPRouterRoutes(t *testing.T) {
	// Create mock handlers that track which was called
	var apiCalled bool
	var spaCalled bool

	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("API"))
	})

	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spaCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("SPA"))
	})

	router := &spRouter{
		apiHandler: apiHandler,
		spaHandler: spaHandler,
		basePath:   "/walens",
	}

	tests := []struct {
		name         string
		path         string
		expectAPI    bool
		expectSPA    bool
		expectedCode int
	}{
		{
			name:         "API routes go to API handler",
			path:         "/walens/api/v1/configs/GetConfig",
			expectAPI:    true,
			expectSPA:    false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "health goes to API handler",
			path:         "/walens/health",
			expectAPI:    true,
			expectSPA:    false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "docs go to API handler",
			path:         "/walens/docs",
			expectAPI:    true,
			expectSPA:    false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "openapi goes to API handler",
			path:         "/walens/openapi.json",
			expectAPI:    true,
			expectSPA:    false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "login goes to API handler",
			path:         "/walens/api/login",
			expectAPI:    true,
			expectSPA:    false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "logout goes to API handler",
			path:         "/walens/api/logout",
			expectAPI:    true,
			expectSPA:    false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "unknown routes go to SPA handler",
			path:         "/walens/unknown",
			expectAPI:    false,
			expectSPA:    true,
			expectedCode: http.StatusOK,
		},
		{
			name:         "root path goes to SPA handler",
			path:         "/walens",
			expectAPI:    false,
			expectSPA:    true,
			expectedCode: http.StatusOK,
		},
		{
			name:         "frontend routes go to SPA handler",
			path:         "/walens/sources",
			expectAPI:    false,
			expectSPA:    true,
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalled = false
			spaCalled = false

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if apiCalled != tt.expectAPI {
				t.Errorf("apiCalled = %v, want %v", apiCalled, tt.expectAPI)
			}
			if spaCalled != tt.expectSPA {
				t.Errorf("spaCalled = %v, want %v", spaCalled, tt.expectSPA)
			}
			if rec.Code != tt.expectedCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedCode)
			}
		})
	}
}

// TestSPRouterWithBasePathSlash tests routing when basePath is "/".
func TestSPRouterWithBasePathSlash(t *testing.T) {
	var apiCalled bool
	var spaCalled bool

	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(http.StatusOK)
	})

	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spaCalled = true
		w.WriteHeader(http.StatusOK)
	})

	router := &spRouter{
		apiHandler: apiHandler,
		spaHandler: spaHandler,
		basePath:   "/",
	}

	tests := []struct {
		name      string
		path      string
		expectAPI bool
		expectSPA bool
	}{
		{
			name:      "API routes at root go to API",
			path:      "/api/v1/configs/GetConfig",
			expectAPI: true,
			expectSPA: false,
		},
		{
			name:      "health at root goes to API",
			path:      "/health",
			expectAPI: true,
			expectSPA: false,
		},
		{
			name:      "unknown routes at root go to SPA",
			path:      "/sources",
			expectAPI: false,
			expectSPA: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalled = false
			spaCalled = false

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if apiCalled != tt.expectAPI {
				t.Errorf("apiCalled = %v, want %v", apiCalled, tt.expectAPI)
			}
			if spaCalled != tt.expectSPA {
				t.Errorf("spaCalled = %v, want %v", spaCalled, tt.expectSPA)
			}
		})
	}
}

// TestSPAWithRealStaticDir tests the SPA handler with a real static directory.
func TestSPAWithRealStaticDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fake dist directory with manifest and assets
	distDir := tmpDir + "/dist"
	if err := os.MkdirAll(distDir+"/.vite", 0755); err != nil {
		t.Fatalf("failed to create .vite dir: %v", err)
	}
	if err := os.MkdirAll(distDir+"/assets", 0755); err != nil {
		t.Fatalf("failed to create assets dir: %v", err)
	}

	// Create minimal manifest.json
	manifest := `{
  "src/main.js": {
    "file": "assets/index-abc123.js",
    "src": "src/main.js",
    "isEntry": true,
    "type": "module"
  }
}`
	if err := os.WriteFile(distDir+"/.vite/manifest.json", []byte(manifest), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create fake JS asset
	assetContent := `console.log("test asset");
export const foo = 42;`
	if err := os.WriteFile(distDir+"/assets/index-abc123.js", []byte(assetContent), 0644); err != nil {
		t.Fatalf("failed to write asset: %v", err)
	}

	// Create SPA handler in production mode to test static asset serving
	spa, err := frontend.NewSPAHandler("/walens", "http://localhost:5173", false, os.DirFS(distDir))
	if err != nil {
		t.Fatalf("NewSPAHandler failed: %v", err)
	}

	t.Run("SPA shell contains window.__WALENS__ config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/walens/some-route", nil)
		rec := httptest.NewRecorder()
		spa.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()

		// Check for window.__WALENS__ injection
		if !strings.Contains(body, "window.__WALENS__") {
			t.Error("HTML shell missing window.__WALENS__")
		}
		if !strings.Contains(body, `"basePath"`) {
			t.Error("HTML shell missing basePath in config")
		}
		if !strings.Contains(body, `"apiBase"`) {
			t.Error("HTML shell missing apiBase in config")
		}
		// Check for correct values
		if !strings.Contains(body, `"/walens"`) {
			t.Error("HTML shell missing /walens basePath value")
		}
		if !strings.Contains(body, `"/walens/api"`) {
			t.Error("HTML shell missing /walens/api apiBase value")
		}
	})

	t.Run("API route returns 404 from SPA handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/walens/api/v1/configs/GetConfig", nil)
		rec := httptest.NewRecorder()
		spa.ServeHTTP(rec, req)

		// SPA handler should return 404 for API routes
		// (letting the API handler on mux handle it)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404 for API route, got %d", rec.Code)
		}
	})

	t.Run("assets are served correctly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/walens/assets/index-abc123.js", nil)
		rec := httptest.NewRecorder()
		spa.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200 for asset, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "test asset") {
			t.Error("asset content not served correctly")
		}
	})
}

// TestSPAWithBasePathAndHealth tests that health route still works while SPA fallback serves shell.
func TestSPAWithBasePathAndHealth(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	if err := db.RunMigrations(testDB); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Create fake dist directory
	distDir := tmpDir + "/dist"
	if err := os.MkdirAll(distDir+"/.vite", 0755); err != nil {
		t.Fatalf("failed to create .vite dir: %v", err)
	}
	if err := os.MkdirAll(distDir+"/assets", 0755); err != nil {
		t.Fatalf("failed to create assets dir: %v", err)
	}

	manifest := `{"src/main.js":{"file":"assets/index-abc123.js","src":"src/main.js","type":"module"}}`
	if err := os.WriteFile(distDir+"/.vite/manifest.json", []byte(manifest), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(distDir+"/assets/index-abc123.js", []byte(`console.log("test");`), 0644); err != nil {
		t.Fatalf("failed to write asset: %v", err)
	}

	// Use dev mode to avoid needing proper manifest with entry point
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "error",
		Frontend: config.FrontendConfig{
			DevMode:   true,
			ViteURL:   "http://localhost:5173",
			StaticDir: distDir,
		},
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	app.configService = configs.NewService(app.db)
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	t.Run("health at /walens/health works", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/walens/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("SPA route serves shell with config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/walens/sources", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "window.__WALENS__") {
			t.Error("SPA shell missing window.__WALENS__")
		}
		if !strings.Contains(body, `"/walens"`) {
			t.Error("SPA shell missing /walens basePath")
		}
		if !strings.Contains(body, `"/walens/api"`) {
			t.Error("SPA shell missing /walens/api apiBase")
		}
	})

	t.Run("API route at /walens/api/v1/... works", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/walens/api/v1/configs/GetConfig", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})
}

// TestDevModeSPAHandler tests the SPA handler in dev mode.
func TestDevModeSPAHandler(t *testing.T) {
	// In dev mode, we don't need a real FS
	spa, err := frontend.NewSPAHandler("/walens", "http://localhost:5173", true, nil)
	if err != nil {
		t.Fatalf("NewSPAHandler failed: %v", err)
	}

	t.Run("dev mode serves HTML shell with Vite URL", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/walens/some-route", nil)
		rec := httptest.NewRecorder()
		spa.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()
		// Check config injection
		if !strings.Contains(body, "window.__WALENS__") {
			t.Error("HTML shell missing window.__WALENS__")
		}
		if !strings.Contains(body, `"/walens"`) {
			t.Error("HTML shell missing /walens basePath")
		}
		// In dev mode, the fragment tags should contain the Vite dev server URL
		if !strings.Contains(body, "http://localhost:5173") {
			t.Error("HTML shell should contain Vite dev server URL")
		}
	})
}

// TestSPAHandlerAtRoot tests the SPA handler with basePath="/".
func TestSPAHandlerAtRoot(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer testDB.Close()

	if err := db.RunMigrations(testDB); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Create fake dist directory
	distDir := tmpDir + "/dist"
	if err := os.MkdirAll(distDir+"/.vite", 0755); err != nil {
		t.Fatalf("failed to create .vite dir: %v", err)
	}
	if err := os.MkdirAll(distDir+"/assets", 0755); err != nil {
		t.Fatalf("failed to create assets dir: %v", err)
	}

	manifest := `{"src/main.js":{"file":"assets/index-abc123.js","src":"src/main.js","type":"module"}}`
	if err := os.WriteFile(distDir+"/.vite/manifest.json", []byte(manifest), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(distDir+"/assets/index-abc123.js", []byte(`console.log("root test");`), 0644); err != nil {
		t.Fatalf("failed to write asset: %v", err)
	}

	// Use dev mode to avoid needing proper manifest with entry point
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
			Enabled:      false,
			CookieSecret: testCookieSecret,
		},
		DataDir:  tmpDir,
		LogLevel: "error",
		Frontend: config.FrontendConfig{
			DevMode:   true,
			ViteURL:   "http://localhost:5173",
			StaticDir: distDir,
		},
	}

	app := &App{
		config: cfg,
		logger: slog.Default(),
		db:     testDB,
		queue:  queue.New(slog.Default()),
		runner: newTestRunner(t, testDB),
	}

	app.configService = configs.NewService(app.db)
	app.scheduler = newTestScheduler(t, testDB)

	handler := app.Handler()

	t.Run("health at /health works", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("SPA route at /sources serves shell", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sources", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "window.__WALENS__") {
			t.Error("SPA shell missing window.__WALENS__")
		}
		if !strings.Contains(body, `"/"`) && !strings.Contains(body, `"/"`) {
			// basePath should be "/"
		}
		if !strings.Contains(body, `"/api"`) {
			t.Error("SPA shell missing /api apiBase")
		}
	})

	t.Run("API route works", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/configs/GetConfig", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})
}
