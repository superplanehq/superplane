package runner

import (
	"sync"

	"github.com/google/uuid"
)

type surveyWaitRegistry struct {
	mu      sync.Mutex
	waiters map[uuid.UUID][]chan struct{}
}

func newSurveyWaitRegistry() *surveyWaitRegistry {
	return &surveyWaitRegistry{waiters: make(map[uuid.UUID][]chan struct{})}
}

var workOrderSurveyWaiters = newSurveyWaitRegistry()

func SubscribeWorkOrderSurvey(id uuid.UUID) <-chan struct{} {
	return workOrderSurveyWaiters.subscribe(id)
}

func UnsubscribeWorkOrderSurvey(id uuid.UUID, ch <-chan struct{}) {
	workOrderSurveyWaiters.unsubscribe(id, ch)
}

func NotifyWorkOrderSurvey(id uuid.UUID) {
	workOrderSurveyWaiters.notify(id)
}

func (r *surveyWaitRegistry) subscribe(id uuid.UUID) <-chan struct{} {
	ch := make(chan struct{}, 1)
	r.mu.Lock()
	r.waiters[id] = append(r.waiters[id], ch)
	r.mu.Unlock()
	return ch
}

func (r *surveyWaitRegistry) unsubscribe(id uuid.UUID, ch <-chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.waiters[id]
	kept := list[:0]
	for _, waiter := range list {
		if waiter == ch {
			continue
		}
		kept = append(kept, waiter)
	}
	if len(kept) == 0 {
		delete(r.waiters, id)
		return
	}
	r.waiters[id] = kept
}

func (r *surveyWaitRegistry) notify(id uuid.UUID) {
	r.mu.Lock()
	waiters := append([]chan struct{}(nil), r.waiters[id]...)
	r.mu.Unlock()
	for _, waiter := range waiters {
		select {
		case waiter <- struct{}{}:
		default:
		}
	}
}
