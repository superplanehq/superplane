package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestListFactoryLineMetrics(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "line-metrics")
	db := database.Conn()
	now := time.Date(2026, 8, 17, 15, 0, 0, 0, time.UTC)

	ship, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	hotfix, err := factoryModel.CreateLine(db, "hotfix", nil)
	require.NoError(t, err)

	t.Run("returns empty metrics when no work orders closed", func(t *testing.T) {
		got, err := ListFactoryLineMetrics(db, factoryModel.OrganizationID, factoryModel.ID, now)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, hotfix.ID, got[0].LineID)
		assert.False(t, got[0].Present)
		assert.Equal(t, ship.ID, got[1].LineID)
		assert.False(t, got[1].Present)
	})

	runID := insertMetricsCanvasRun(t, factoryModel.OrganizationID, userID)

	_ = insertClosedWorkOrder(t, metricsOrderSetup{
		factory:   factoryModel,
		userID:    userID,
		lineID:    ship.ID,
		runID:     runID,
		closedAt:  now.Add(-2 * 24 * time.Hour),
		merged:    true,
		costCents: 320,
		extraRuns: 1,
		userNotes: 1,
		autoNotes: 4,
		stepIndex: 0,
	})
	_ = insertClosedWorkOrder(t, metricsOrderSetup{
		factory:   factoryModel,
		userID:    userID,
		lineID:    ship.ID,
		runID:     insertMetricsCanvasRun(t, factoryModel.OrganizationID, userID),
		closedAt:  now.Add(-3 * 24 * time.Hour),
		merged:    false,
		costCents: 80,
	})
	_ = insertClosedWorkOrder(t, metricsOrderSetup{
		factory:   factoryModel,
		userID:    userID,
		lineID:    hotfix.ID,
		runID:     insertMetricsCanvasRun(t, factoryModel.OrganizationID, userID),
		closedAt:  now.Add(-40 * 24 * time.Hour),
		merged:    true,
		costCents: 1100,
	})

	got, err := ListFactoryLineMetrics(db, factoryModel.OrganizationID, factoryModel.ID, now)
	require.NoError(t, err)
	require.Len(t, got, 2)

	hotfixMetrics := got[0]
	assert.Equal(t, hotfix.ID, hotfixMetrics.LineID)
	assert.False(t, hotfixMetrics.Present, "prior-window closes must not fill the current window")

	shipMetrics := got[1]
	assert.Equal(t, ship.ID, shipMetrics.LineID)
	require.True(t, shipMetrics.Present)
	assert.Equal(t, 50.0, shipMetrics.SuccessRatePct)
	assert.Equal(t, 1, shipMetrics.MergedCount)
	assert.Equal(t, 2, shipMetrics.TotalClosedCount)
	assert.InDelta(t, 1.0, shipMetrics.ReworkPerWorkOrder, 0.001)
	assert.InDelta(t, 4.00, shipMetrics.CostPerSuccessUsd, 0.001)
	assert.InDelta(t, 1.0/30.0, shipMetrics.ThroughputPerDay, 0.0001)
	assert.Equal(t, 50.0, shipMetrics.SuccessDeltaPts)
	assert.Len(t, shipMetrics.SuccessTrendPct, FactoryLineMetricsTrendDays)
	assert.Len(t, shipMetrics.ThroughputTrend, FactoryLineMetricsTrendDays)
	assert.Equal(t, 1, shipMetrics.ThroughputTrend[FactoryLineMetricsTrendDays-3])
}

func TestListFactoryLineMetrics_UsesLatestExecutionLine(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "line-metrics-latest")
	db := database.Conn()
	now := time.Date(2026, 8, 17, 15, 0, 0, 0, time.UTC)

	ship, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	hotfix, err := factoryModel.CreateLine(db, "hotfix", nil)
	require.NoError(t, err)

	order := insertClosedWorkOrder(t, metricsOrderSetup{
		factory:  factoryModel,
		userID:   userID,
		lineID:   ship.ID,
		runID:    insertMetricsCanvasRun(t, factoryModel.OrganizationID, userID),
		closedAt: now.Add(-time.Hour),
		merged:   true,
	})

	later := now.Add(-30 * time.Minute)
	require.NoError(t, db.Create(&FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: factoryModel.OrganizationID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         hotfix.ID,
		StepIndex:      0,
		StepName:       "verify",
		RunID:          insertMetricsCanvasRun(t, factoryModel.OrganizationID, userID),
		Status:         FactoryWorkOrderExecutionStatusFinished,
		CreatedAt:      later,
		UpdatedAt:      later,
	}).Error)

	got, err := ListFactoryLineMetrics(db, factoryModel.OrganizationID, factoryModel.ID, now)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.False(t, got[1].Present)
	require.True(t, got[0].Present)
	assert.Equal(t, hotfix.ID, got[0].LineID)
	assert.Equal(t, 1, got[0].MergedCount)
}

type metricsOrderSetup struct {
	factory   *Factory
	userID    uuid.UUID
	lineID    uuid.UUID
	runID     uuid.UUID
	closedAt  time.Time
	merged    bool
	costCents int64
	extraRuns int
	userNotes int
	autoNotes int
	stepIndex int
}

func insertClosedWorkOrder(t *testing.T, setup metricsOrderSetup) *FactoryWorkOrder {
	t.Helper()
	db := database.Conn()

	order, err := setup.factory.CreateWorkOrder(db, "Metrics order", "", &setup.userID, nil, nil)
	require.NoError(t, err)

	_, err = order.UpdateStatus(db, FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateOpen,
		Actor:   &setup.userID,
	})
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateClosed,
		Result:  FactoryWorkOrderResultCompleted,
		Actor:   &setup.userID,
	})
	require.NoError(t, err)

	created := setup.closedAt.Add(-time.Hour)
	require.NoError(t, db.Create(&FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: setup.factory.OrganizationID,
		FactoryID:      setup.factory.ID,
		WorkOrderID:    order.ID,
		LineID:         setup.lineID,
		StepIndex:      setup.stepIndex,
		StepName:       "implement",
		RunID:          setup.runID,
		Status:         FactoryWorkOrderExecutionStatusFinished,
		CostCents:      setup.costCents,
		CreatedAt:      created,
		UpdatedAt:      created,
	}).Error)

	for i := 0; i < setup.extraRuns; i++ {
		require.NoError(t, db.Create(&FactoryWorkOrderExecution{
			ID:             uuid.New(),
			OrganizationID: setup.factory.OrganizationID,
			FactoryID:      setup.factory.ID,
			WorkOrderID:    order.ID,
			LineID:         setup.lineID,
			StepIndex:      setup.stepIndex,
			StepName:       "implement",
			RunID:          insertMetricsCanvasRun(t, setup.factory.OrganizationID, setup.userID),
			Status:         FactoryWorkOrderExecutionStatusFinished,
			CreatedAt:      created.Add(time.Duration(i+1) * time.Minute),
			UpdatedAt:      created.Add(time.Duration(i+1) * time.Minute),
		}).Error)
	}

	if setup.merged {
		_, err = order.CreateArtifact(db, FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypePR,
			Data: map[string]any{
				"url":   fmt.Sprintf("https://github.com/example/repo/pull/%s", order.ID),
				"state": PrArtifactStateMerged,
			},
			CreatedBy: &setup.userID,
		})
		require.NoError(t, err)
	}

	for i := 0; i < setup.userNotes; i++ {
		require.NoError(t, order.RecordCommentAdded(db, "steer this", factory.WorkOrderCommentAuthor{
			Kind:   factory.CommentAuthorKindUser,
			UserID: ptrTo(setup.userID.String()),
		}, nil))
	}
	for i := 0; i < setup.autoNotes; i++ {
		require.NoError(t, order.RecordCommentAdded(db, "bot note", factory.WorkOrderCommentAuthor{
			Kind: factory.CommentAuthorKindAutomation,
		}, nil))
	}

	require.NoError(t, db.Model(&FactoryWorkOrderEvent{}).
		Where("work_order_id = ? AND type = ? AND data->>'toState' = ?", order.ID, factory.EventTypeOrderStatusUpdated, FactoryWorkOrderStateClosed).
		Update("created_at", setup.closedAt).Error)

	return order
}

func insertMetricsCanvasRun(t *testing.T, orgID, userID uuid.UUID) uuid.UUID {
	t.Helper()
	now := time.Now()
	versionID := uuid.New()
	canvas := Canvas{
		ID:             uuid.New(),
		OrganizationID: orgID,
		LiveVersionID:  &versionID,
		Name:           fmt.Sprintf("metrics-canvas-%s", uuid.New()),
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
		NodeID:        "trigger",
		Name:          "Trigger",
		State:         CanvasNodeStateReady,
		Type:          NodeTypeTrigger,
		Ref:           datatypes.NewJSONType(NodeRef{}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(Position{}),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	run := CanvasRun{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     node.NodeID,
		VersionID:  versionID,
		Input:      NewJSONValue(map[string]any{}),
		State:      CanvasRunStateFinished,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&canvas).Error; err != nil {
			return err
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		return tx.Create(&run).Error
	}))
	return run.ID
}

func ptrTo[T any](value T) *T {
	return &value
}

func TestUtcDay(t *testing.T) {
	got := utcDay(time.Date(2026, 8, 17, 23, 59, 0, 0, time.FixedZone("x", -2*3600)))
	assert.Equal(t, time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC), got)
}
