package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__DescribeFactory_AttachesLineMetrics(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)
	now := time.Now().In(time.Local)

	originalNow := timeNow
	timeNow = func() time.Time { return now }
	t.Cleanup(func() { timeNow = originalNow })

	t.Run("idle line has no metrics", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateLine(db, "idle", nil)
		require.NoError(t, err)

		resp, err := DescribeFactory(ctx, r.Organization.ID.String(), factoryModel.ID.String())
		require.NoError(t, err)
		require.Len(t, resp.Factory.Lines, 1)
		assert.Nil(t, resp.Factory.Lines[0].Metrics)
	})

	t.Run("closed work orders attach metrics to that line only", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		busy, err := factoryModel.CreateLine(db, "busy", nil)
		require.NoError(t, err)
		idle, err := factoryModel.CreateLine(db, "idle", nil)
		require.NoError(t, err)

		order, err := factoryModel.CreateWorkOrder(db, "Closed", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
		require.NoError(t, err)
		createDescribeMetricsExecution(t, db, r, factoryModel.ID, order.ID, busy.ID, busy.Name, now.Add(-2*time.Hour))
		_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateClosed,
			Result:  models.FactoryWorkOrderResultCompleted,
		})
		require.NoError(t, err)

		resp, err := DescribeFactory(ctx, r.Organization.ID.String(), factoryModel.ID.String())
		require.NoError(t, err)
		require.Len(t, resp.Factory.Lines, 2)

		busyLine := factoryLineByID(resp.Factory.Lines, busy.ID.String())
		idleLine := factoryLineByID(resp.Factory.Lines, idle.ID.String())
		require.NotNil(t, busyLine)
		require.NotNil(t, idleLine)
		require.NotNil(t, busyLine.Metrics)
		assert.Equal(t, int32(1), busyLine.Metrics.TotalClosedCount)
		assert.Equal(t, int32(1), busyLine.Metrics.MergedCount)
		assert.Equal(t, 100.0, busyLine.Metrics.SuccessRatePct)
		require.NotNil(t, busyLine.Metrics.DurationMinutes)
		assert.InDelta(t, 120.0, *busyLine.Metrics.DurationMinutes, 1.0)
		assert.Nil(t, busyLine.Metrics.CostPerSuccessUsd)
		assert.Nil(t, idleLine.Metrics)
	})
}

func factoryLineByID(lines []*pb.FactoryLine, id string) *pb.FactoryLine {
	for _, line := range lines {
		if line.GetId() == id {
			return line
		}
	}
	return nil
}

func createDescribeMetricsRun(t *testing.T, db *gorm.DB, r *support.ResourceRegistry) uuid.UUID {
	t.Helper()
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		nil,
	)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	var run *models.CanvasRun
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, rootEvent)
		if err != nil {
			return err
		}
		return rootEvent.RoutedInTransaction(tx)
	}))
	return run.ID
}

func createDescribeMetricsExecution(
	t *testing.T,
	db *gorm.DB,
	r *support.ResourceRegistry,
	factoryID, workOrderID, lineID uuid.UUID,
	lineName string,
	createdAt time.Time,
) {
	t.Helper()
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryID, workOrderID, lineID, lineName, nil)
	runID := createDescribeMetricsRun(t, db, r)
	execution := models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryID,
		WorkOrderID:    workOrderID,
		LineID:         lineID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       lineName,
		RunID:          &runID,
		Status:         models.FactoryWorkOrderExecutionStatusFinished,
		Result:         models.CanvasRunResultPassed,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	require.NoError(t, db.Create(&execution).Error)
}
