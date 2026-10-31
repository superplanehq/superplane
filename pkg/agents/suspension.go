package agents

import (
	"context"
	"errors"
	"fmt"
)

// SessionStopObserver confirms stopping independently of interrupt acceptance.
type SessionStopObserver interface {
	SessionStopped(context.Context, string) (bool, error)
}

// StopSuspendedSession performs no local state transitions. The caller retains
// durable stopping intent and checks that local stream ownership has ended.
func (s *Service) StopSuspendedSession(ctx context.Context, providerSessionID string) (bool, error) {
	observer, ok := s.provider.(SessionStopObserver)
	if !ok {
		return false, fmt.Errorf("agent provider cannot confirm session stopping")
	}
	if err := s.provider.InterruptSession(ctx, providerSessionID); err != nil && !errors.Is(err, ErrProviderSessionUnavailable) {
		return false, err
	}
	return observer.SessionStopped(ctx, providerSessionID)
}
