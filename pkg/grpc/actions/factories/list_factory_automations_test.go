package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestListFactoryAutomationsReportsBacklogIdentityAfterRename(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	renamed, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "trigger",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "onWorkOrder"},
			}),
			Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateBacklogID, 3)),
		},
	}, nil)
	decoy, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "trigger",
			Type:   models.NodeTypeTrigger,
			Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onWorkOrder"}}),
		},
	}, nil)
	legacy, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "trigger", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onWorkOrder"}})},
		{NodeID: "task-refinement-enabled", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "if"}})},
		{NodeID: "refine-task", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}})},
		{NodeID: "add-run-error", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "addRunError"}})},
	}, nil)
	require.NoError(t, db.Model(renamed).Updates(map[string]any{"factory_id": factory.ID, "name": "Planning"}).Error)
	require.NoError(t, db.Model(decoy).Updates(map[string]any{"factory_id": factory.ID, "name": "Backlog"}).Error)
	require.NoError(t, db.Model(legacy).Updates(map[string]any{"factory_id": factory.ID, "name": "Old planner"}).Error)

	response, err := ListFactoryAutomations(t.Context(), r.Organization.ID.String(), &pb.ListFactoryAutomationsRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)

	got := map[string]string{}
	for _, automation := range response.Automations {
		got[automation.GetId()] = automation.GetTemplateId()
	}
	assert.Equal(t, models.FactoryAppTemplateBacklogID, got[renamed.ID.String()])
	assert.Equal(t, models.FactoryAppTemplateBacklogID, got[legacy.ID.String()])
	assert.Empty(t, got[decoy.ID.String()])
}
