package runner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/queue"
	"github.com/walens/walens/internal/services/jobs"
)

// Runner consumes jobs from the queue and processes them.
type Runner struct {
	logger  *slog.Logger
	queue   *queue.Queue
	jobsSvc *jobs.Service
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New creates a new job runner. Queue must be set before Start.
func New(logger *slog.Logger) *Runner {
	return &Runner{
		logger: logger,
	}
}

// SetQueue sets the queue for the runner to consume from.
func (r *Runner) SetQueue(q *queue.Queue) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queue = q
}

// SetJobsService sets the jobs service for job state management.
func (r *Runner) SetJobsService(svc *jobs.Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobsSvc = svc
}

// Start begins the runner worker goroutine that consumes jobs from the queue.
// The provided context should be the application context; runner will run until
// the context is cancelled or Stop is called.
func (r *Runner) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.ctx != nil {
		r.mu.Unlock()
		r.logger.Warn("runner already started")
		return nil
	}
	if r.queue == nil {
		r.mu.Unlock()
		r.logger.Error("runner started without a queue")
		return fmt.Errorf("runner requires a queue")
	}
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.mu.Unlock()

	r.wg.Add(1)
	go r.run()

	r.logger.Info("runner started")
	return nil
}

// run is the main worker loop that processes jobs from the queue.
func (r *Runner) run() {
	defer r.wg.Done()
	r.logger.Info("runner worker started")

	for {
		// Wait for a job or context cancellation.
		jobID, ok := r.queue.DequeueBlocks(r.ctx)
		if !ok {
			// Queue closed or context cancelled.
			r.logger.Info("runner worker exiting loop")
			return
		}

		// Process the job.
		if err := r.ProcessJob(r.ctx, jobID); err != nil {
			r.logger.Error("job processing failed", "job_id", jobID, "error", err)
		}
	}
}

// ProcessJob processes a single job with precheck.
// It performs pre-run checks for source_sync jobs to ensure:
// 1. The source is enabled
// 2. At least one enabled device is subscribed to the source
//
// If precheck fails, the job is completed with an informational message
// and no actual work is performed.
func (r *Runner) ProcessJob(ctx context.Context, jobID string) error {
	r.logger.Info("processing job", "job_id", jobID)

	if r.jobsSvc == nil {
		return fmt.Errorf("jobs service not set")
	}

	// Parse job ID
	jobUUID, err := dbtypes.NewUUIDFromString(jobID)
	if err != nil {
		return fmt.Errorf("invalid job ID: %w", err)
	}

	// Perform precheck and start job
	job, canProceed, err := r.jobsSvc.PrecheckAndStartJob(ctx, jobUUID)
	if err != nil {
		return fmt.Errorf("precheck/start job: %w", err)
	}

	if !canProceed {
		r.logger.Info("job precheck failed, skipped",
			"job_id", jobID,
			"source_id", job.SourceID,
			"message", job.Message)
		return nil
	}

	r.logger.Info("job precheck passed, proceeding with work",
		"job_id", jobID,
		"source_id", job.SourceID)

	// TODO: Actual job work - fetch, download, materialize
	// Placeholder: simulate work
	time.Sleep(100 * time.Millisecond)

	// Complete the job
	msg := "Job completed successfully"
	_, err = r.jobsSvc.CompleteJob(ctx, jobs.CompleteJobRequest{
		ID:      jobUUID,
		Message: &msg,
	})
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}

	r.logger.Info("job completed", "job_id", jobID)
	return nil
}

// Stop gracefully stops the runner, waiting for the current job to finish.
func (r *Runner) Stop() error {
	r.mu.Lock()
	if r.cancel == nil {
		r.mu.Unlock()
		r.logger.Warn("runner not started")
		return nil
	}
	shouldCancel := true
	if r.queue != nil && r.queue.IsClosed() {
		shouldCancel = false
	}
	if shouldCancel {
		r.cancel()
	}
	r.mu.Unlock()

	r.logger.Info("runner stopping, waiting for worker to finish", "cancelled", shouldCancel)
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		r.logger.Info("runner stopped gracefully")
	case <-time.After(30 * time.Second):
		r.logger.Warn("runner stop timed out")
	}

	r.mu.Lock()
	r.ctx = nil
	r.cancel = nil
	r.mu.Unlock()

	return nil
}

// Run executes a single job synchronously (for manual trigger use cases).
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Info("runner run called (no-op in skeleton)")
	return nil
}
