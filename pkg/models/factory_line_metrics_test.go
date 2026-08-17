package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const metricsRunNodeID = "step"

func TestFactory_ListClosedWorkOrderMetricsRows(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, userID, factoryModel := setupFactoryWithUser(t, "line-metrics")
	tx := database.Conn()
	canvasID, versionID := setupCanvasForMetricsTest(t, tx, organization.ID, userID)
	newRun := func(createdAt time.Time) uuid.UUID {
		return createMetricsRun(t, tx, canvasID, versionID, createdAt)
	}

	lineA, err := factoryModel.CreateLine(tx, "line-a", nil)
	require.NoError(t, err)
	lineB, err := factoryModel.CreateLine(tx, "line-b", nil)
	require.NoError(t, err)

	since := time.Now().AddDate(0, 0, -60)

	t.Run("attributes to the line of the most recent execution", func(t *testing.T) {
		order := createClosedWorkOrder(t, tx, factoryModel, userID, FactoryWorkOrderResultCompleted, time.Now().AddDate(0, 0, -1))
		createMetricsExecution(t, tx, factoryModel, newRun(time.Now().AddDate(0, 0, -3)), order.ID, lineA.ID, 0, 0, time.Now().AddDate(0, 0, -3))
		createMetricsExecution(t, tx, factoryModel, newRun(time.Now().AddDate(0, 0, -2)), order.ID, lineB.ID, 1, 0, time.Now().AddDate(0, 0, -2))

		rows, err := factoryModel.ListClosedWorkOrderMetricsRows(tx, since)
		require.NoError(t, err)
		row := findMetricsRow(t, rows, order.ID)
		assert.Equal(t, lineB.ID, row.LineID, "should attribute to the *latest* execution's line")
	})

	t.Run("sums cost across all executions of the work order", func(t *testing.T) {
		order := createClosedWorkOrder(t, tx, factoryModel, userID, FactoryWorkOrderResultCompleted, time.Now().AddDate(0, 0, -1))
		createMetricsExecution(t, tx, factoryModel, newRun(time.Now().AddDate(0, 0, -3)), order.ID, lineA.ID, 0, 100, time.Now().AddDate(0, 0, -3))
		createMetricsExecution(t, tx, factoryModel, newRun(time.Now().AddDate(0, 0, -2)), order.ID, lineA.ID, 1, 250, time.Now().AddDate(0, 0, -2))

		rows, err := factoryModel.ListClosedWorkOrderMetricsRows(tx, since)
		require.NoError(t, err)
		row := findMetricsRow(t, rows, order.ID)
		assert.Equal(t, int64(350), row.CostCents)
	})

	t.Run("counts rework: user comments, back-to-draft, and restarts", func(t *testing.T) {
		order := createClosedWorkOrder(t, tx, factoryModel, userID, FactoryWorkOrderResultCompleted, time.Now().AddDate(0, 0, -1))
		createMetricsExecution(t, tx, factoryModel, newRun(time.Now().AddDate(0, 0, -5)), order.ID, lineA.ID, 0, 0, time.Now().AddDate(0, 0, -5))
		// A restart: a second dispatch from step 0.
		createMetricsExecution(t, tx, factoryModel, newRun(time.Now().AddDate(0, 0, -4)), order.ID, lineA.ID, 0, 0, time.Now().AddDate(0, 0, -4))
		createMetricsExecution(t, tx, factoryModel, newRun(time.Now().AddDate(0, 0, -3)), order.ID, lineA.ID, 1, 0, time.Now().AddDate(0, 0, -3))

		createMetricsEvent(t, tx, order.ID, "order.comment.added", `{"author": {"kind": "user"}}`)
		createMetricsEvent(t, tx, order.ID, "order.comment.added", `{"author": {"kind": "automation"}}`) // not counted
		createMetricsEvent(t, tx, order.ID, "order.status.updated", `{"toState": "draft"}`)
		createMetricsEvent(t, tx, order.ID, "order.status.updated", `{"toState": "open"}`) // not counted

		rows, err := factoryModel.ListClosedWorkOrderMetricsRows(tx, since)
		require.NoError(t, err)
		row := findMetricsRow(t, rows, order.ID)
		// 1 user comment + 1 back-to-draft + 1 restart = 3
		assert.Equal(t, 3, row.ReworkCount)
	})

	t.Run("excludes work orders with no executions", func(t *testing.T) {
		order := createClosedWorkOrder(t, tx, factoryModel, userID, FactoryWorkOrderResultRejected, time.Now().AddDate(0, 0, -1))

		rows, err := factoryModel.ListClosedWorkOrderMetricsRows(tx, since)
		require.NoError(t, err)
		assert.False(t, hasMetricsRow(rows, order.ID))
	})

	t.Run("excludes work orders closed before the cutoff", func(t *testing.T) {
		order := createClosedWorkOrder(t, tx, factoryModel, userID, FactoryWorkOrderResultCompleted, time.Now().AddDate(0, 0, -90))
		createMetricsExecution(t, tx, factoryModel, newRun(time.Now().AddDate(0, 0, -90)), order.ID, lineA.ID, 0, 0, time.Now().AddDate(0, 0, -90))

		rows, err := factoryModel.ListClosedWorkOrderMetricsRows(tx, since)
		require.NoError(t, err)
		assert.False(t, hasMetricsRow(rows, order.ID))
	})

	t.Run("does not leak rows across lines in the same factory", func(t *testing.T) {
		orderA := createClosedWorkOrder(t, tx, factoryModel, userID, FactoryWorkOrderResultCompleted, time.Now().AddDate(0, 0, -1))
		createMetricsExecution(t, tx, factoryModel, newRun(time.Now().AddDate(0, 0, -2)), orderA.ID, lineA.ID, 0, 0, time.Now().AddDate(0, 0, -2))
		orderB := createClosedWorkOrder(t, tx, factoryModel, userID, FactoryWorkOrderResultCompleted, time.Now().AddDate(0, 0, -1))
		createMetricsExecution(t, tx, factoryModel, newRun(time.Now().AddDate(0, 0, -2)), orderB.ID, lineB.ID, 0, 0, time.Now().AddDate(0, 0, -2))

		rows, err := factoryModel.ListClosedWorkOrderMetricsRows(tx, since)
		require.NoError(t, err)
		assert.Equal(t, lineA.ID, findMetricsRow(t, rows, orderA.ID).LineID)
		assert.Equal(t, lineB.ID, findMetricsRow(t, rows, orderB.ID).LineID)
	})
}

// setupCanvasForMetricsTest creates the minimum canvas + live version needed
// to satisfy the `workflow_runs` foreign keys that back every execution's
// `run_id`; it returns (canvas id, version id) to reuse across runs.
func setupCanvasForMetricsTest(t *testing.T, tx *gorm.DB, organizationID, userID uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()

	now := time.Now()
	versionID := uuid.New()
	canvas := Canvas{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		LiveVersionID:  &versionID,
		Name:           "Line metrics canvas",
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	version := CanvasVersion{
		ID:         versionID,
		WorkflowID: canvas.ID,
		OwnerID:    &userID,
		Nodes:      datatypes.NewJSONSlice([]Node{}),
		Edges:      datatypes.NewJSONSlice([]Edge{}),
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}

	node := CanvasNode{
		WorkflowID:    canvas.ID,
		NodeID:        metricsRunNodeID,
		Name:          "Step",
		State:         CanvasNodeStateReady,
		Type:          NodeTypeComponent,
		Ref:           datatypes.NewJSONType(NodeRef{Component: &ComponentRef{Name: "noop"}}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(Position{}),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	// live_version_id is a deferred FK (canvas <-> version are mutually
	// referential); both inserts must land in the same transaction so the
	// constraint is only checked once both rows exist.
	require.NoError(t, tx.Transaction(func(itx *gorm.DB) error {
		if err := itx.Create(&canvas).Error; err != nil {
			return err
		}
		if err := itx.Create(&version).Error; err != nil {
			return err
		}
		return itx.Create(&node).Error
	}))

	return canvas.ID, versionID
}

// createMetricsRun creates a `workflow_runs` row so an execution's `run_id`
// foreign key has something to point at; each execution needs its own run
// (run_id is unique per execution).
func createMetricsRun(t *testing.T, tx *gorm.DB, canvasID, versionID uuid.UUID, createdAt time.Time) uuid.UUID {
	t.Helper()

	run := &CanvasRun{
		ID:         uuid.New(),
		WorkflowID: canvasID,
		NodeID:     metricsRunNodeID,
		VersionID:  versionID,
		State:      CanvasRunStateFinished,
		CreatedAt:  &createdAt,
		UpdatedAt:  &createdAt,
	}
	require.NoError(t, tx.Create(run).Error)
	return run.ID
}

func createClosedWorkOrder(t *testing.T, tx *gorm.DB, f *Factory, createdBy uuid.UUID, result string, closedAt time.Time) *FactoryWorkOrder {
	t.Helper()

	number, err := f.allocateNextWorkOrderNumber(tx)
	require.NoError(t, err)

	order := &FactoryWorkOrder{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		Number:         number,
		Title:          "Order",
		State:          FactoryWorkOrderStateClosed,
		Result:         result,
		CreatedByID:    &createdBy,
		CreatedAt:      closedAt,
		UpdatedAt:      closedAt,
	}
	require.NoError(t, tx.Create(order).Error)
	return order
}

func createMetricsExecution(t *testing.T, tx *gorm.DB, f *Factory, runID, workOrderID, lineID uuid.UUID, stepIndex int, costCents int64, createdAt time.Time) *FactoryWorkOrderExecution {
	t.Helper()

	execution := &FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		WorkOrderID:    workOrderID,
		LineID:         lineID,
		StepIndex:      stepIndex,
		StepName:       "step",
		RunID:          runID,
		Status:         FactoryWorkOrderExecutionStatusFinished,
		Result:         FactoryWorkOrderResultCompleted,
		CostCents:      costCents,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	require.NoError(t, tx.Create(execution).Error)
	return execution
}

func createMetricsEvent(t *testing.T, tx *gorm.DB, workOrderID uuid.UUID, eventType, data string) {
	t.Helper()

	event := &FactoryWorkOrderEvent{
		ID:          uuid.New(),
		WorkOrderID: workOrderID,
		Type:        eventType,
		Data:        datatypes.JSON(data),
		CreatedAt:   time.Now(),
	}
	require.NoError(t, tx.Create(event).Error)
}

func findMetricsRow(t *testing.T, rows []ClosedWorkOrderMetricsRow, workOrderID uuid.UUID) ClosedWorkOrderMetricsRow {
	t.Helper()
	for _, row := range rows {
		if row.WorkOrderID == workOrderID {
			return row
		}
	}
	t.Fatalf("no metrics row found for work order %s", workOrderID)
	return ClosedWorkOrderMetricsRow{}
}

func hasMetricsRow(rows []ClosedWorkOrderMetricsRow, workOrderID uuid.UUID) bool {
	for _, row := range rows {
		if row.WorkOrderID == workOrderID {
			return true
		}
	}
	return false
}
