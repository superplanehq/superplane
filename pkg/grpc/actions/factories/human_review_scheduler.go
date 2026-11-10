package factories

import (
	"context"
	"sync"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
)

var factoryHumanReviewScheduler = newHumanReviewScheduler(4, 100, reconcileFactoryHumanReview)

type humanReviewRequest struct {
	ctx  context.Context
	deps IntakeDependencies
	pr   models.FactoryPullRequest
}

type humanReviewJob struct {
	request humanReviewRequest
	running bool
	dirty   bool
}

// Webhooks and polling share a bounded queue. Repeated events replace queued
// snapshots or request one more pass after an in-flight provider read.
type humanReviewScheduler struct {
	mu          sync.Mutex
	concurrency int
	capacity    int
	workers     int
	jobs        map[uuid.UUID]*humanReviewJob
	queue       []*humanReviewJob
	reconcile   func(context.Context, IntakeDependencies, models.FactoryPullRequest) error
}

func newHumanReviewScheduler(concurrency, capacity int, reconcile func(context.Context, IntakeDependencies, models.FactoryPullRequest) error) *humanReviewScheduler {
	return &humanReviewScheduler{concurrency: concurrency, capacity: capacity, jobs: make(map[uuid.UUID]*humanReviewJob), reconcile: reconcile}
}

func (s *humanReviewScheduler) schedule(ctx context.Context, deps IntakeDependencies, pr models.FactoryPullRequest) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := humanReviewRequest{ctx: ctx, deps: deps, pr: pr}
	if job, ok := s.jobs[pr.ID]; ok {
		job.request = request
		job.dirty = job.running
		return true
	}
	if len(s.jobs) >= s.capacity {
		return false
	}
	job := &humanReviewJob{request: request}
	s.jobs[pr.ID] = job
	s.queue = append(s.queue, job)
	if s.workers < s.concurrency {
		s.workers++
		go s.run()
	}
	return true
}

func (s *humanReviewScheduler) run() {
	for {
		s.mu.Lock()
		if len(s.queue) == 0 {
			s.workers--
			s.mu.Unlock()
			return
		}
		job := s.queue[0]
		s.queue[0] = nil
		s.queue = s.queue[1:]
		job.running = true
		job.dirty = false
		request := job.request
		s.mu.Unlock()

		if err := s.reconcile(request.ctx, request.deps, request.pr); err != nil {
			log.WithError(err).WithField("pull_request", request.pr.ID).Warn("Human review reconciliation failed")
		}

		s.mu.Lock()
		job.running = false
		if job.dirty {
			s.queue = append(s.queue, job)
		} else {
			delete(s.jobs, request.pr.ID)
		}
		s.mu.Unlock()
	}
}
