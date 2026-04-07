package runner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/walens/walens/internal/queue"
)

// Runner consumes jobs from the queue and processes them.
type Runner struct {
	logger *slog.Logger
	queue  *queue.Queue
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
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

// ProcessJob processes a single job. This is a placeholder implementation.
func (r *Runner) ProcessJob(ctx context.Context, jobID string) error {
	r.logger.Info("processing job", "job_id", jobID)
	// Placeholder: load job from DB, resolve source, fetch, download, materialize.
	// Simulate work so skeleton shows real timing.
	time.Sleep(100 * time.Millisecond)
	r.logger.Debug("job processed", "job_id", jobID)
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
