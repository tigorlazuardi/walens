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

	"github.com/walens/walens/internal/config"
	"github.com/walens/walens/internal/db"
	"github.com/walens/walens/internal/logger"
	"github.com/walens/walens/internal/queue"
	"github.com/walens/walens/internal/runner"
	"github.com/walens/walens/internal/scheduler"
)

// App manages the lifecycle of all application components.
type App struct {
	config    *config.Config
	logger    *slog.Logger
	db        *sql.DB
	server    *http.Server
	scheduler *scheduler.Scheduler
	queue     *queue.Queue
	runner    *runner.Runner
}

// New creates a new application instance.
func New(cfg *config.Config) *App {
	log := logger.New(cfg.LogLevel)

	q := queue.New(log)
	sc := scheduler.New(log)
	ru := runner.New(log)
	ru.SetQueue(q)

	return &App{
		config:    cfg,
		logger:    log,
		scheduler: sc,
		queue:     q,
		runner:    ru,
	}
}

// Start initializes and starts all application components.
// Startup order: DB -> scheduler -> runner -> HTTP server.
func (a *App) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// 4. Start HTTP server (needs scheduler for health checks)
	if err := a.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	a.logger.Info("application started",
		"host", a.config.Server.Host,
		"port", a.config.Server.Port,
		"base_path", a.config.Server.BasePath,
	)

	return a.waitForShutdown()
}

// initDB opens the database connection and applies SQLite pragmas.
func (a *App) initDB() error {
	var err error
	a.db, err = db.Open(a.config.Database.Path)
	if err != nil {
		return fmt.Errorf("database open failed: %w", err)
	}
	a.logger.Info("database connected", "path", a.config.Database.Path)

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

// startHTTPServer configures and starts the HTTP server with health endpoint.
func (a *App) startHTTPServer() error {
	mux := http.NewServeMux()
	basePath := a.config.Server.BasePath

	// Register health endpoint under the configured base path.
	healthPath := joinPath(basePath, "health")
	mux.HandleFunc(healthPath, a.handleHealth)

	a.logger.Info("HTTP server configured", "base_path", basePath, "health_path", healthPath)

	a.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		a.logger.Info("HTTP server listening", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// handleHealth returns the health status of the application.
func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check DB connectivity
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	dbOK := true
	if a.db != nil {
		if err := db.Ping(ctx, a.db); err != nil {
			dbOK = false
			a.logger.Warn("health check: database ping failed", "error", err)
		}
	} else {
		dbOK = false
	}

	// Report status
	status := "ok"
	httpStatus := http.StatusOK
	if !dbOK {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	fmt.Fprintf(w, `{"status":"%s","queue_size":%d}`, status, a.queue.Size())
}

// waitForShutdown listens for shutdown signals and orchestrates graceful shutdown.
// Shutdown order (reverse of startup):
//   - HTTP server (stop accepting new connections)
//   - queue close (stop accepting new jobs, wake runner)
//   - runner stop (let runner drain current job then exit)
//   - scheduler stop
//   - DB close
func (a *App) waitForShutdown() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	a.logger.Info("shutdown signal received", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Shutdown HTTP server
	a.logger.Info("shutting down HTTP server")
	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("HTTP server shutdown error", "error", err)
	} else {
		a.logger.Info("HTTP server stopped")
	}

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
