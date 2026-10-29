package me

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestListModelStatusRequiresSessionAndRejectsUnknownWindows(t *testing.T) {
	_, err := ListModelStatus(context.Background(), &pb.ListModelStatusRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, grpcerrors.Code(err))

	r := support.Setup(t)
	_, err = ListModelStatus(meContext(r), &pb.ListModelStatusRequest{Window: "7d"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func TestListModelStatusReturnsScoredAndEmptyAllowlistCells(t *testing.T) {
	r := support.Setup(t)
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
			RunnerTaskID: "status-task-a",
			Sequence:     1,
		},
		{
			Provider:     models.UsageProviderAnthropic,
			Model:        "claude-sonnet-4-6",
			Outcome:      models.HostedModelCallOutcomeError,
			OccurredAt:   now.Add(-time.Hour),
			RunnerTaskID: "status-task-old",
			Sequence:     1,
		},
	}))

	response, err := ListModelStatus(meContext(r), &pb.ListModelStatusRequest{})
	require.NoError(t, err)
	assert.Equal(t, models.HostedModelCallOutcomeWindow5m, response.Window)
	require.Len(t, response.Cells, 2)

	opus := response.Cells[0]
	assert.Equal(t, models.UsageProviderAnthropic, opus.Provider)
	assert.Equal(t, "claude-opus-4-6", opus.Model)
	assert.Nil(t, opus.Score)
	assert.Zero(t, opus.CallCount)

	sonnet := response.Cells[1]
	assert.Equal(t, models.UsageProviderAnthropic, sonnet.Provider)
	assert.Equal(t, "claude-sonnet-4-6", sonnet.Model)
	require.NotNil(t, sonnet.Score)
	assert.Equal(t, int32(100), *sonnet.Score)
	assert.Equal(t, int32(1), sonnet.CallCount)
	assert.Equal(t, int32(1), sonnet.Attempt_1Count)
}
