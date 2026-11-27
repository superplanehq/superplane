package models_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestFactoryLineStagesMigrationBackfillsExistingLines(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry,
	}})
	require.NoError(t, err)
	migration, err := os.ReadFile("../../db/migrations/20261120161421_add-custom-line-stages.up.sql")
	require.NoError(t, err)

	tx := db.Begin()
	defer tx.Rollback()
	require.NoError(t, tx.Exec(`
		DELETE FROM factory_work_order_queue_items;
		ALTER TABLE factory_lines DROP COLUMN stages;
		ALTER TABLE factory_work_order_line_dispatches DROP COLUMN stages;
		DROP INDEX idx_factory_work_order_queue_items_stage;
		ALTER TABLE factory_work_order_queue_items
		  DROP CONSTRAINT factory_work_order_queue_items_line_dispatch_step_key,
		  DROP COLUMN app_id,
		  DROP COLUMN stage_key,
		  ADD CONSTRAINT factory_work_order_queue_items_line_dispatch_id_key UNIQUE (line_dispatch_id);
		CREATE INDEX idx_factory_work_order_queue_items_step
		  ON factory_work_order_queue_items (line_id, step_index, created_at);
	`).Error)
	require.NoError(t, tx.Exec(string(migration)).Error)
	var encoded string
	require.NoError(t, tx.Raw("SELECT stages FROM factory_lines WHERE id = ?", line.ID).Scan(&encoded).Error)
	var stages []models.FactoryLineStage
	require.NoError(t, json.Unmarshal([]byte(encoded), &stages))

	require.Len(t, stages, 4)
	assert.Equal(t, []string{"backlog", "phase-0", "verify", "done"}, []string{
		stages[0].Key,
		stages[1].Key,
		stages[2].Key,
		stages[3].Key,
	})
	assert.Contains(t, stages[1].Name, "Implement")
	assert.Equal(t, 0, *stages[1].StepIndex)
}

func TestFactoryStageKindsMigrationPreservesLineAndRetrySnapshots(t *testing.T) {
	r := support.Setup(t)
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	factory, err := models.CreateFactory(tx, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	line, err := factory.CreateLine(tx, "ship", nil)
	require.NoError(t, err)
	legacyStages := `[
		{"id":"backlog-id","key":"backlog","name":"Backlog","kind":"backlog"},
		{"id":"qa-id","key":"custom-qa","name":"QA","kind":"custom","auto_advance":false,"extra":"keep"},
		{"id":"verify-id","key":"verify","name":"Check","kind":"verify"},
		{"id":"review-id","key":"custom-review","name":"Review","kind":"verify","auto_advance":true},
		{"id":"done-id","key":"done","name":"Done","kind":"done"}
	]`
	expectedStages := `[
		{"id":"backlog-id","key":"backlog","name":"Backlog","kind":"backlog"},
		{"id":"qa-id","key":"custom-qa","name":"QA","kind":"phase","auto_advance":false,"extra":"keep"},
		{"id":"verify-id","key":"verify","name":"Check","kind":"phase","auto_advance":false},
		{"id":"review-id","key":"custom-review","name":"Review","kind":"phase","auto_advance":true},
		{"id":"done-id","key":"done","name":"Done","kind":"done"}
	]`
	legacySteps := `[{"stage_kind":"custom","stage_key":"custom-qa","extra":1},{"stage_kind":"verify","stage_key":"verify"},{"type":"runApp"}]`
	expectedSteps := `[{"stage_kind":"phase","stage_key":"custom-qa","extra":1},{"stage_kind":"phase","stage_key":"verify"},{"type":"runApp"}]`
	require.NoError(t, tx.Exec("UPDATE factory_lines SET stages = ?::jsonb, steps = ?::jsonb WHERE id = ?", legacyStages, legacySteps, line.ID).Error)
	ids := []uuid.UUID{}
	for _, state := range []string{models.FactoryWorkOrderLineDispatchStateActive, models.FactoryWorkOrderLineDispatchStateFinished} {
		order, err := factory.CreateWorkOrder(tx, state, "", &r.User, nil, nil)
		require.NoError(t, err)
		now := time.Now()
		dispatch := models.FactoryWorkOrderLineDispatch{
			ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID,
			WorkOrderID: order.ID, LineID: line.ID, LineName: line.Name, State: state,
			Steps: datatypes.JSONSlice[models.FactoryLineStep]{}, Stages: datatypes.JSONSlice[models.FactoryLineStage]{},
			CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, tx.Create(&dispatch).Error)
		require.NoError(t, tx.Exec("UPDATE factory_work_order_line_dispatches SET stages = ?::jsonb, steps = ?::jsonb WHERE id = ?", legacyStages, legacySteps, dispatch.ID).Error)
		ids = append(ids, dispatch.ID)
	}
	empty, err := factory.CreateLine(tx, "empty", nil)
	require.NoError(t, err)
	require.NoError(t, tx.Exec("UPDATE factory_lines SET stages = '[]'::jsonb, steps = '[]'::jsonb WHERE id = ?", empty.ID).Error)
	_, err = factory.CreateLine(tx, "null steps", nil)
	require.NoError(t, err)
	migration, err := os.ReadFile("../../db/migrations/20261124182440_unify-factory-stage-kinds.up.sql")
	require.NoError(t, err)
	for range 2 {
		require.NoError(t, tx.Exec(string(migration)).Error)
		var row struct{ Stages, Steps string }
		require.NoError(t, tx.Raw("SELECT stages, steps FROM factory_lines WHERE id = ?", line.ID).Scan(&row).Error)
		assert.JSONEq(t, expectedStages, row.Stages)
		assert.JSONEq(t, expectedSteps, row.Steps)
		for _, id := range ids {
			require.NoError(t, tx.Raw("SELECT stages, steps FROM factory_work_order_line_dispatches WHERE id = ?", id).Scan(&row).Error)
			assert.JSONEq(t, expectedStages, row.Stages)
			assert.JSONEq(t, expectedSteps, row.Steps)
		}
		require.NoError(t, tx.Raw("SELECT stages, steps FROM factory_lines WHERE id = ?", empty.ID).Scan(&row).Error)
		assert.JSONEq(t, `[]`, row.Stages)
		assert.JSONEq(t, `[]`, row.Steps)
	}
}

func TestWorkspaceStageIDsMigrationMaterializesEmptyLines(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	retained, err := factory.CreateLine(tx, "Retained", nil)
	require.NoError(t, err)
	first, err := factory.CreateLine(tx, "Empty", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry}})
	require.NoError(t, err)
	second, err := factory.CreateLine(tx, "Null steps", nil)
	require.NoError(t, err)
	require.NoError(t, tx.Exec("UPDATE factory_lines SET stages = '[]'::jsonb WHERE id IN ?", []uuid.UUID{first.ID, second.ID}).Error)
	require.NoError(t, tx.Exec("UPDATE factory_lines SET steps = 'null'::jsonb WHERE id = ?", second.ID).Error)
	migration, err := os.ReadFile("../../db/migrations/20261127173610_backfill-empty-workspace-stages.up.sql")
	require.NoError(t, err)
	require.NoError(t, tx.Exec(string(migration)).Error)
	publicIDs := map[string]bool{}
	for _, stage := range retained.Stages {
		publicIDs[stage.ConfigurationID] = true
	}
	snapshots := map[uuid.UUID][]models.FactoryLineStage{retained.ID: retained.StagesValue()}
	for _, id := range []uuid.UUID{first.ID, second.ID} {
		line, err := factory.FindLine(tx, id)
		require.NoError(t, err)
		stageCount := 3
		if id == first.ID {
			stageCount++
		}
		require.Len(t, line.Stages, stageCount)
		if id == first.ID {
			require.Equal(t, first.Steps, line.Steps)
			require.Equal(t, app.Name, line.Stages[1].Name)
			require.Equal(t, 0, *line.Stages[1].StepIndex)
		}
		for _, stage := range line.Stages {
			require.NotEqual(t, uuid.Nil, stage.ID)
			require.NotEmpty(t, stage.ConfigurationID)
			require.False(t, publicIDs[stage.ConfigurationID])
			publicIDs[stage.ConfigurationID] = true
		}
		require.Equal(t, models.FactoryLineStagePhase, line.Stages[len(line.Stages)-2].Kind)
		require.False(t, line.Stages[len(line.Stages)-2].AutoAdvanceEnabled())
		snapshots[id] = line.StagesValue()
	}
	require.NoError(t, tx.Exec(string(migration)).Error)
	for id, stages := range snapshots {
		line, err := factory.FindLine(tx, id)
		require.NoError(t, err)
		require.Equal(t, stages, line.StagesValue(), "repeated backfill must preserve stored identities")
	}
}
