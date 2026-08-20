package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__ListClosedWorkOrderMetricRows(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	now := time.Now()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	lineA, err := factoryModel.CreateLine(db, "line-a", nil)
	require.NoError(t, err)
	lineB, err := factoryModel.CreateLine(db, "line-b", nil)
	require.NoError(t, err)

	windowFrom := now.Add(-30 * 24 * time.Hour)
	windowTo := now.Add(time.Hour)

	t.Run("attributes the work order to the latest execution line", func(t *testing.T) {
		order := createClosedMetricOrder(t, db, factoryModel, r)
		older := now.Add(-2 * time.Hour)
		newer := now.Add(-time.Hour)
		createMetricExecution(t, db, r, factoryModel.ID, order.ID, lineA.ID, lineA.Name, older, 0)
		createMetricExecution(t, db, r, factoryModel.ID, order.ID, lineB.ID, lineB.Name, newer, 0)
		closeWorkOrderAt(t, db, order, now.Add(-30*time.Minute))

		rows := listMetricRows(t, db, factoryModel.ID, windowFrom, windowTo)
		row := requireMetricRow(t, rows, order.ID)
		assert.Equal(t, lineB.ID, row.LineID)
	})

	t.Run("uses the close event time not updated_at", func(t *testing.T) {
		order := createClosedMetricOrder(t, db, factoryModel, r)
		createMetricExecution(t, db, r, factoryModel.ID, order.ID, lineA.ID, lineA.Name, now.Add(-time.Hour), 0)
		closeWorkOrderAt(t, db, order, now.Add(-40*24*time.Hour))
		require.NoError(t, db.Model(order).Update("updated_at", now).Error)

		rows := listMetricRows(t, db, factoryModel.ID, windowFrom, windowTo)
		assert.Nil(t, findMetricRow(rows, order.ID), "close 40 days ago must fall outside the window")
	})

	t.Run("includes a close event inside the window", func(t *testing.T) {
		order := createClosedMetricOrder(t, db, factoryModel, r)
		createMetricExecution(t, db, r, factoryModel.ID, order.ID, lineA.ID, lineA.Name, now.Add(-time.Hour), 0)
		closedAt := now.Add(-5 * 24 * time.Hour)
		closeWorkOrderAt(t, db, order, closedAt)

		rows := listMetricRows(t, db, factoryModel.ID, windowFrom, windowTo)
		row := requireMetricRow(t, rows, order.ID)
		assert.WithinDuration(t, closedAt, row.ClosedAt, time.Second)
	})

	t.Run("completed close is success and merge time falls back to close", func(t *testing.T) {
		order := createClosedMetricOrder(t, db, factoryModel, r)
		createMetricExecution(t, db, r, factoryModel.ID, order.ID, lineA.ID, lineA.Name, now.Add(-time.Hour), 0)
		closedAt := now.Add(-time.Hour)
		closeWorkOrderAt(t, db, order, closedAt)

		rows := listMetricRows(t, db, factoryModel.ID, windowFrom, windowTo)
		row := requireMetricRow(t, rows, order.ID)
		assert.True(t, row.Merged)
		require.NotNil(t, row.MergedAt)
		assert.WithinDuration(t, closedAt, *row.MergedAt, time.Second)
	})

	t.Run("rejected close with an unmerged PR is not success", func(t *testing.T) {
		order := createClosedMetricOrder(t, db, factoryModel, r)
		createMetricExecution(t, db, r, factoryModel.ID, order.ID, lineA.ID, lineA.Name, now.Add(-time.Hour), 0)
		attachOpenPR(t, db, order, "https://github.com/example/repo/pull/metrics-rejected")
		closeWorkOrderAtWithResult(t, db, order, now.Add(-time.Hour), models.FactoryWorkOrderResultRejected)

		rows := listMetricRows(t, db, factoryModel.ID, windowFrom, windowTo)
		row := requireMetricRow(t, rows, order.ID)
		assert.False(t, row.Merged)
		assert.Nil(t, row.MergedAt)
	})

	t.Run("GitHub-shaped closed merged PR counts as success without merged_at", func(t *testing.T) {
		order := createClosedMetricOrder(t, db, factoryModel, r)
		createMetricExecution(t, db, r, factoryModel.ID, order.ID, lineA.ID, lineA.Name, now.Add(-time.Hour), 0)
		attachGitHubMergedPR(t, db, order, "https://github.com/example/repo/pull/metrics-github")
		closedAt := now.Add(-45 * time.Minute)
		closeWorkOrderAtWithResult(t, db, order, closedAt, models.FactoryWorkOrderResultFailed)

		rows := listMetricRows(t, db, factoryModel.ID, windowFrom, windowTo)
		row := requireMetricRow(t, rows, order.ID)
		assert.True(t, row.Merged)
		require.NotNil(t, row.MergedAt)
		assert.WithinDuration(t, closedAt, *row.MergedAt, time.Second)
	})

	t.Run("merged PR artifact counts as success", func(t *testing.T) {
		order := createClosedMetricOrder(t, db, factoryModel, r)
		firstExec := now.Add(-3 * time.Hour)
		createMetricExecution(t, db, r, factoryModel.ID, order.ID, lineA.ID, lineA.Name, firstExec, 25)
		mergedAt := now.Add(-90 * time.Minute)
		attachMergedPR(t, db, order, "https://github.com/example/repo/pull/metrics-1", mergedAt)
		closeWorkOrderAt(t, db, order, now.Add(-time.Hour))

		rows := listMetricRows(t, db, factoryModel.ID, windowFrom, windowTo)
		row := requireMetricRow(t, rows, order.ID)
		assert.True(t, row.Merged)
		require.NotNil(t, row.MergedAt)
		assert.WithinDuration(t, mergedAt, *row.MergedAt, time.Second)
		require.NotNil(t, row.FirstExecutionAt)
		assert.WithinDuration(t, firstExec, *row.FirstExecutionAt, time.Second)
		assert.Equal(t, int64(25), row.CostCents)
	})

	t.Run("excludes work orders with no executions", func(t *testing.T) {
		order := createClosedMetricOrder(t, db, factoryModel, r)
		closeWorkOrderAt(t, db, order, now.Add(-time.Hour))

		rows := listMetricRows(t, db, factoryModel.ID, windowFrom, windowTo)
		assert.Nil(t, findMetricRow(rows, order.ID))
	})

	t.Run("sums execution cost across retries", func(t *testing.T) {
		order := createClosedMetricOrder(t, db, factoryModel, r)
		createMetricExecution(t, db, r, factoryModel.ID, order.ID, lineA.ID, lineA.Name, now.Add(-2*time.Hour), 100)
		createMetricExecution(t, db, r, factoryModel.ID, order.ID, lineA.ID, lineA.Name, now.Add(-time.Hour), 50)
		closeWorkOrderAt(t, db, order, now.Add(-30*time.Minute))

		rows := listMetricRows(t, db, factoryModel.ID, windowFrom, windowTo)
		row := requireMetricRow(t, rows, order.ID)
		assert.Equal(t, int64(150), row.CostCents)
	})
}

func listMetricRows(t *testing.T, db *gorm.DB, factoryID uuid.UUID, from, to time.Time) []models.ClosedWorkOrderMetricRow {
	t.Helper()
	rows, err := models.ListClosedWorkOrderMetricRows(db, factoryID, from, to)
	require.NoError(t, err)
	return rows
}

func requireMetricRow(t *testing.T, rows []models.ClosedWorkOrderMetricRow, orderID uuid.UUID) models.ClosedWorkOrderMetricRow {
	t.Helper()
	row := findMetricRow(rows, orderID)
	require.NotNil(t, row, "expected metric row for work order %s", orderID)
	return *row
}

func findMetricRow(rows []models.ClosedWorkOrderMetricRow, orderID uuid.UUID) *models.ClosedWorkOrderMetricRow {
	for i := range rows {
		if rows[i].WorkOrderID == orderID {
			return &rows[i]
		}
	}
	return nil
}

func createClosedMetricOrder(t *testing.T, db *gorm.DB, factoryModel *models.Factory, r *support.ResourceRegistry) *models.FactoryWorkOrder {
	t.Helper()
	order, err := factoryModel.CreateWorkOrder(db, "Metric order", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	return order
}

func closeWorkOrderAt(t *testing.T, db *gorm.DB, order *models.FactoryWorkOrder, closedAt time.Time) {
	t.Helper()
	closeWorkOrderAtWithResult(t, db, order, closedAt, models.FactoryWorkOrderResultCompleted)
}

func closeWorkOrderAtWithResult(t *testing.T, db *gorm.DB, order *models.FactoryWorkOrder, closedAt time.Time, result string) {
	t.Helper()
	_, err := order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  result,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.FactoryWorkOrderEvent{}).
		Where("work_order_id = ? AND type = ? AND data->>'toState' = ?", order.ID, factory.EventTypeOrderStatusUpdated, models.FactoryWorkOrderStateClosed).
		Update("created_at", closedAt).Error)
}

func createMetricRun(t *testing.T, r *support.ResourceRegistry) uuid.UUID {
	t.Helper()
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		nil,
	)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	return createRunForRootEvent(t, rootEvent).ID
}

func createMetricExecution(
	t *testing.T,
	db *gorm.DB,
	r *support.ResourceRegistry,
	factoryID, workOrderID, lineID uuid.UUID,
	lineName string,
	createdAt time.Time,
	costCents int64,
) {
	t.Helper()
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryID, workOrderID, lineID, lineName, nil)
	runID := createMetricRun(t, r)
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
		CostCents:      costCents,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	require.NoError(t, db.Create(&execution).Error)
}

func attachMergedPR(t *testing.T, db *gorm.DB, order *models.FactoryWorkOrder, url string, mergedAt time.Time) {
	t.Helper()
	_, err := order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypePR,
		Key:  url,
		Data: map[string]any{
			"url":      url,
			"state":    models.PrArtifactStateMerged,
			"mergedAt": mergedAt.UTC().Format(time.RFC3339),
		},
	})
	require.NoError(t, err)
}

func attachOpenPR(t *testing.T, db *gorm.DB, order *models.FactoryWorkOrder, url string) {
	t.Helper()
	_, err := order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypePR,
		Key:  url,
		Data: map[string]any{
			"url":   url,
			"state": models.PrArtifactStateOpen,
		},
	})
	require.NoError(t, err)
}

func attachGitHubMergedPR(t *testing.T, db *gorm.DB, order *models.FactoryWorkOrder, url string) {
	t.Helper()
	artifact, err := order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypePR,
		Key:  url,
		Data: map[string]any{
			"url":    url,
			"state":  models.PrArtifactStateClosed,
			"merged": true,
		},
	})
	require.NoError(t, err)
	require.Nil(t, artifact.MergedAt, "GitHub-shaped payload must not stamp merged_at")
}
