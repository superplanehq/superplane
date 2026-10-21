package contexts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestBuildRefinementModel(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: models.UsageProviderAnthropic, Enabled: true, APIKey: []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{}) })
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "trigger", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "onWorkOrder"}}), Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateBacklogID, 3))},
		{NodeID: "refine-task", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}})},
		{NodeID: "start", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}})},
	}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", f.ID).Error)
	canvas.FactoryID = &f.ID
	order, err := f.CreateWorkOrder(db, "Refine this", "", &r.User, nil, nil)
	require.NoError(t, err)
	for _, model := range []string{"claude-opus-4-6", "", "gpt-5"} {
		t.Run(model, func(t *testing.T) {
			var override *string
			if model != "" {
				override = &model
			}
			require.NoError(t, db.Model(order).Update("refinement_model", override).Error)
			event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "trigger", "default", nil, map[string]any{
				"type": "workOrder.created", "data": map[string]any{"taskRefinementEnabled": true, "workOrder": map[string]any{"id": order.ID.String()}},
			})
			run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
			require.NoError(t, err)
			require.NoError(t, models.MaybeAttachAnalysisSession(db, canvas, event, run))
			linkRunToWorkOrder(t, r, f, order.ID, run.ID)
			execution, err := models.FindWorkOrderExecutionByRunID(db, run.ID)
			require.NoError(t, err)
			dispatch, err := models.FindWorkOrderLineDispatch(db, execution.LineDispatchID)
			require.NoError(t, err)
			require.NoError(t, db.Model(dispatch).Updates(map[string]any{"model": "claude-sonnet-4-6", "thinking_level": "high"}).Error)
			configuration := map[string]any{"model": "claude-sonnet-4-6", "thinkingLevel": "low", "credentials": map[string]any{"source": "hosted"}}
			resolved, err := NewNodeConfigurationBuilder(db, canvas.ID).WithNodeID("refine-task").WithRootEvent(&event.ID).Build(configuration)
			require.NoError(t, err)
			expected := "claude-sonnet-4-6"
			if model == "claude-opus-4-6" {
				expected = model
			}
			assert.Equal(t, expected, resolved["model"])
			assert.Equal(t, "low", resolved["thinkingLevel"])
			assert.Equal(t, "claude-sonnet-4-6", configuration["model"])
			started, err := NewNodeConfigurationBuilder(db, canvas.ID).WithNodeID("start").WithRootEvent(&event.ID).Build(configuration)
			require.NoError(t, err)
			assert.Equal(t, "claude-sonnet-4-6", started["model"])
			require.NoError(t, models.EndPlanningSessionForFinishedRun(db, run.ID, models.CanvasRunResultPassed))
		})
	}
}
