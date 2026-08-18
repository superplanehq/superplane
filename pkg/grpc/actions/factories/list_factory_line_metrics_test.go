package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"

	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
)

func Test__ListFactoryLineMetrics(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	t.Run("invalid factory id -> error", func(t *testing.T) {
		_, err := ListFactoryLineMetrics(context.Background(), r.Organization.ID.String(), &pb.ListFactoryLineMetricsRequest{
			FactoryId: "not-a-uuid",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("unknown factory -> not found", func(t *testing.T) {
		_, err := ListFactoryLineMetrics(context.Background(), r.Organization.ID.String(), &pb.ListFactoryLineMetricsRequest{
			FactoryId: "00000000-0000-0000-0000-000000000001",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("returns a row per line", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateLine(db, "ship", nil)
		require.NoError(t, err)
		_, err = factoryModel.CreateLine(db, "hotfix", nil)
		require.NoError(t, err)

		response, err := ListFactoryLineMetrics(ctx, r.Organization.ID.String(), &pb.ListFactoryLineMetricsRequest{
			FactoryId: factoryModel.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, response.Lines, 2)
		assert.Empty(t, response.Lines[0].Metrics)
		assert.Empty(t, response.Lines[1].Metrics)
	})
}

func TestSerializeFactoryLineMetricsOmitsAbsentRows(t *testing.T) {
	absentID := "11111111-1111-4111-8111-111111111111"
	presentID := "22222222-2222-4222-8222-222222222222"
	got := serializeFactoryLineMetrics([]models.FactoryLineMetrics{
		{LineID: mustUUID(t, absentID), Present: false},
		{
			LineID:             mustUUID(t, presentID),
			Present:            true,
			SuccessRatePct:     50,
			MergedCount:        1,
			TotalClosedCount:   2,
			ReworkPerWorkOrder: 1,
			CostPerSuccessUsd:  4,
			SuccessTrendPct:    []float64{0, 50},
			SuccessDeltaPts:    50,
			ThroughputPerDay:   1.0 / 30.0,
			ThroughputTrend:    []int{0, 1},
		},
	})

	require.Len(t, got, 2)
	assert.Equal(t, absentID, got[0].LineId)
	assert.Nil(t, got[0].Metrics)
	assert.Equal(t, presentID, got[1].LineId)
	require.NotNil(t, got[1].Metrics)
	assert.Equal(t, 50.0, got[1].Metrics.SuccessRatePct)
	assert.Equal(t, int32(1), got[1].Metrics.MergedCount)
	assert.Equal(t, []int32{0, 1}, got[1].Metrics.ThroughputTrend)
}

func mustUUID(t *testing.T, value string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(value)
	require.NoError(t, err)
	return id
}
