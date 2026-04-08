package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/walens/walens/internal/auth"
	"github.com/walens/walens/internal/config"
	"github.com/walens/walens/internal/db"
	"github.com/walens/walens/internal/logger"
	"github.com/walens/walens/internal/queue"
	"github.com/walens/walens/internal/routes"
	"github.com/walens/walens/internal/runner"
	"github.com/walens/walens/internal/scheduler"
	"github.com/walens/walens/internal/services/configs"
	sourcesvc "github.com/walens/walens/internal/services/sources"
	"github.com/walens/walens/internal/sources"
	"github.com/walens/walens/internal/sources/booru"
	"github.com/walens/walens/internal/sources/reddit"
)

type authCookieSecureContextKey struct{}

// App manages the lifecycle of all application components.
type App struct {
	config         *config.Config
	logger         *slog.Logger
	db             *sql.DB
	configService  *configs.Service
	sourcesService *sourcesvc.Service
	sourceRegistry *sources.Registry
	server         *http.Server
	scheduler      *scheduler.Scheduler
	queue          *queue.Queue
	runner         *runner.Runner
	handler        http.Handler
}

// New creates a new application instance.
func New(cfg *config.Config) *App {
	log := logger.New(cfg.LogLevel)

	q := queue.New(log)
	sc := scheduler.New(log)
	ru := runner.New(log)
	ru.SetQueue(q)
	registry := newSourceRegistry()

	return &App{
		config:         cfg,
		logger:         log,
		scheduler:      sc,
		queue:          q,
		runner:         ru,
		sourceRegistry: registry,
	}
}

func newSourceRegistry() *sources.Registry {
	registry := sources.NewRegistry()
	registry.Register(booru.New())
	registry.Register(reddit.New())
	return registry
}

// Handler returns the HTTP handler for the application, useful for testing.
func (a *App) Handler() http.Handler {
	if a.handler == nil {
		a.handler = a.buildHTTPHandler()
	}
	return a.handler
}

// Start initializes and starts all application components.
// Startup order: DB -> scheduler -> runner -> HTTP server.
func (a *App) Start() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-ctx.Done()
		cancel()
	}()

	// Validate auth config
	if err := a.config.Auth.Validate(); err != nil {
		return fmt.Errorf("auth configuration error: %w", err)
	}

	// 1. Initialize database
	if err := a.initDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// 2. Start scheduler (needs DB for reload)
	if err := a.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// 3. Start runner (consumes from queue)
	if err := a.runner.Start(ctx); err != nil {
		return fmt.Errorf("failed to start runner: %w", err)
	}

	a.logger.Info("application started",
		"host", a.config.Server.Host,
		"port", a.config.Server.Port,
		"base_path", a.config.Server.BasePath,
	)

	// 4. Start HTTP server (needs scheduler for health checks)
	if err := a.startHTTPServer(ctx); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return a.waitForShutdown(ctx)
}

// initDB opens the database connection and applies SQLite pragmas.
func (a *App) initDB() error {
	var err error
	a.db, err = db.Open(a.config.Database.Path)
	if err != nil {
		return fmt.Errorf("database open failed: %w", err)
	}
	a.logger.Info("database connected", "path", a.config.Database.Path)

	// Run migrations after opening DB and before scheduler starts
	if err := db.RunMigrations(a.db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	a.logger.Info("database migrations applied")

	// Load persisted config after migrations. If absent or empty, inject defaults.
	a.configService = configs.NewService(a.db)

	// Initialize sources service with db and registry
	a.sourcesService = sourcesvc.NewService(a.db, a.sourceRegistry)

	defaultPersistedCfg := configs.DefaultPersistedConfig()
	defaultPersistedCfg.ApplyBootstrapConfig(a.config)
	persistedCfg, err := a.configService.BootstrapDefault(context.Background(), defaultPersistedCfg)
	if err != nil {
		return fmt.Errorf("failed to bootstrap persisted config: %w", err)
	}
	a.logger.Info("persisted config loaded",
		"data_dir", persistedCfg.DataDir,
		"log_level", persistedCfg.LogLevel,
	)

	// Apply persisted config back to active config so runtime uses DB values.
	// Note: BasePath is NOT applied from persisted config - it is bootstrap-only.
	a.config.ApplyPersistedConfig(persistedCfg.DataDir, persistedCfg.LogLevel)

	// Rebuild logger with persisted log level, then rebuild dependent components.
	a.logger = logger.New(persistedCfg.LogLevel)
	a.queue = queue.New(a.logger)
	a.runner = runner.New(a.logger)
	a.runner.SetQueue(a.queue)
	a.scheduler = scheduler.New(a.logger)

	// Give scheduler access to DB for reload queries
	a.scheduler.SetDB(a.db)

	return nil
}

// joinPath safely joins a base path with a suffix, ensuring correct path construction.
// Examples:
//   - joinPath("/", "health") => "/health"
//   - joinPath("/walens", "health") => "/walens/health"
//   - joinPath("/walens/", "health") => "/walens/health"
func joinPath(base, suffix string) string {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = "/"
	}
	return path.Join(base, suffix)
}

func (a *App) buildHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	basePath := a.config.Server.BasePath
	if a.sourceRegistry == nil {
		a.sourceRegistry = newSourceRegistry()
	}
	a.sourcesService = sourcesvc.NewService(a.db, a.sourceRegistry)
	humaConfig := huma.DefaultConfig("Walens API", "0.0.1")
	humaConfig.OpenAPIPath = joinPath(basePath, "/openapi")
	humaConfig.DocsPath = joinPath(basePath, "/docs")
	humaConfig.SchemasPath = joinPath(basePath, "/schemas")
	api := humago.New(mux, humaConfig)

	// Build auth config for middleware
	authConfig := auth.Config{
		Enabled:      a.config.Auth.Enabled,
		Username:     a.config.Auth.Username,
		Password:     a.config.Auth.Password,
		BasePath:     a.config.Server.BasePath,
		CookieSecure: a.config.Auth.CookieSecure,
		CookieSecret: a.config.Auth.CookieSecret,
	}

	a.registerHumaRoutes(api, basePath, authConfig)

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), authCookieSecureContextKey{}, authConfig.CookieSecure)
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
	if authConfig.Enabled {
		handler = authConfig.Middleware()(handler)
	}

	return handler
}

type healthOutput struct {
	Body struct {
		Status    string `json:"status"`
		QueueSize int    `json:"queue_size"`
	}
}

type loginInput struct {
	Body struct {
		Username string `json:"username" doc:"Bootstrap auth username."`
		Password string `json:"password" doc:"Bootstrap auth password."`
	}
}

type loginOutput struct {
	SetCookie string `header:"Set-Cookie"`
}

type logoutInput struct {
	Secure   bool   `json:"-"`
	BasePath string `json:"-"`
}

type logoutOutput struct{}

var _ huma.Resolver = (*logoutInput)(nil)

func (i *logoutInput) Resolve(ctx huma.Context) []error {
	if secure, ok := ctx.Context().Value(authCookieSecureContextKey{}).(bool); ok {
		i.Secure = secure
	}
	i.BasePath = strings.TrimSuffix(ctx.Operation().Path, "/api/logout")
	if i.BasePath == "" {
		i.BasePath = "/"
	}
	cookie := auth.ClearAuthCookie(i.BasePath)
	cookie.Secure = i.Secure
	ctx.AppendHeader("Set-Cookie", cookie.String())
	return nil
}

func (a *App) registerHumaRoutes(api huma.API, basePath string, authConfig auth.Config) {
	huma.Register(api, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        joinPath(basePath, "health"),
		Summary:     "Get application health",
		Description: "Returns Walens process health, including database availability and current in-memory queue size for infrastructure monitoring.",
		Tags:        []string{"infra"},
	}, func(ctx context.Context, input *struct{}) (*healthOutput, error) {
		output := &healthOutput{}
		output.Body.QueueSize = a.queue.Size()
		output.Body.Status = "ok"

		if a.db == nil {
			output.Body.Status = "degraded"
			return output, huma.Error503ServiceUnavailable("database unavailable")
		}

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := db.Ping(pingCtx, a.db); err != nil {
			a.logger.Warn("health check: database ping failed", "error", err)
			output.Body.Status = "degraded"
			return output, huma.Error503ServiceUnavailable("database unavailable")
		}

		return output, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "post-login",
		Method:      http.MethodPost,
		Path:        joinPath(basePath, "/api/login"),
		Summary:     "Bootstrap browser auth cookie",
		Description: "Validates bootstrap Basic Auth credentials and sets an HTTP-only auth cookie for browser clients. Native or external clients should use the Authorization header directly instead of this endpoint.",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, input *loginInput) (*loginOutput, error) {
		if err := authConfig.ValidateCredentials(input.Body.Username, input.Body.Password); err != nil {
			return nil, huma.Error401Unauthorized("invalid credentials")
		}

		cookieValue, err := auth.BuildCookieValue(authConfig.CookieSecret, input.Body.Username, input.Body.Password)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to build auth cookie", err)
		}
		cookie := auth.NewAuthCookie(cookieValue, auth.CookieOptions{
			Secure:   authConfig.CookieSecure,
			SameSite: http.SameSiteLaxMode,
			Path:     basePath,
		})
		return &loginOutput{SetCookie: cookie.String()}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "post-logout",
		Method:      http.MethodPost,
		Path:        joinPath(basePath, "/api/logout"),
		Summary:     "Clear browser auth cookie",
		Description: "Clears the HTTP-only browser auth cookie. Header-based clients should simply stop sending Authorization credentials.",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, input *logoutInput) (*logoutOutput, error) {
		return &logoutOutput{}, nil
	})

	// Register configs RPC routes
	routes.RegisterConfigsRoutes(api, basePath, a.configService)

	// Register source types RPC routes
	routes.RegisterSourceTypesRoutes(api, basePath, a.sourceRegistry)

	// Register sources RPC routes
	routes.RegisterSourcesRoutes(api, basePath, a.sourcesService)

	// Register source_schedules RPC routes
	routes.RegisterSourceSchedulesRoutes(api, basePath, a.db, a.scheduler)

	// Register devices RPC routes
	routes.RegisterDevicesRoutes(api, basePath, a.db)
}

// startHTTPServer configures and starts the HTTP server with health endpoint.
func (a *App) startHTTPServer(ctx context.Context) error {
	basePath := a.config.Server.BasePath
	a.handler = a.buildHTTPHandler()
	a.logger.Info("HTTP server configured", "base_path", basePath)
	a.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port),
		Handler:           a.handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	err := make(chan error, 1)
	go func() {
		a.logger.Info("HTTP server listening", "addr", a.server.Addr)
		err <- a.server.ListenAndServe()
	}()
	select {
	case err := <-err:
		return fmt.Errorf("failed to start server: %w", err)
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second*5)
		defer cancel()
		if err := a.server.Shutdown(ctx); err != nil {
			a.logger.ErrorContext(ctx, "failed to shutdown server", "error", err)
		}
		return nil
	}
}

// waitForShutdown listens for shutdown signals and orchestrates graceful shutdown.
// Shutdown order (reverse of startup):
//   - queue close (stop accepting new jobs, wake runner)
//   - runner stop (let runner drain current job then exit)
//   - scheduler stop
//   - DB close
func (a *App) waitForShutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	// 2. Close queue (stop accepting new jobs, wake runner's dequeue)
	a.logger.Info("closing queue")
	a.queue.Close()
	// At this point runner will see closed queue and exit its loop.
	a.logger.Info("queue closed")

	// 3. Stop runner (wait for worker to finish current job and exit)
	a.logger.Info("stopping runner")
	if err := a.runner.Stop(); err != nil {
		a.logger.Error("runner stop error", "error", err)
	} else {
		a.logger.Info("runner stopped")
	}

	// 4. Stop scheduler
	a.logger.Info("stopping scheduler")
	if err := a.scheduler.Stop(); err != nil {
		a.logger.Error("scheduler stop error", "error", err)
	} else {
		a.logger.Info("scheduler stopped")
	}

	// 5. Close database
	if a.db != nil {
		a.logger.Info("closing database")
		if err := a.db.Close(); err != nil {
			a.logger.Error("database close error", "error", err)
		} else {
			a.logger.Info("database closed")
		}
	}

	a.logger.Info("application shutdown complete")
	return nil
}
