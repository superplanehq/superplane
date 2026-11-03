package workers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestRunFinishedTriggerPersistsIndependentDeduplicatedRuns(t *testing.T) {
	for _, result := range []string{models.CanvasRunResultPassed, models.CanvasRunResultFailed, models.CanvasRunResultCancelled} {
		t.Run(result, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
				{NodeID: "assessment", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
				{NodeID: "publication", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRunFinished"}}), Configuration: datatypes.NewJSONType(map[string]any{"sourceEntryNodeId": "assessment"})},
			}, nil)
			db := database.DB(t.Context())
			run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "assessment", models.CanvasRunStateStarted, "")
			require.NoError(t, err)
			if result == models.CanvasRunResultFailed {
				require.NoError(t, run.AddError(db, "Assessment failed", 4096))
			}
			if result == models.CanvasRunResultCancelled {
				require.NoError(t, db.Model(run).Update("state", models.CanvasRunStateCancelling).Error)
			}
			finalizer := NewRunFinalizer("", r.Registry)
			dispatch := func(tx *gorm.DB) error {
				_, _, err := finalizer.maybeFinalizeRun(tx, run.ID, runFinalizerTriggerSweep, nil, nil)
				return err
			}
			rollback := errors.New("rollback completion")
			require.ErrorIs(t, db.Transaction(func(tx *gorm.DB) error {
				require.NoError(t, dispatch(tx))
				return rollback
			}), rollback)
			var events []models.CanvasEvent
			require.NoError(t, db.Where("workflow_id = ? AND node_id = ?", canvas.ID, "publication").Find(&events).Error)
			require.Empty(t, events)
			for range 2 {
				require.NoError(t, db.Transaction(dispatch))
			}
			require.NoError(t, db.Where("workflow_id = ? AND node_id = ?", canvas.ID, "publication").Find(&events).Error)
			require.Len(t, events, 1)
			assert.NotEqual(t, run.ID, events[0].RunID)
			require.NoError(t, db.First(run, "id = ?", run.ID).Error)
			assert.Equal(t, result, run.Result)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return NewRunCallbackDispatcher(tx, r.Registry, run).DispatchFinished() }))
			var payload struct {
				Data struct {
					RunID       string `json:"runId"`
					Result      string `json:"result"`
					EntryNodeID string `json:"entryNodeId"`
				} `json:"data"`
			}
			raw, err := events[0].Data.MarshalJSON()
			require.NoError(t, err)
			require.NoError(t, models.UnmarshalJSONValue(raw, &payload))
			assert.Equal(t, run.ID.String(), payload.Data.RunID)
			assert.Equal(t, result, payload.Data.Result)
			assert.Equal(t, "assessment", payload.Data.EntryNodeID)
			publication, err := models.FindCanvasRunInTransaction(db, canvas.ID, events[0].RunID)
			require.NoError(t, err)
			publication.State = models.CanvasRunStateFinished
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return NewRunCallbackDispatcher(tx, r.Registry, publication).DispatchFinished()
			}))
			var count int64
			require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "publication").Count(&count).Error)
			assert.EqualValues(t, 1, count)
		})
	}
}
