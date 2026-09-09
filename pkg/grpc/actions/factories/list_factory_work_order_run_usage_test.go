package factories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListFactoryWorkOrderRunUsage(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Publish draft", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}, nil))

	var execution *models.FactoryWorkOrderExecution
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, result, dispatchErr := line.Dispatch(tx, order)
		if dispatchErr != nil {
			return dispatchErr
		}
		execution = result.Execution
		return nil
	}))
	require.NotNil(t, execution.RunID)

	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		FundingSource:   models.UsageFundingSourceBYOK,
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}))
	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		FleetID:         "e1-large-amd64",
		DurationSeconds: 90,
		IdempotencyKey:  "runner:compute:run-usage-handler:" + uuid.New().String(),
	}))

	resp, err := ListFactoryWorkOrderRunUsage(context.Background(), r.Organization.ID.String(), &pb.ListFactoryWorkOrderRunUsageRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)
	require.Equal(t, uint32(1), resp.TotalCount)
	require.Len(t, resp.Rows, 1)
	row := resp.Rows[0]
	assert.Equal(t, execution.ID.String(), row.WorkOrderExecutionId)
	assert.Equal(t, factory.WorkOrderKey(order.Number), row.WorkOrderKey)
	assert.Equal(t, "Publish draft", row.Title)
	assert.Equal(t, r.UserModel.GetEmail(), row.UserEmail)
	assert.Equal(t, int64(1_000_000), row.TotalTokens)
	assert.Equal(t, int64(90), row.DurationSeconds)
	assert.True(t, row.UsedByok)
	assert.Contains(t, row.Models, "anthropic/claude-sonnet-4-6")
	assert.Contains(t, row.MachineTypes, "e1-large-amd64")
	assert.Positive(t, row.CostCents)
}

func Test__ListFactoryWorkOrderRunUsage__RejectsInvalidWindow(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	_, err = ListFactoryWorkOrderRunUsage(context.Background(), r.Organization.ID.String(), &pb.ListFactoryWorkOrderRunUsageRequest{
		FactoryId: factory.ID.String(),
		StartTime: timestamppb.New(now),
		EndTime:   timestamppb.New(now.Add(-time.Hour)),
	})
	require.Error(t, err)
}

func Test__ClampWorkOrderRunUsagePageSize(t *testing.T) {
	assert.Equal(t, workOrderRunUsagePageSizeDefault, clampWorkOrderRunUsagePageSize(0))
	assert.Equal(t, workOrderRunUsagePageSizeMax, clampWorkOrderRunUsagePageSize(500))
	assert.Equal(t, 10, clampWorkOrderRunUsagePageSize(10))
}
