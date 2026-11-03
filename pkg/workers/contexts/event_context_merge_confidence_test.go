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

func TestBitbucketMergeConfidenceDoesNotRestartOnDescriptionUpdates(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "trigger", Type: models.NodeTypeTrigger,
		Ref:      datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "bitbucket.onPullRequest"}}),
		Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1)),
	}}, nil)
	events := NewEventContext(db, &nodes[0], nil, nil)
	for _, revision := range []string{"abc", "abc", "next"} {
		require.NoError(t, events.Emit("bitbucket.pullRequest", map[string]any{
			"pullRequest":  map[string]any{"id": "tracked-pr"},
			"pull_request": map[string]any{"head": map[string]any{"sha": revision}},
			"pullrequest":  map[string]any{"description": "Updated merge confidence section"},
		}))
	}
	var count int64
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
	assert.EqualValues(t, 2, count, "publication must not start an assessment for the same head")
	// An explicit rerun bypasses webhook deduplication.
	require.NoError(t, events.Emit("factory.mergeConfidence", map[string]any{}))
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
	assert.EqualValues(t, 3, count)
}
