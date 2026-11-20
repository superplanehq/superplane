package models_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
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
	migration, err := os.ReadFile("../../db/migrations/20261120150841_add-custom-line-stages.up.sql")
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
