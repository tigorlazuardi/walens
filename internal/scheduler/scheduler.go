package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/services/jobs"
)

// cronLogger wraps slog.Logger to implement cron.Logger interface
type cronLogger struct {
	logger *slog.Logger
}

func (c *cronLogger) Printf(format string, v ...interface{}) {
	c.logger.Info(fmt.Sprintf(format, v...))
}

// ScheduledJob represents a scheduled job entry with its metadata
type ScheduledJob struct {
	EntryID    cron.EntryID
	SourceID   dbtypes.UUID
	ScheduleID dbtypes.UUID
	SourceName string
	SourceType string
	CronExpr   string
}

// Scheduler manages cron-based job scheduling for source syncs.
type Scheduler struct {
	logger    *slog.Logger
	db        *sql.DB
	jobsSvc   *jobs.Service
	cron      *cron.Cron
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	ready     bool
	schedules map[string]ScheduledJob // key: "sourceID:scheduleID"
	enqueueFn func(jobID string)      // function to enqueue job to the queue
}

// New creates a new scheduler instance.
func New(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		logger:    logger,
		schedules: make(map[string]ScheduledJob),
	}
}

// SetDB sets the database handle for schedule queries.
func (s *Scheduler) SetDB(db *sql.DB) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db = db
}

// SetJobsService sets the jobs service for creating jobs.
func (s *Scheduler) SetJobsService(svc *jobs.Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobsSvc = svc
}

// SetEnqueueFunc sets the function used to enqueue jobs.
func (s *Scheduler) SetEnqueueFunc(fn func(jobID string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enqueueFn = fn
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

	// Create cron with seconds support and logging
	cronLog := &cronLogger{logger: s.logger}
	s.cron = cron.New(cron.WithSeconds(), cron.WithLogger(cron.VerbosePrintfLogger(cronLog)))

	s.wg.Add(1)
	go s.run()

	// Perform initial reload to load schedules from DB.
	if err := s.Reload(); err != nil {
		s.logger.Error("initial scheduler reload failed", "error", err)
	}

	s.logger.Info("scheduler started")
	return nil
}

// run is the main scheduler loop that keeps the cron running.
func (s *Scheduler) run() {
	defer s.wg.Done()

	s.logger.Info("scheduler loop started")

	// Start the cron scheduler
	s.cron.Start()

	<-s.ctx.Done()

	s.logger.Info("scheduler loop context cancelled, stopping cron")
	s.cron.Stop()
}

// Reload rebuilds the scheduler state from the database.
// This should be called whenever sources, source_schedules, or their
// enabled/disabled state changes.
func (s *Scheduler) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		s.logger.Warn("scheduler reload: database not available")
		return nil
	}

	s.logger.Info("scheduler reload: loading sources and schedules from database")

	// Clear existing schedules from cron
	for key, scheduled := range s.schedules {
		s.cron.Remove(scheduled.EntryID)
		delete(s.schedules, key)
		s.logger.Debug("scheduler reload: removed existing schedule", "key", key)
	}

	// Load enabled sources with their enabled schedules
	sourceSchedules, err := s.loadEnabledSourceSchedules()
	if err != nil {
		return fmt.Errorf("load enabled source schedules: %w", err)
	}

	// Add each schedule to cron
	for _, ss := range sourceSchedules {
		if err := s.addScheduleLocked(ss); err != nil {
			s.logger.Warn("scheduler reload: failed to add schedule",
				"source_id", ss.SourceID,
				"schedule_id", ss.ScheduleID,
				"cron_expr", ss.CronExpr,
				"error", err)
			continue
		}
	}

	s.ready = true
	s.logger.Info("scheduler reload complete", "schedules_loaded", len(s.schedules))
	return nil
}

// SourceSchedule represents a source with its schedule
type SourceSchedule struct {
	SourceID   dbtypes.UUID
	SourceName string
	SourceType string
	ScheduleID dbtypes.UUID
	CronExpr   string
}

// loadEnabledSourceSchedules loads all enabled sources with their enabled schedules.
func (s *Scheduler) loadEnabledSourceSchedules() ([]SourceSchedule, error) {
	query := `
		SELECT 
			s.id as source_id,
			s.name as source_name,
			s.source_type,
			ss.id as schedule_id,
			ss.cron_expr
		FROM sources s
		INNER JOIN source_schedules ss ON ss.source_id = s.id
		WHERE s.is_enabled = 1 AND ss.is_enabled = 1
		ORDER BY s.id, ss.id
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query enabled source schedules: %w", err)
	}
	defer rows.Close()

	var results []SourceSchedule
	for rows.Next() {
		var ss SourceSchedule
		var sourceIDStr, scheduleIDStr string
		if err := rows.Scan(&sourceIDStr, &ss.SourceName, &ss.SourceType, &scheduleIDStr, &ss.CronExpr); err != nil {
			s.logger.Warn("failed to scan source schedule", "error", err)
			continue
		}

		sourceID, err := dbtypes.NewUUIDFromString(sourceIDStr)
		if err != nil {
			s.logger.Warn("failed to parse source ID", "error", err, "value", sourceIDStr)
			continue
		}
		ss.SourceID = sourceID

		scheduleID, err := dbtypes.NewUUIDFromString(scheduleIDStr)
		if err != nil {
			s.logger.Warn("failed to parse schedule ID", "error", err, "value", scheduleIDStr)
			continue
		}
		ss.ScheduleID = scheduleID

		results = append(results, ss)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source schedules: %w", err)
	}

	return results, nil
}

// addScheduleLocked adds a single schedule to the cron (must be called with lock held).
func (s *Scheduler) addScheduleLocked(ss SourceSchedule) error {
	key := fmt.Sprintf("%s:%s", ss.SourceID.UUID.String(), ss.ScheduleID.UUID.String())

	// Check if already exists
	if _, exists := s.schedules[key]; exists {
		s.logger.Debug("schedule already exists, skipping", "key", key)
		return nil
	}

	// Create the job function
	jobFunc := func() {
		s.executeScheduledJob(ss)
	}

	// Parse and add to cron
	entryID, err := s.cron.AddFunc(ss.CronExpr, jobFunc)
	if err != nil {
		return fmt.Errorf("invalid cron expression '%s': %w", ss.CronExpr, err)
	}

	s.schedules[key] = ScheduledJob{
		EntryID:    entryID,
		SourceID:   ss.SourceID,
		ScheduleID: ss.ScheduleID,
		SourceName: ss.SourceName,
		SourceType: ss.SourceType,
		CronExpr:   ss.CronExpr,
	}

	nextRun := s.cron.Entry(entryID).Next
	s.logger.Info("schedule added",
		"source_id", ss.SourceID.UUID.String(),
		"source_name", ss.SourceName,
		"schedule_id", ss.ScheduleID.UUID.String(),
		"cron_expr", ss.CronExpr,
		"next_run", nextRun)

	return nil
}

// executeScheduledJob creates and enqueues a job for a scheduled source sync.
func (s *Scheduler) executeScheduledJob(ss SourceSchedule) {
	s.logger.Info("executing scheduled job",
		"source_id", ss.SourceID.UUID.String(),
		"source_name", ss.SourceName,
		"schedule_id", ss.ScheduleID.UUID.String())

	if s.jobsSvc == nil {
		s.logger.Error("cannot execute scheduled job: jobs service not set")
		return
	}

	// Create the job
	req := jobs.CreateJobRequest{
		JobType:             jobs.JobTypeSourceSync,
		SourceID:            &ss.SourceID,
		SourceName:          ss.SourceName,
		SourceType:          ss.SourceType,
		TriggerKind:         jobs.TriggerKindSchedule,
		RunAfter:            time.Now().UTC(),
		RequestedImageCount: 0, // Use source default
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	job, err := s.jobsSvc.CreateJob(ctx, req)
	if err != nil {
		s.logger.Error("failed to create scheduled job", "error", err)
		return
	}

	jobID := job.ID.UUID.String()
	s.logger.Info("scheduled job created", "job_id", jobID)

	// Enqueue to the in-memory queue
	if s.enqueueFn != nil {
		s.enqueueFn(jobID)
		s.logger.Info("scheduled job enqueued", "job_id", jobID)
	} else {
		s.logger.Warn("scheduled job not enqueued: enqueue function not set", "job_id", jobID)
	}
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
	if s.cron != nil {
		s.cron.Stop()
	}
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

// GetScheduleCount returns the number of active schedules.
func (s *Scheduler) GetScheduleCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.schedules)
}
