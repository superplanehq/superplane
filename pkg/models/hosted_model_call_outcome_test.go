package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestScoreHostedModelCallOutcomes(t *testing.T) {
	assert.Nil(t, models.ScoreHostedModelCallOutcomes(0, 0, 0, 0, 0))

	score := models.ScoreHostedModelCallOutcomes(1, 0, 0, 0, 0)
	require.NotNil(t, score)
	assert.Equal(t, 100, *score)

	score = models.ScoreHostedModelCallOutcomes(0, 0, 0, 0, 1)
	require.NotNil(t, score)
	assert.Equal(t, 0, *score)

	score = models.ScoreHostedModelCallOutcomes(1, 0, 0, 0, 1)
	require.NotNil(t, score)
	assert.Equal(t, 50, *score)

	score = models.ScoreHostedModelCallOutcomes(2, 0, 0, 0, 1)
	require.NotNil(t, score)
	assert.Equal(t, 67, *score)
}

func TestHostedModelCallOutcomeWindowDuration(t *testing.T) {
	window, duration, ok := models.HostedModelCallOutcomeWindowDuration("")
	assert.True(t, ok)
	assert.Equal(t, models.HostedModelCallOutcomeWindow5m, window)
	assert.Equal(t, 5*time.Minute, duration)

	window, duration, ok = models.HostedModelCallOutcomeWindowDuration("24h")
	assert.True(t, ok)
	assert.Equal(t, models.HostedModelCallOutcomeWindow24h, window)
	assert.Equal(t, 24*time.Hour, duration)

	_, _, ok = models.HostedModelCallOutcomeWindowDuration("7d")
	assert.False(t, ok)
}

func TestListHostedModelStatusCellsMergesAllowlistAndRecentCalls(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = db.Where("1 = 1").Delete(&models.HostedModelCallOutcome{}).Error
		_ = db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{}).Error
	})

	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, models.InsertHostedModelCallOutcomes(db, []models.HostedModelCallOutcomeInput{
		{
			Provider:     models.UsageProviderAnthropic,
			Model:        "claude-sonnet-4-6",
			Outcome:      models.HostedModelCallOutcomeAttempt1,
			OccurredAt:   now.Add(-time.Minute),
			RunnerTaskID: "task-a",
			Sequence:     1,
		},
		{
			Provider:     models.UsageProviderOpenRouter,
			Model:        "x-ai/grok-4.6",
			Outcome:      models.HostedModelCallOutcomeError,
			OccurredAt:   now.Add(-time.Minute),
			RunnerTaskID: "task-b",
			Sequence:     1,
		},
		{
			Provider:     models.UsageProviderAnthropic,
			Model:        "claude-sonnet-4-6",
			Outcome:      models.HostedModelCallOutcomeError,
			OccurredAt:   now.Add(-time.Hour),
			RunnerTaskID: "task-old",
			Sequence:     1,
		},
	}))
	require.NoError(t, models.InsertHostedModelCallOutcomes(db, []models.HostedModelCallOutcomeInput{
		{
			Provider:     models.UsageProviderAnthropic,
			Model:        "claude-sonnet-4-6",
			Outcome:      models.HostedModelCallOutcomeAttempt1,
			OccurredAt:   now.Add(-time.Minute),
			RunnerTaskID: "task-a",
			Sequence:     1,
		},
	}))

	cells, err := models.ListHostedModelStatusCells(db, now.Add(-5*time.Minute))
	require.NoError(t, err)
	require.Len(t, cells, 3)

	opus := cells[0]
	assert.Equal(t, models.UsageProviderAnthropic, opus.Provider)
	assert.Equal(t, "claude-opus-4-6", opus.Model)
	assert.Nil(t, opus.Score)
	assert.Zero(t, opus.CallCount)

	sonnet := cells[1]
	assert.Equal(t, models.UsageProviderAnthropic, sonnet.Provider)
	assert.Equal(t, "claude-sonnet-4-6", sonnet.Model)
	require.NotNil(t, sonnet.Score)
	assert.Equal(t, 100, *sonnet.Score)
	assert.Equal(t, 1, sonnet.CallCount)

	grok := cells[2]
	assert.Equal(t, models.UsageProviderOpenRouter, grok.Provider)
	assert.Equal(t, "x-ai/grok-4.6", grok.Model)
	require.NotNil(t, grok.Score)
	assert.Equal(t, 0, *grok.Score)
	assert.Equal(t, 1, grok.CallCount)
}

func TestDeleteExpiredHostedModelCallOutcomes(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = db.Where("1 = 1").Delete(&models.HostedModelCallOutcome{}).Error
	})

	now := time.Now()
	require.NoError(t, models.InsertHostedModelCallOutcomes(db, []models.HostedModelCallOutcomeInput{
		{
			Provider:     models.UsageProviderOpenAI,
			Model:        "gpt-5",
			Outcome:      models.HostedModelCallOutcomeAttempt1,
			OccurredAt:   now.Add(-26 * time.Hour),
			RunnerTaskID: "old-task",
			Sequence:     1,
		},
		{
			Provider:     models.UsageProviderOpenAI,
			Model:        "gpt-5",
			Outcome:      models.HostedModelCallOutcomeAttempt1,
			OccurredAt:   now.Add(-time.Hour),
			RunnerTaskID: "new-task",
			Sequence:     1,
		},
	}))

	deleted, err := models.DeleteExpiredHostedModelCallOutcomes(db, now.Add(-25*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	cells, err := models.ListHostedModelStatusCells(db, now.Add(-24*time.Hour))
	require.NoError(t, err)
	require.Len(t, cells, 1)
	assert.Equal(t, "gpt-5", cells[0].Model)
	assert.Equal(t, 1, cells[0].CallCount)
}
