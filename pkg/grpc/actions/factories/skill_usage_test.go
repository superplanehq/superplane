package factories

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestListAutomationsMentioningSkill(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	for _, tc := range []struct {
		name, prompt string
		outside      bool
	}{
		{"Used twice", "Use /review-copy. Then (/review-copy) again.", false},
		{"Multiline", "First step\n/review-copy", false},
		{"Longer command", "/review-copy-extra", false},
		{"URL", "https://example.com/review-copy", false},
		{"Path", "/review-copy/file", false},
		{"Other workspace", "/review-copy", true},
	} {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
			NodeID: "agent", Name: "Agent", Type: models.NodeTypeComponent,
			Configuration: datatypes.NewJSONType(map[string]any{
				"steps":       []any{map[string]any{"type": "prompt", "prompt": tc.prompt}},
				"description": "/review-copy", "command": "echo /review-copy",
			}),
		}}, nil)
		require.NoError(t, db.Model(canvas).Updates(map[string]any{"factory_id": factory.ID, "name": tc.name}).Error)
		if tc.outside {
			require.NoError(t, db.Model(canvas).Update("factory_id", nil).Error)
		}
	}
	response, err := ListFactoryAutomations(t.Context(), r.Organization.ID.String(), &pb.ListFactoryAutomationsRequest{FactoryId: factory.ID.String(), SkillName: "review-copy"})
	require.NoError(t, err)
	require.Len(t, response.Automations, 2)
	require.Equal(t, "Multiline", response.Automations[0].Name)
	require.Equal(t, "Used twice", response.Automations[1].Name)
	// Ordinary listing still returns every automation, including non-matches.
	all, err := ListFactoryAutomations(t.Context(), r.Organization.ID.String(), &pb.ListFactoryAutomationsRequest{FactoryId: factory.ID.String()})
	require.NoError(t, err)
	require.Len(t, all.Automations, 5)
}
