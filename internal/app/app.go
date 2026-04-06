package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/walens/walens/internal/config"
	"github.com/walens/walens/internal/db"
	"github.com/walens/walens/internal/logger"
	"github.com/walens/walens/internal/queue"
	"github.com/walens/walens/internal/runner"
	"github.com/walens/walens/internal/scheduler"
)

type App struct {
	config    *config.Config
	logger    *slog.Logger
	db        *sql.DB
	server    *http.Server
	scheduler *scheduler.Scheduler
	queue     *queue.Queue
	runner    *runner.Runner
}

func New(cfg *config.Config) *App {
	log := logger.New(cfg.LogLevel)

	return &App{
		config:    cfg,
		logger:    log,
		scheduler: scheduler.New(log),
		queue:     queue.New(log),
		runner:    runner.New(log),
	}
}

func (a *App) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := a.initDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := a.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	if err := a.runner.Start(ctx); err != nil {
		return fmt.Errorf("failed to start runner: %w", err)
	}

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

func (a *App) initDB() error {
	var err error
	a.db, err = db.Open(a.config.Database.Path)
	if err != nil {
		return err
	}
	a.logger.Info("database connected", "path", a.config.Database.Path)
	return nil
}

func (a *App) startHTTPServer() error {
	mux := http.NewServeMux()

	basePath := a.config.Server.BasePath
	a.logger.Info("HTTP server configured", "base_path", basePath)

	a.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

func (a *App) waitForShutdown() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	a.logger.Info("shutdown signal received", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("HTTP server shutdown error", "error", err)
	}

	if err := a.runner.Stop(); err != nil {
		a.logger.Error("runner stop error", "error", err)
	}

	if err := a.scheduler.Stop(); err != nil {
		a.logger.Error("scheduler stop error", "error", err)
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error("database close error", "error", err)
		}
	}

	a.logger.Info("application shutdown complete")
	return nil
}
