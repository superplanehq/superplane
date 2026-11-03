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
	emit := func(revision, base string) int64 {
		t.Helper()
		require.NoError(t, events.Emit("bitbucket.pullRequest", map[string]any{
			"pullRequest": map[string]any{"id": "tracked-pr"},
			"pull_request": map[string]any{
				"head": map[string]any{"sha": revision},
				"base": map[string]any{"ref": base},
			},
			"pullrequest": map[string]any{"description": "Updated merge confidence section"},
		}))
		var count int64
		require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
		return count
	}
	assert.EqualValues(t, 1, emit("abc", "main"))
	assert.EqualValues(t, 1, emit("abc", "main"), "publication must not start an assessment for the same diff")
	assert.EqualValues(t, 2, emit("next", "main"), "a new head starts an assessment")
	assert.EqualValues(t, 3, emit("next", "develop"), "a retargeted PR starts an assessment")
	assert.EqualValues(t, 3, emit("next", "develop"))
	assert.EqualValues(t, 4, emit("next", "main"), "a return to an earlier target starts an assessment")
	// An explicit rerun bypasses webhook deduplication.
	require.NoError(t, events.Emit("factory.mergeConfidence", map[string]any{}))
	var count int64
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
	assert.EqualValues(t, 5, count)
}
