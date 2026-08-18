package requesterror

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecord(t *testing.T) {
	t.Run("returns the recorded error", func(t *testing.T) {
		ctx, recorder := NewContext(context.Background())
		err := errors.New("db down")

		Record(ctx, err)

		assert.Equal(t, err, recorder.Err())
	})

	t.Run("keeps the first error", func(t *testing.T) {
		ctx, recorder := NewContext(context.Background())
		first := errors.New("db down")

		Record(ctx, first)
		Record(ctx, errors.New("later error"))

		assert.Equal(t, first, recorder.Err())
	})

	t.Run("ignores a nil error", func(t *testing.T) {
		ctx, recorder := NewContext(context.Background())

		Record(ctx, nil)

		assert.NoError(t, recorder.Err())
	})

	t.Run("does nothing without a recorder in the context", func(t *testing.T) {
		require.NotPanics(t, func() {
			Record(context.Background(), errors.New("db down"))
		})
	})

	t.Run("records from a derived context", func(t *testing.T) {
		ctx, recorder := NewContext(context.Background())
		err := errors.New("db down")

		type otherKey struct{}
		Record(context.WithValue(ctx, otherKey{}, "value"), err)

		assert.Equal(t, err, recorder.Err())
	})
}

func TestRecorderErr(t *testing.T) {
	t.Run("returns nil when nothing is recorded", func(t *testing.T) {
		_, recorder := NewContext(context.Background())

		assert.NoError(t, recorder.Err())
	})

	t.Run("returns nil for a nil recorder", func(t *testing.T) {
		var recorder *Recorder

		assert.NoError(t, recorder.Err())
	})
}

func TestRecordIsSafeForConcurrentUse(t *testing.T) {
	ctx, recorder := NewContext(context.Background())

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Record(ctx, fmt.Errorf("error %d", i))
		}()
	}
	wg.Wait()

	assert.Error(t, recorder.Err())
}
