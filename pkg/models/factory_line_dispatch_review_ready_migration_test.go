package models_test

import (
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

func TestLineDispatchReviewReadyMigration(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Build", "start")
	version, err := models.FindLiveCanvasVersionInTransaction(db, app.ID)
	require.NoError(t, err)
	migration, err := os.ReadFile("../../db/migrations/20261123151840_add-line-dispatch-review-ready.up.sql")
	require.NoError(t, err)
	tx := db.Begin()
	defer tx.Rollback()
	require.NoError(t, tx.Exec("ALTER TABLE factory_work_order_line_dispatches DROP COLUMN review_ready").Error)

	cases := []struct {
		name, state, result string
		siblings            []string
		handoff, ready      bool
	}{
		{"parallel pass", "finished", "passed", []string{"passed", "passed"}, false, true},
		{"failed sibling", "finished", "passed", []string{"failed", "passed"}, false, false},
		{"missing sibling", "finished", "passed", []string{"passed"}, false, false},
		{"cancelled traversal", "finished", "cancelled", []string{"passed", "passed"}, false, false},
		{"failed traversal", "finished", "failed", []string{"passed", "failed"}, false, false},
		{"active traversal", "active", "", []string{"passed", "passed"}, false, false},
		{"direct handoff", "finished", "passed", []string{"passed"}, true, true},
		{"pending sibling", "finished", "passed", []string{"passed", "pending"}, false, false},
	}
	ids := make([]uuid.UUID, len(cases))
	for i, tc := range cases {
		order, err := factory.CreateWorkOrder(tx, tc.name, "", &r.User, nil, nil)
		require.NoError(t, err)
		ids[i] = uuid.New()
		dispatch := models.FactoryWorkOrderLineDispatch{
			ID: ids[i], OrganizationID: r.Organization.ID, FactoryID: factory.ID, WorkOrderID: order.ID,
			LineID: line.ID, LineName: line.Name, State: tc.state, Result: tc.result,
			Steps: datatypes.NewJSONSlice([]models.FactoryLineStep{{StageKey: "custom-qa"}, {StageKey: "custom-qa"}}),
		}
		require.NoError(t, tx.Omit("ReviewReady").Create(&dispatch).Error)
		for index, result := range tc.siblings {
			execution := models.FactoryWorkOrderExecution{
				ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID, WorkOrderID: order.ID,
				LineID: line.ID, LineDispatchID: dispatch.ID, StepIndex: index, Status: "finished", Result: result,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}
			if result == "pending" {
				execution.Status = models.FactoryWorkOrderExecutionStatusPending
				execution.Result = ""
			}
			if !tc.handoff {
				run := models.CanvasRun{ID: uuid.New(), WorkflowID: app.ID, NodeID: entry, VersionID: version.ID}
				require.NoError(t, tx.Create(&run).Error)
				execution.RunID = &run.ID
			}
			require.NoError(t, tx.Create(&execution).Error)
		}
	}
	require.NoError(t, tx.Exec(string(migration)).Error)
	for i, tc := range cases {
		dispatch, err := models.FindWorkOrderLineDispatch(tx, ids[i])
		require.NoError(t, err)
		assert.Equal(t, tc.ready, dispatch.ReviewReady, tc.name)
		assert.Equal(t, tc.state, dispatch.State, tc.name)
	}
	reloadedLine, err := factory.FindLine(tx, line.ID)
	require.NoError(t, err)
	assert.Equal(t, line.StagesValue(), reloadedLine.StagesValue())
}
