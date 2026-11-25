package contexts

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestBuildCustomChatContinuationSkipsCompletedSteps(t *testing.T) {
	for _, taskKey := range []string{"workOrder", "work_order"} {
		t.Run(taskKey, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			steps := []any{
				map[string]any{"name": "Prepare", "type": "bash", "command": "deploy --prepare"},
				map[string]any{"name": "Inspect", "type": "prompt", "prompt": "Inspect the deployment"},
				map[string]any{"name": "Discuss", "type": "prompt", "prompt": "Discuss deployment health", "continueInChat": true},
			}
			configuration := map[string]any{"steps": steps, "workingDirectory": "repo", "executionTimeoutSeconds": 900}
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
				{NodeID: "on-task", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "onWorkOrder"}}), Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateCustomAutomationID, 1))},
				{NodeID: "verify", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}}), Configuration: datatypes.NewJSONType(configuration)},
			}, nil)
			require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
			canvas.FactoryID = &factory.ID
			order, err := factory.CreateWorkOrder(db, "Inspect deployment", "", &r.User, []uuid.UUID{r.User}, nil)
			require.NoError(t, err)
			event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "on-task", "default", nil, map[string]any{
				"type": "workOrder.created", "data": map[string]any{taskKey: map[string]any{"id": order.ID.String()}},
			})
			run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
			require.NoError(t, err)
			require.NoError(t, models.MaybeAttachCustomAutomationSession(db, canvas, event, run))
			session, err := models.FindPlanningSessionByRun(db, run.ID)
			require.NoError(t, err)
			initial, err := NewNodeConfigurationBuilder(db, canvas.ID).WithRunID(run.ID).WithNodeID("verify").WithRootEvent(&event.ID).Build(configuration)
			require.NoError(t, err)
			assert.Equal(t, configuration, initial)

			require.NoError(t, session.End(db))
			restarted, err := models.RestartCustomAutomationSession(db, session)
			require.NoError(t, err)
			continued, err := NewNodeConfigurationBuilder(db, canvas.ID).WithRunID(restarted.RunID).WithNodeID("verify").WithRootEvent(&restarted.ID).Build(configuration)
			require.NoError(t, err)
			assert.Equal(t, map[string]any{
				"steps": []any{steps[2]}, "workingDirectory": "repo", "executionTimeoutSeconds": 900,
			}, continued)
			assert.Len(t, configuration["steps"], 3, "dispatch must not mutate the stored automation")
		})
	}
}
