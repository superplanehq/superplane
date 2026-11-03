package workers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestHostedModelCallOutcomeCleanupWorkerDeletesRowsOlderThanRetention(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = db.Where("runner_task_id IN ?", []string{"cleanup-old", "cleanup-new"}).Delete(&models.HostedModelCallOutcome{}).Error
	})

	now := time.Now()
	require.NoError(t, models.InsertHostedModelCallOutcomes(db, []models.HostedModelCallOutcomeInput{
		{
			Provider:     models.UsageProviderOpenAI,
			Model:        "gpt-5",
			Outcome:      models.HostedModelCallOutcomeAttempt1,
			OccurredAt:   now.Add(-26 * time.Hour),
			RunnerTaskID: "cleanup-old",
			Sequence:     1,
		},
		{
			Provider:     models.UsageProviderOpenAI,
			Model:        "gpt-5",
			Outcome:      models.HostedModelCallOutcomeAttempt1,
			OccurredAt:   now.Add(-time.Hour),
			RunnerTaskID: "cleanup-new",
			Sequence:     1,
		},
	}))

	worker := NewHostedModelCallOutcomeCleanupWorker()
	worker.now = func() time.Time { return now }
	worker.tick(context.Background())

	var oldCount int64
	require.NoError(t, db.Model(&models.HostedModelCallOutcome{}).Where("runner_task_id = ?", "cleanup-old").Count(&oldCount).Error)
	assert.Zero(t, oldCount)

	var newCount int64
	require.NoError(t, db.Model(&models.HostedModelCallOutcome{}).Where("runner_task_id = ?", "cleanup-new").Count(&newCount).Error)
	assert.Equal(t, int64(1), newCount)
}
