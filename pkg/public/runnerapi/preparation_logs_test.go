package runnerapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	runnerlogsfs "github.com/superplanehq/superplane/pkg/runners/logs/fs"
)

func TestRunnerLogInitializationUsesTheDurablePreparationBudget(t *testing.T) {
	for _, exhaustBudget := range []bool{false, true} {
		name := "retry storage outage"
		if exhaustBudget {
			name = "fail at durable deadline"
		}
		t.Run(name, func(t *testing.T) {
			fixture := createRunnerProviderFixture(t)
			db := database.DB(t.Context())
			reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
			require.NoError(t, err)
			fixture.server.registry = reg
			fixture.server.activeLogStoreName = runnerlogs.StoreFS
			store, err := runnerlogsfs.New(t.TempDir())
			require.NoError(t, err)
			failing := &unavailableLogInitialization{Store: store, waitForDeadline: exhaustBudget}
			previous := runnerlogs.Current()
			runnerlogs.SetCurrent(failing)
			t.Cleanup(func() { runnerlogs.SetCurrent(previous) })
			now := time.Now()
			require.NoError(t, fixture.preparation.Retry(db, now, now))
			deadline := fixture.preparation.Deadline
			if exhaustBudget {
				deadline = now.Add(100 * time.Millisecond)
				require.NoError(t, db.Model(fixture.preparation).Update("deadline", deadline).Error)
			}

			fixture.server.prepareAndDeliver(t.Context(), &runnerConnection{}, fixture.runnerID, fixture.connectionID, fixture.task.ID)

			require.Equal(t, 1, failing.calls)
			require.True(t, failing.deadline.After(now))
			require.False(t, failing.deadline.After(deadline), "log storage must share the clipped preparation deadline")
			require.LessOrEqual(t, failing.remaining, models.RunnerPreparationAttemptTimeout)
			task, err := models.FindRunnerTask(db, fixture.task.ID)
			require.NoError(t, err)
			require.Nil(t, task.StartedAt)
			require.Nil(t, task.DispatchStartedAt)
			runner, err := models.FindRunner(db, fixture.runnerID)
			require.NoError(t, err)
			require.Equal(t, models.RunnerStateIdle, runner.State)
			if exhaustBudget {
				require.Equal(t, models.RunnerTaskStateFailed, task.State)
				return
			}
			require.Equal(t, models.RunnerTaskStateReserved, task.State)
			var preparation models.RunnerTaskPreparation
			require.NoError(t, db.First(&preparation, "task_id = ?", task.ID).Error)
			require.Nil(t, preparation.PreparedAt, "log failure must not accept a cached payload")
			require.Nil(t, preparation.AttemptExpiresAt)
			require.True(t, preparation.NextRetryAt.After(now))
			require.WithinDuration(t, deadline, preparation.Deadline, time.Microsecond)
		})
	}
}

type unavailableLogInitialization struct {
	runnerlogs.Store
	waitForDeadline bool
	calls           int
	deadline        time.Time
	remaining       time.Duration
}

func (s *unavailableLogInitialization) Initialize(ctx context.Context, _ uuid.UUID) error {
	s.calls++
	s.deadline, _ = ctx.Deadline()
	s.remaining = time.Until(s.deadline)
	if s.waitForDeadline {
		<-ctx.Done()
		return ctx.Err()
	}
	return errors.New("log storage unavailable")
}
