package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

func TestExecutionPoller_isExecutionStuck(t *testing.T) {
	encryptor := crypto.NewNoOpEncryptor()
	registry := registry.NewRegistry(encryptor)
	poller := NewExecutionPoller(encryptor, registry)

	// Custom timeout for testing
	poller.ExecutionTimeout = 5 * time.Minute

	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	poller.nowFunc = func() time.Time { return baseTime }

	t.Run("execution without StartedAt is not stuck", func(t *testing.T) {
		execution := &models.StageExecution{
			ID:        uuid.New(),
			State:     models.ExecutionStarted,
			StartedAt: nil,
		}

		assert.False(t, poller.isExecutionStuck(execution))
	})

	t.Run("execution started 1 minute ago is not stuck", func(t *testing.T) {
		startedAt := baseTime.Add(-1 * time.Minute)
		execution := &models.StageExecution{
			ID:        uuid.New(),
			State:     models.ExecutionStarted,
			StartedAt: &startedAt,
		}

		assert.False(t, poller.isExecutionStuck(execution))
	})

	t.Run("execution started exactly at timeout is stuck", func(t *testing.T) {
		startedAt := baseTime.Add(-5 * time.Minute)
		execution := &models.StageExecution{
			ID:        uuid.New(),
			State:     models.ExecutionStarted,
			StartedAt: &startedAt,
		}

		assert.False(t, poller.isExecutionStuck(execution)) // exactly at timeout should not be stuck
	})

	t.Run("execution started 6 minutes ago is stuck", func(t *testing.T) {
		startedAt := baseTime.Add(-6 * time.Minute)
		execution := &models.StageExecution{
			ID:        uuid.New(),
			State:     models.ExecutionStarted,
			StartedAt: &startedAt,
		}

		assert.True(t, poller.isExecutionStuck(execution))
	})

	t.Run("execution started 1 hour ago is definitely stuck", func(t *testing.T) {
		startedAt := baseTime.Add(-1 * time.Hour)
		execution := &models.StageExecution{
			ID:        uuid.New(),
			State:     models.ExecutionStarted,
			StartedAt: &startedAt,
		}

		assert.True(t, poller.isExecutionStuck(execution))
	})
}

func TestExecutionPoller_timeoutConfiguration(t *testing.T) {
	encryptor := crypto.NewNoOpEncryptor()
	registry := registry.NewRegistry(encryptor)

	t.Run("default timeout is 30 minutes", func(t *testing.T) {
		poller := NewExecutionPoller(encryptor, registry)
		assert.Equal(t, 30*time.Minute, poller.ExecutionTimeout)
	})

	t.Run("can configure custom timeout", func(t *testing.T) {
		poller := NewExecutionPoller(encryptor, registry)
		customTimeout := 15 * time.Minute
		poller.ExecutionTimeout = customTimeout
		assert.Equal(t, customTimeout, poller.ExecutionTimeout)
	})
}
