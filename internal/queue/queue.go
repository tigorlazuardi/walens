package queue

import (
	"log/slog"
	"sync"
)

type Queue struct {
	logger *slog.Logger
	mu     sync.Mutex
	jobs   []string
}

func New(logger *slog.Logger) *Queue {
	return &Queue{
		logger: logger,
		jobs:   make([]string, 0),
	}
}

func (q *Queue) Enqueue(jobID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = append(q.jobs, jobID)
	q.logger.Debug("job enqueued", "job_id", jobID)
}

func (q *Queue) Dequeue() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.jobs) == 0 {
		return "", false
	}
	jobID := q.jobs[0]
	q.jobs = q.jobs[1:]
	return jobID, true
}

func (q *Queue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

func (q *Queue) Drain() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = make([]string, 0)
}
