package telemetry

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerErrorRecorder(t *testing.T) {
	t.Run("records the error for the request", func(t *testing.T) {
		ctx, recorder := WithServerErrorRecorder(context.Background())
		require.Nil(t, recorder.Err())

		boom := errors.New("boom")
		RecordServerError(ctx, boom)

		assert.Equal(t, boom, recorder.Err())
	})

	t.Run("keeps the first error", func(t *testing.T) {
		ctx, recorder := WithServerErrorRecorder(context.Background())

		rootCause := errors.New("column factories.key does not exist")
		RecordServerError(ctx, rootCause)
		RecordServerError(ctx, errors.New("internal server error"))

		assert.Equal(t, rootCause, recorder.Err())
	})

	t.Run("ignores nil errors", func(t *testing.T) {
		ctx, recorder := WithServerErrorRecorder(context.Background())

		RecordServerError(ctx, nil)

		assert.Nil(t, recorder.Err())
	})

	t.Run("is a no-op without a recorder in the context", func(t *testing.T) {
		assert.NotPanics(t, func() {
			RecordServerError(context.Background(), errors.New("boom"))
		})
	})

	t.Run("is safe for concurrent use", func(t *testing.T) {
		ctx, recorder := WithServerErrorRecorder(context.Background())

		var wg sync.WaitGroup
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				RecordServerError(ctx, errors.New("boom"))
				_ = recorder.Err()
			}()
		}
		wg.Wait()

		assert.Error(t, recorder.Err())
	})
}
