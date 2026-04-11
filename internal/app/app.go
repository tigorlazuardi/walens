package app

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
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
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/logger"
	"github.com/walens/walens/internal/queue"
	"github.com/walens/walens/internal/routes"
	imagesroutes "github.com/walens/walens/internal/routes/images"
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
)

type authCookieSecureContextKey struct{}

// App manages the lifecycle of all application components.
type App struct {
	config         *config.Config
	logger         *slog.Logger
	db             *sql.DB
	configService  *configs.Service
	jobsService    *jobs.Service
	sourcesService *sourcesvc.Service
	sourceRegistry *sources.Registry
	storageSvc     *storage.Service
	imageSvc       *images.Service
	tagsService    *tags.Service
	server         *http.Server
	scheduler      *scheduler.Scheduler
	queue          *queue.Queue
	runner         *runner.Runner
	handler        http.Handler
	api            huma.API
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

	// Initialize jobs service
	a.jobsService = jobs.NewService(a.db)

	// Boot recovery: requeue unfinished jobs from persisted state
	if err := a.recoverJobs(context.Background()); err != nil {
		a.logger.Warn("boot recovery: failed to recover jobs", "error", err)
	}

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

	// Initialize storage and image services with the persisted data directory.
	a.storageSvc = storage.NewService(storage.Config{BaseDir: persistedCfg.DataDir})
	a.imageSvc = images.NewService(a.db)
	a.tagsService = tags.NewService(a.db)

	// Rebuild logger with persisted log level, then rebuild dependent components.
	a.logger = logger.New(persistedCfg.LogLevel)
	a.queue = queue.New(a.logger)
	a.runner = runner.New(a.logger)
	a.runner.SetQueue(a.queue)
	a.runner.SetJobsService(a.jobsService)
	a.runner.SetStorageService(a.storageSvc)
	a.runner.SetImageService(a.imageSvc)
	a.runner.SetTagsService(a.tagsService)
	a.runner.SetSourceRegistry(a.sourceRegistry)
	a.scheduler = scheduler.New(a.logger)

	// Give scheduler access to DB and jobs service for reload queries and job creation
	a.scheduler.SetDB(a.db)
	a.scheduler.SetJobsService(a.jobsService)
	a.scheduler.SetEnqueueFunc(a.queue.Enqueue)

	// Wire up scheduler reload to services that need it
	a.sourcesService.SetScheduler(a.scheduler)

	return nil
}

// recoverJobs performs boot recovery by requeuing unfinished jobs from the database.
// It retrieves queued and running jobs, resets running jobs to queued state,
// marks them for recovery, and enqueues them in the in-memory queue.
func (a *App) recoverJobs(ctx context.Context) error {
	if a.jobsService == nil || a.db == nil {
		return nil
	}

	// Get all jobs that need recovery (queued and running)
	jobsToRecover, err := a.jobsService.GetJobsForRecovery(ctx)
	if err != nil {
		return fmt.Errorf("get jobs for recovery: %w", err)
	}

	if len(jobsToRecover) == 0 {
		a.logger.Info("boot recovery: no unfinished jobs to recover")
		return nil
	}

	a.logger.Info("boot recovery: found unfinished jobs", "count", len(jobsToRecover))

	// Reset running jobs back to queued state
	recoveredCount, err := a.jobsService.RecoverRunningJobs(ctx)
	if err != nil {
		a.logger.Warn("boot recovery: failed to reset running jobs", "error", err)
	} else if recoveredCount > 0 {
		a.logger.Info("boot recovery: reset running jobs to queued", "count", recoveredCount)
	}

	// Collect job IDs for marking as recovery
	jobIDs := make([]dbtypes.UUID, 0, len(jobsToRecover))
	for _, job := range jobsToRecover {
		jobIDs = append(jobIDs, job.ID)
	}

	// Mark recovered jobs with trigger_kind=recovery
	if len(jobIDs) > 0 {
		markedCount, err := a.jobsService.MarkJobsForRecovery(ctx, jobIDs)
		if err != nil {
			a.logger.Warn("boot recovery: failed to mark jobs for recovery", "error", err)
		} else {
			a.logger.Info("boot recovery: marked jobs for recovery", "count", markedCount)
		}
	}

	// Enqueue all recovered jobs in the in-memory queue
	enqueuedCount := 0
	for _, job := range jobsToRecover {
		a.queue.Enqueue(job.ID.UUID.String())
		enqueuedCount++
	}

	a.logger.Info("boot recovery: enqueued jobs to in-memory queue", "count", enqueuedCount)
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

// spRouter routes requests to either the API handler or the SPA handler.
// Deprecated: kept for test compatibility only.
type spRouter struct {
	apiHandler http.Handler
	spaHandler http.Handler
	basePath   string
}

// ServeHTTP routes requests to the appropriate handler.
// Deprecated: kept for test compatibility only.
func (s *spRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Path

	// API routes start with basePath + /api
	apiPath := path.Join(s.basePath, "api")
	if strings.HasPrefix(reqPath, apiPath) {
		s.apiHandler.ServeHTTP(w, r)
		return
	}

	// Docs/OpenAPI routes
	docsPath := path.Join(s.basePath, "docs")
	openapiPath := path.Join(s.basePath, "openapi")
	schemasPath := path.Join(s.basePath, "schemas")
	if strings.HasPrefix(reqPath, docsPath) ||
		strings.HasPrefix(reqPath, openapiPath) ||
		strings.HasPrefix(reqPath, schemasPath) {
		s.apiHandler.ServeHTTP(w, r)
		return
	}

	// Health route
	healthPath := path.Join(s.basePath, "health")
	if reqPath == healthPath {
		s.apiHandler.ServeHTTP(w, r)
		return
	}

	// Login/logout routes under basePath/api
	loginPath := path.Join(s.basePath, "api", "login")
	logoutPath := path.Join(s.basePath, "api", "logout")
	if reqPath == loginPath || reqPath == logoutPath {
		s.apiHandler.ServeHTTP(w, r)
		return
	}

	// For all other routes, serve the SPA
	s.spaHandler.ServeHTTP(w, r)
}

func (a *App) buildHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	basePath := a.config.Server.BasePath
	if a.sourceRegistry == nil {
		a.sourceRegistry = newSourceRegistry()
	}
	a.sourcesService = sourcesvc.NewService(a.db, a.sourceRegistry)
	humaConfig := huma.DefaultConfig("Walens API", "0.0.1")
	humaConfig.FieldsOptionalByDefault = true
	humaConfig.OpenAPIPath = joinPath(basePath, "/openapi")
	humaConfig.DocsPath = joinPath(basePath, "/docs")
	humaConfig.SchemasPath = joinPath(basePath, "/schemas")
	api := humago.New(mux, humaConfig)
	a.api = api

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

	// Enhance OpenAPI spec for codegen friendliness: title case tags, stable operation IDs, README overview
	a.enhanceOpenAPISpec()

	// Mount direct image/thumbnail HTTP GET handlers on mux (before API handler wrapping)
	// These serve actual image files directly without going through Huma RPC
	// Use the same db that was passed to Huma routes for consistency
	imageSvcForHandlers := images.NewService(a.db)
	thumbHandler := &imagesroutes.ServeThumbnailHandler{BasePath: basePath, ImageSvc: imageSvcForHandlers}
	imageHandler := &imagesroutes.ServeImageHandler{BasePath: basePath, ImageSvc: imageSvcForHandlers}
	mux.Handle(thumbHandler.Pattern(), thumbHandler)
	mux.Handle(imageHandler.Pattern(), imageHandler)

	// Create the API handler with auth context
	var apiHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), authCookieSecureContextKey{}, authConfig.CookieSecure)
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
	if authConfig.Enabled {
		apiHandler = authConfig.Middleware()(apiHandler)
	}

	// Create SPA handler for frontend fallback
	var staticFS fs.FS
	if a.config.Frontend.DevMode {
		// In dev mode, use the frontend directory
		staticFS = os.DirFS(a.config.Frontend.StaticDir)
	} else {
		// In production, try to use the configured static dir
		if _, err := os.Stat(a.config.Frontend.StaticDir); err == nil {
			staticFS = os.DirFS(a.config.Frontend.StaticDir)
		}
	}

	spa, err := NewSPAHandler(
		basePath,
		a.config.Frontend.ViteURL,
		a.config.Frontend.DevMode,
		staticFS,
	)
	if err != nil {
		// Log warning but continue - SPA won't work but API will
		a.logger.Warn("failed to initialize SPA handler", "error", err)
	}

	// If SPA handler exists, mount it on mux as fallback
	// Huma routes are already registered, so they take precedence
	// over the SPA fallback due to ServeMux specificity rules
	if spa != nil {
		if basePath == "/" {
			// For root base path, mount SPA at "/" which is the true catch-all pattern
			// More specific patterns (like /health, /api/...) registered by Huma
			// take precedence over "/" due to ServeMux longest-match-wins semantics
			mux.Handle("/", spa)
		} else {
			// For non-root base path, mount at basePath + "/..." as catch-all
			// Also register exact basePath and basePath+"/" for SPA root routes
			mux.Handle(basePath, spa)
			mux.Handle(basePath+"/", spa)
			mux.Handle(basePath+"/*", spa)
		}
	}

	return apiHandler
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
		OperationID: "GetHealth",
		Method:      http.MethodGet,
		Path:        joinPath(basePath, "health"),
		Summary:     "Get application health",
		Description: "Returns Walens process health, including database availability and current in-memory queue size for infrastructure monitoring.",
		Tags:        []string{"Infra"},
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
		OperationID: "Login",
		Method:      http.MethodPost,
		Path:        joinPath(basePath, "/api/login"),
		Summary:     "Bootstrap browser auth cookie",
		Description: "Validates bootstrap Basic Auth credentials and sets an HTTP-only auth cookie for browser clients. Native or external clients should use the Authorization header directly instead of this endpoint.",
		Tags:        []string{"Auth"},
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
		OperationID: "Logout",
		Method:      http.MethodPost,
		Path:        joinPath(basePath, "/api/logout"),
		Summary:     "Clear browser auth cookie",
		Description: "Clears the HTTP-only browser auth cookie. Header-based clients should simply stop sending Authorization credentials.",
		Tags:        []string{"Auth"},
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

	// Register device_subscriptions RPC routes
	routes.RegisterDeviceSubscriptionsRoutes(api, basePath, a.db)

	// Register images RPC routes
	routes.RegisterImagesRoutes(api, basePath, a.db)

	// Register jobs RPC routes
	routes.RegisterJobsRoutes(api, basePath, a.db)

	// Register runtime status routes
	if a.scheduler != nil {
		statusDeps := &appRuntimeStatusDeps{
			app: a,
		}
		routes.RegisterRuntimeStatusRoutes(api, basePath, statusDeps)
	}
}

// appRuntimeStatusDeps implements runtime_status.RuntimeStatusDeps using the App struct.
type appRuntimeStatusDeps struct {
	app *App
}

func (d *appRuntimeStatusDeps) QueueSize() int {
	if d.app.queue == nil {
		return 0
	}
	return d.app.queue.Size()
}

func (d *appRuntimeStatusDeps) SchedulerReady() bool {
	if d.app.scheduler == nil {
		return false
	}
	return d.app.scheduler.IsReady()
}

func (d *appRuntimeStatusDeps) GetScheduleCount() int {
	if d.app.scheduler == nil {
		return 0
	}
	return d.app.scheduler.GetScheduleCount()
}

func (d *appRuntimeStatusDeps) IsRunnerActive() bool {
	// Runner is considered active if context is not nil (has been started and not stopped)
	return d.app.runner != nil
}

func (d *appRuntimeStatusDeps) IsAuthEnabled() bool {
	return d.app.config.Auth.Enabled
}

// enhanceOpenAPISpec adds tag descriptions and README overview to the OpenAPI spec.
// Tags and OperationIDs are set directly in route files.
func (a *App) enhanceOpenAPISpec() {
	if a.api == nil {
		return
	}

	// Read README.md for OpenAPI description/overview
	readmeContent := ""
	if data, err := os.ReadFile("README.md"); err == nil {
		readmeContent = strings.TrimSpace(string(data))
	}

	openapi := a.api.OpenAPI()
	if openapi == nil {
		return
	}

	// Set README as OpenAPI description if available
	if readmeContent != "" {
		openapi.Info.Description = readmeContent
	}

	// Tag descriptions for API docs
	tagDescriptions := map[string]string{
		"Auth":                 "Authentication and session management",
		"Configs":              "Application configuration persistence",
		"Device Subscriptions": "Device-to-source subscription management",
		"Devices":              "Device registration and management",
		"Images":               "Image metadata, thumbnails, and serving",
		"Infra":                "Health checks and infrastructure monitoring",
		"Jobs":                 "Background job status and history",
		"Runtime Status":       "Runtime state of scheduler, queue, and runner",
		"Source Schedules":     "Cron-based source fetch scheduling",
		"Source Types":         "Registered source implementation types",
		"Sources":              "Configured wallpaper source instances",
	}

	// Add tag descriptions to OpenAPI spec
	for _, pathItem := range openapi.Paths {
		a.addTagDescription(pathItem.Get, tagDescriptions)
		a.addTagDescription(pathItem.Post, tagDescriptions)
		a.addTagDescription(pathItem.Put, tagDescriptions)
		a.addTagDescription(pathItem.Delete, tagDescriptions)
		a.addTagDescription(pathItem.Patch, tagDescriptions)
	}
}

// addTagDescription adds description to the first tag found in an operation.
func (a *App) addTagDescription(op *huma.Operation, tagDescriptions map[string]string) {
	if op == nil || len(op.Tags) == 0 {
		return
	}
	tag := op.Tags[0]
	if desc, ok := tagDescriptions[tag]; ok {
		// Check if tag already exists in OpenAPI tags
		found := false
		for i, t := range a.api.OpenAPI().Tags {
			if t.Name == tag {
				a.api.OpenAPI().Tags[i].Description = desc
				found = true
				break
			}
		}
		if !found {
			a.api.OpenAPI().Tags = append(a.api.OpenAPI().Tags, &huma.Tag{Name: tag, Description: desc})
		}
	}
}

// OpenAPIYAML builds the OpenAPI specification and returns it as YAML bytes.
// This does not start the HTTP server, scheduler, or any background workers.
func (a *App) OpenAPIYAML() ([]byte, error) {
	mux := http.NewServeMux()
	basePath := a.config.Server.BasePath

	if a.sourceRegistry == nil {
		a.sourceRegistry = newSourceRegistry()
	}

	humaConfig := huma.DefaultConfig("Walens API", "0.0.1")
	humaConfig.FieldsOptionalByDefault = true
	humaConfig.OpenAPIPath = joinPath(basePath, "/openapi")
	humaConfig.DocsPath = joinPath(basePath, "/docs")
	humaConfig.SchemasPath = joinPath(basePath, "/schemas")
	api := humago.New(mux, humaConfig)

	authConfig := auth.Config{
		Enabled:      a.config.Auth.Enabled,
		Username:     a.config.Auth.Username,
		Password:     a.config.Auth.Password,
		BasePath:     a.config.Server.BasePath,
		CookieSecure: a.config.Auth.CookieSecure,
		CookieSecret: a.config.Auth.CookieSecret,
	}

	a.api = api
	a.registerHumaRoutes(api, basePath, authConfig)
	a.enhanceOpenAPISpec()

	return api.OpenAPI().YAML()
}

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
