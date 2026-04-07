package scheduler

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"
)

// Scheduler manages cron-based job scheduling for source syncs.
type Scheduler struct {
	logger *slog.Logger
	db     *sql.DB
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	ready  bool // true after first successful reload
}

// New creates a new scheduler instance.
func New(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		logger: logger,
	}
}

// Start begins the scheduler background goroutine.
// It performs an initial reload to load schedules from the database.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.ctx != nil {
		s.mu.Unlock()
		s.logger.Warn("scheduler already started")
		return nil
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run()

	// Perform initial reload to load schedules from DB.
	if err := s.Reload(); err != nil {
		s.logger.Error("initial scheduler reload failed", "error", err)
	}

	s.logger.Info("scheduler started")
	return nil
}

// run is the main scheduler loop. Currently a placeholder that keeps
// the scheduler running and periodically logs its status.
func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	s.logger.Info("scheduler loop started")

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("scheduler loop context cancelled")
			return
		case <-ticker.C:
			s.mu.RLock()
			ready := s.ready
			s.mu.RUnlock()
			if ready {
				s.logger.Debug("scheduler tick", "status", "running")
			}
		}
	}
}

// Reload rebuilds the scheduler state from the database.
// This should be called whenever sources, source_schedules, or their
// enabled/disabled state changes.
func (s *Scheduler) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// In a real implementation, this would:
	// 1. Load all enabled sources from DB
	// 2. Load all enabled schedules for those sources
	// 3. Rebuild cron entries in memory
	// 4. Validate schedule proximity and emit warnings

	// Placeholder: just log the reload.
	s.logger.Info("scheduler reload: loading sources and schedules from database")

	if s.db != nil {
		// Example query that would be used in real implementation:
		// SELECT s.id, s.name, ss.id, ss.cron_expr
		// FROM sources s
		// JOIN source_schedules ss ON ss.source_id = s.id
		// WHERE s.is_enabled = 1 AND ss.is_enabled = 1
		s.logger.Debug("scheduler reload: db available for queries")
	}

	s.ready = true
	s.logger.Info("scheduler reload complete")
	return nil
}

// SetDB sets the database handle for schedule queries.
func (s *Scheduler) SetDB(db *sql.DB) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db = db
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if s.cancel == nil {
		s.mu.Unlock()
		s.logger.Warn("scheduler not started")
		return nil
	}
	s.cancel()
	s.mu.Unlock()

	s.logger.Info("scheduler stopping, waiting for loop to exit")
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("scheduler stopped gracefully")
	case <-time.After(30 * time.Second):
		s.logger.Warn("scheduler stop timed out")
	}

	s.mu.Lock()
	s.ctx = nil
	s.cancel = nil
	s.ready = false
	s.mu.Unlock()

	return nil
}

// IsReady returns true if the scheduler has completed at least one successful reload.
func (s *Scheduler) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}
