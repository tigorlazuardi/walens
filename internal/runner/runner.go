package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/walens/walens/internal/db/generated/model"
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/queue"
	"github.com/walens/walens/internal/services/images"
	"github.com/walens/walens/internal/services/jobs"
	"github.com/walens/walens/internal/services/tags"
	"github.com/walens/walens/internal/sources"
	"github.com/walens/walens/internal/storage"
)

// Runner consumes jobs from the queue and processes them.
type Runner struct {
	logger         *slog.Logger
	queue          *queue.Queue
	jobsSvc        *jobs.Service
	sourceRegistry *sources.Registry
	materializer   *Materializer
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

type RunnerDependencies struct {
	Logger         *slog.Logger
	Queue          *queue.Queue
	JobsService    *jobs.Service
	StorageService *storage.Service
	ImageService   *images.Service
	TagsService    *tags.Service
	SourceRegistry *sources.Registry
}

// New creates a new job runner.
func New(deps RunnerDependencies) *Runner {
	if deps.Logger == nil {
		panic("runner.New: Logger is required")
	}
	if deps.Queue == nil {
		panic("runner.New: Queue is required")
	}
	if deps.JobsService == nil {
		panic("runner.New: JobsService is required")
	}
	if deps.StorageService == nil {
		panic("runner.New: StorageService is required")
	}
	if deps.ImageService == nil {
		panic("runner.New: ImageService is required")
	}
	if deps.TagsService == nil {
		panic("runner.New: TagsService is required")
	}
	if deps.SourceRegistry == nil {
		panic("runner.New: SourceRegistry is required")
	}
	return &Runner{
		logger:         deps.Logger,
		queue:          deps.Queue,
		jobsSvc:        deps.JobsService,
		sourceRegistry: deps.SourceRegistry,
		materializer: NewMaterializer(MaterializerDependencies{
			Logger:         deps.Logger,
			StorageService: deps.StorageService,
			ImageService:   deps.ImageService,
			JobsService:    deps.JobsService,
			TagsService:    deps.TagsService,
		}),
	}
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

	// For source_sync jobs, fetch and materialize images
	if job.JobType == jobs.JobTypeSourceSync && job.SourceID != nil {
		if err := r.processSourceSyncJob(ctx, job); err != nil {
			errMsg := err.Error()
			_, failErr := r.jobsSvc.FailJob(ctx, jobs.FailJobRequest{
				ID:           jobUUID,
				ErrorMessage: errMsg,
			})
			if failErr != nil {
				r.logger.Error("failed to mark job as failed", "error", failErr)
			}
			return fmt.Errorf("source sync job failed: %w", err)
		}
	}

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

// processSourceSyncJob handles the actual source fetch and image materialization.
func (r *Runner) processSourceSyncJob(ctx context.Context, job *model.Jobs) error {
	sourceID := *job.SourceID

	// Get source type from job or database
	sourceType := ""
	if job.SourceType != nil {
		sourceType = *job.SourceType
	}

	if sourceType == "" {
		return fmt.Errorf("source type is empty")
	}

	// Get source from registry
	src := r.sourceRegistry.Get(sourceType)
	if src == nil {
		return fmt.Errorf("unknown source type: %s", sourceType)
	}

	// Get subscribed devices
	devices, err := r.materializer.imageSvc.GetSubscribedDevices(ctx, sourceID)
	if err != nil {
		if errors.Is(err, images.ErrNoSubscribedDevices) {
			r.logger.Info("no subscribed devices for source", "source_id", sourceID)
			return nil
		}
		return fmt.Errorf("get subscribed devices: %w", err)
	}

	// Get source params from job input
	var sourceParams []byte
	if job.JSONInput != nil {
		sourceParams = []byte(job.JSONInput)
	}

	// Use default lookup count if not specified
	lookupCount := src.DefaultLookupCount()
	if job.RequestedImageCount > 0 {
		lookupCount = int(job.RequestedImageCount)
	}

	// Create materialize request
	matReq := MaterializeRequest{
		JobID:        job.ID,
		SourceID:     sourceID,
		SourceType:   sourceType,
		SourceParams: sourceParams,
		LookupCount:  lookupCount,
		Devices:      devices,
	}

	// Materialize images
	result, err := r.materializer.MaterializeImage(ctx, matReq, src)
	if err != nil {
		return fmt.Errorf("materialize images: %w", err)
	}

	r.logger.Info("source sync job completed",
		"job_id", job.ID,
		"downloaded", result.DownloadedCount,
		"hardlinked", result.HardlinkedCount,
		"copied", result.CopiedCount,
		"skipped", result.SkippedCount,
		"stored", result.StoredCount)

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
