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

func TestManualCustomAutomationIgnoresPullRequestWebhooks(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "on-pull-request",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "github.onPullRequest"},
			}),
			Metadata: datatypes.NewJSONType(map[string]any{
				models.FactoryAppTemplateMetadataKey: map[string]any{
					"id":        models.FactoryAppTemplateCustomAutomationID,
					"startMode": "manual",
				},
			}),
		},
	}, nil)
	node := nodes[0]
	events := NewEventContext(db, &node, nil, nil)

	require.NoError(t, events.Emit("github.pullRequest", map[string]any{
		"pullRequest": map[string]any{"id": "tracked-pr"},
		"workOrder":   map[string]any{"id": r.User.String()},
		"pull_request": map[string]any{
			"head": map[string]any{"sha": "abc123"},
		},
	}))
	require.NoError(t, events.Emit("factory.customAutomation", map[string]any{"manual": true}))
	require.NoError(t, events.Emit("factory.customAutomation", map[string]any{"manual": true}))

	var count int64
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
	assert.EqualValues(t, 2, count)
}
