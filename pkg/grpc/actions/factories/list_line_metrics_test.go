package factories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__ListLineMetrics(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid factory id -> error", func(t *testing.T) {
		_, err := ListLineMetrics(context.Background(), r.Organization.ID.String(), &pb.ListLineMetricsRequest{
			FactoryId: "not-a-uuid",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("negative window_days -> error", func(t *testing.T) {
		windowDays := int32(-1)
		_, err := ListLineMetrics(context.Background(), r.Organization.ID.String(), &pb.ListLineMetricsRequest{
			FactoryId:  uuid.NewString(),
			WindowDays: &windowDays,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("unknown factory -> not found", func(t *testing.T) {
		_, err := ListLineMetrics(context.Background(), r.Organization.ID.String(), &pb.ListLineMetricsRequest{
			FactoryId: uuid.NewString(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("returns aggregated metrics for lines with closed work orders", func(t *testing.T) {
		db := database.DB(t.Context())

		f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		line, err := f.CreateLine(db, "line-a", nil)
		require.NoError(t, err)
		otherLine, err := f.CreateLine(db, "line-b", nil)
		require.NoError(t, err)
		_ = otherLine // no closed work orders on this line: should be absent from the response

		canvasID, versionID := setupMetricsCanvas(t, db, r.Organization.ID)

		closedAt := time.Now().AddDate(0, 0, -1)
		order := createTestClosedWorkOrder(t, db, f, models.FactoryWorkOrderResultCompleted, closedAt)
		runID := createTestMetricsRun(t, db, canvasID, versionID, closedAt)
		createTestExecution(t, db, f, runID, order.ID, line.ID, 0, 500, closedAt)

		response, err := ListLineMetrics(context.Background(), r.Organization.ID.String(), &pb.ListLineMetricsRequest{
			FactoryId: f.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, response.Metrics, 1)

		metrics := response.Metrics[0]
		assert.Equal(t, line.ID.String(), metrics.LineId)
		assert.Equal(t, int64(1), metrics.MergedCount)
		assert.Equal(t, int64(1), metrics.TotalClosedCount)
		assert.Equal(t, 100.0, metrics.SuccessRatePct)
		assert.Equal(t, int64(500), metrics.CostPerSuccessCents)
	})
}

const testMetricsRunNodeID = "step"

// setupMetricsCanvas creates the minimum canvas + live version + node needed
// to satisfy the `workflow_runs` foreign keys behind an execution's run_id.
func setupMetricsCanvas(t *testing.T, db *gorm.DB, organizationID uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()

	now := time.Now()
	versionID := uuid.New()
	canvas := models.Canvas{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		LiveVersionID:  &versionID,
		Name:           support.RandomName("canvas"),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	version := models.CanvasVersion{
		ID:         versionID,
		WorkflowID: canvas.ID,
		Nodes:      datatypes.NewJSONSlice([]models.Node{}),
		Edges:      datatypes.NewJSONSlice([]models.Edge{}),
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	node := models.CanvasNode{
		WorkflowID:    canvas.ID,
		NodeID:        testMetricsRunNodeID,
		Name:          "Step",
		State:         models.CanvasNodeStateReady,
		Type:          models.NodeTypeComponent,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(models.Position{}),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	// live_version_id is a deferred FK (canvas <-> version are mutually
	// referential); both inserts must land in the same transaction.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&canvas).Error; err != nil {
			return err
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		return tx.Create(&node).Error
	}))

	return canvas.ID, versionID
}

func createTestMetricsRun(t *testing.T, db *gorm.DB, canvasID, versionID uuid.UUID, createdAt time.Time) uuid.UUID {
	t.Helper()

	run := &models.CanvasRun{
		ID:         uuid.New(),
		WorkflowID: canvasID,
		NodeID:     testMetricsRunNodeID,
		VersionID:  versionID,
		State:      models.CanvasRunStateFinished,
		CreatedAt:  &createdAt,
		UpdatedAt:  &createdAt,
	}
	require.NoError(t, db.Create(run).Error)
	return run.ID
}

func createTestClosedWorkOrder(t *testing.T, db *gorm.DB, f *models.Factory, result string, closedAt time.Time) *models.FactoryWorkOrder {
	t.Helper()

	order, err := f.CreateWorkOrder(db, "Order", "", nil, nil, nil)
	require.NoError(t, err)

	require.NoError(t, db.Model(order).Updates(map[string]any{
		"state":      models.FactoryWorkOrderStateClosed,
		"result":     result,
		"updated_at": closedAt,
	}).Error)
	order.State = models.FactoryWorkOrderStateClosed
	order.Result = result
	order.UpdatedAt = closedAt

	return order
}

func createTestExecution(t *testing.T, db *gorm.DB, f *models.Factory, runID, workOrderID, lineID uuid.UUID, stepIndex int, costCents int64, createdAt time.Time) {
	t.Helper()

	execution := &models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		WorkOrderID:    workOrderID,
		LineID:         lineID,
		StepIndex:      stepIndex,
		StepName:       "step",
		RunID:          runID,
		Status:         models.FactoryWorkOrderExecutionStatusFinished,
		Result:         models.FactoryWorkOrderResultCompleted,
		CostCents:      costCents,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	require.NoError(t, db.Create(execution).Error)
}
