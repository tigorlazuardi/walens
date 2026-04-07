package queue

import (
	"context"
	"log/slog"
	"sync"
)

// Queue is an in-memory job queue with blocking dequeue and graceful shutdown.
// Uses a channel for job delivery with mutex-protected size tracking.
type Queue struct {
	logger *slog.Logger
	mu     sync.Mutex
	jobs   []string // for drain/Size visibility
	closed bool
	ch     chan string // unbuffered channel for job delivery
}

// New creates a new in-memory queue.
func New(logger *slog.Logger) *Queue {
	return &Queue{
		logger: logger,
		jobs:   make([]string, 0),
		closed: false,
		ch:     make(chan string),
	}
}

// Enqueue adds a job ID to the queue. Thread-safe.
// Drops silently if queue is closed.
func (q *Queue) Enqueue(jobID string) {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		q.logger.Warn("enqueue attempted on closed queue", "job_id", jobID)
		return
	}
	q.jobs = append(q.jobs, jobID)
	q.logger.Debug("job enqueued", "job_id", jobID, "queue_size", len(q.jobs))
	q.mu.Unlock()

	// Non-blocking send to channel; receiver always waits via <-q.ch.
	// If receiver is not yet waiting, the job sits in q.jobs until they call DequeueBlocks.
	// We signal via the channel to wake them up.
	select {
	case q.ch <- jobID:
		// receiver was waiting, sent successfully
	default:
		// receiver not waiting yet; job is already in q.jobs, channel already had capacity
	}
}

// DequeueBlocks waits for a job ID, respecting context cancellation.
// Returns ("", false) when context is cancelled, times out, or queue is closed with no jobs.
func (q *Queue) DequeueBlocks(ctx context.Context) (string, bool) {
	for {
		q.mu.Lock()
		if len(q.jobs) > 0 {
			jobID := q.jobs[0]
			q.jobs = q.jobs[1:]
			q.logger.Debug("job dequeued", "job_id", jobID, "remaining", len(q.jobs))
			q.mu.Unlock()
			return jobID, true
		}
		if q.closed {
			q.mu.Unlock()
			q.logger.Debug("dequeue on closed empty queue")
			return "", false
		}
		q.mu.Unlock()

		// Wait for a job signal or context cancellation.
		select {
		case jobID := <-q.ch:
			q.mu.Lock()
			// The job was sent via channel; it should already be in q.jobs from Enqueue.
			// But due to race, the job might have been already taken by another dequeue.
			// So we search for it in the slice.
			found := false
			for i, j := range q.jobs {
				if j == jobID {
					q.jobs = append(q.jobs[:i], q.jobs[i+1:]...)
					found = true
					break
				}
			}
			if !found {
				// Another dequeue already took it; loop to wait again.
				q.mu.Unlock()
				continue
			}
			q.logger.Debug("job dequeued via channel", "job_id", jobID, "remaining", len(q.jobs))
			q.mu.Unlock()
			return jobID, true

		case <-ctx.Done():
			q.logger.Debug("dequeue context cancelled")
			return "", false
		}
	}
}

// Dequeue is a non-blocking dequeue. Returns ("", false) if empty or closed.
func (q *Queue) Dequeue() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.jobs) == 0 {
		return "", false
	}
	jobID := q.jobs[0]
	q.jobs = q.jobs[1:]
	q.logger.Debug("job dequeued", "job_id", jobID, "remaining", len(q.jobs))
	return jobID, true
}

// Size returns the current number of jobs in the queue.
func (q *Queue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

// Drain removes all jobs from the queue without stopping it.
func (q *Queue) Drain() {
	q.mu.Lock()
	defer q.mu.Unlock()
	count := len(q.jobs)
	q.jobs = make([]string, 0)
	q.logger.Info("queue drained", "job_count", count)
}

// Close gracefully closes the queue, signaling no more enqueues.
// Jobs already in the queue remain for dequeue until drained.
func (q *Queue) Close() {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	q.closed = true
	q.logger.Info("queue closed", "pending_jobs", len(q.jobs))
	q.mu.Unlock()

	// Wake up any dequeue blocker; they will see closed=true after jobs are drained.
	close(q.ch)
}

// IsClosed returns true if the queue has been closed.
func (q *Queue) IsClosed() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.closed
}
