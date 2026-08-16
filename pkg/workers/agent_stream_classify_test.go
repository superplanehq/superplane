package workers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClassifyTurnError(t *testing.T) {
	background := context.Background()

	expiredTurn, cancelExpired := context.WithTimeout(background, time.Nanosecond)
	defer cancelExpired()
	<-expiredTurn.Done()

	canceledParent, cancelParent := context.WithCancel(background)
	cancelParent()

	liveTurn, cancelLive := context.WithTimeout(background, time.Hour)
	defer cancelLive()

	providerErr := errors.New("provider exploded")

	tests := []struct {
		name      string
		parentCtx context.Context
		turnCtx   context.Context
		err       error
		want      error
	}{
		{
			name:      "clean end is not an error even during shutdown",
			parentCtx: canceledParent,
			turnCtx:   liveTurn,
			err:       nil,
			want:      nil,
		},
		{
			name:      "expired turn deadline fails the turn",
			parentCtx: background,
			turnCtx:   expiredTurn,
			err:       context.DeadlineExceeded,
			want:      errTurnDeadlineExpired,
		},
		{
			name:      "expired turn deadline wins over a racing shutdown",
			parentCtx: canceledParent,
			turnCtx:   expiredTurn,
			err:       context.DeadlineExceeded,
			want:      errTurnDeadlineExpired,
		},
		{
			name:      "shutdown with a truncated stream has no outcome",
			parentCtx: canceledParent,
			turnCtx:   liveTurn,
			err:       context.Canceled,
			want:      errWorkerShutdown,
		},
		{
			name:      "provider-internal deadline is a plain provider error, not the turn deadline",
			parentCtx: background,
			turnCtx:   liveTurn,
			err:       context.DeadlineExceeded,
			want:      context.DeadlineExceeded,
		},
		{
			name:      "provider error passes through",
			parentCtx: background,
			turnCtx:   liveTurn,
			err:       providerErr,
			want:      providerErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyTurnError(tt.parentCtx, tt.turnCtx, tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
