package contexts

import (
	"testing"
	"time"

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
	updatedAt := time.Now().Add(-time.Hour)
	emitAt := func(revision, base string, at time.Time) int64 {
		t.Helper()
		require.NoError(t, events.Emit("bitbucket.pullRequest", map[string]any{
			"pullRequest": map[string]any{"id": "tracked-pr"},
			"pull_request": map[string]any{
				"head": map[string]any{"sha": revision},
				"base": map[string]any{"ref": base},
			},
			"pullrequest": map[string]any{
				"description": "Updated merge confidence section",
				"updated_on":  at.UTC().Format(time.RFC3339Nano),
			},
		}))
		var count int64
		require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
		return count
	}
	emit := func(revision, base string) int64 {
		t.Helper()
		updatedAt = updatedAt.Add(time.Minute)
		return emitAt(revision, base, updatedAt)
	}
	assert.EqualValues(t, 1, emit("abc", "main"))
	assert.EqualValues(t, 1, emit("abc", "main"), "publication must not start an assessment for the same diff")
	assert.EqualValues(t, 2, emit("next", "main"), "a new head starts an assessment")
	assert.EqualValues(t, 3, emit("next", "develop"), "a retargeted PR starts an assessment")
	assert.EqualValues(t, 3, emit("next", "develop"))
	assert.EqualValues(t, 4, emit("next", "main"), "a return to an earlier target starts an assessment")
	assert.EqualValues(t, 4, emitAt("abc", "main", updatedAt.Add(-time.Minute)), "a delayed webhook for an earlier state must not start an assessment")
	assert.EqualValues(t, 5, emit("abc", "main"), "a later update to an earlier head starts an assessment")
	// An explicit rerun bypasses webhook deduplication.
	require.NoError(t, events.Emit("factory.mergeConfidence", map[string]any{}))
	var count int64
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
	assert.EqualValues(t, 6, count)
}

// Starts from the PR checks start path and starts stored before dedup keys
// existed must count as the newest start for the same PR.
func TestBitbucketMergeConfidenceSeesStartsWithOtherKeys(t *testing.T) {
	legacyKey := "acme/app|42|abc"
	for name, key := range map[string]*string{"checks start key": &legacyKey, "no key": nil} {
		t.Run(name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
				NodeID: "trigger", Type: models.NodeTypeTrigger,
				Ref:      datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "bitbucket.onPullRequest"}}),
				Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1)),
			}}, nil)
			updatedAt := time.Now().Add(-time.Hour)
			payload := func(revision string, at time.Time) map[string]any {
				return map[string]any{
					"pullRequest": map[string]any{"id": "tracked-pr"},
					"repository":  map[string]any{"full_name": "Acme/App"},
					"pull_request": map[string]any{
						"number": 42,
						"head":   map[string]any{"sha": revision},
						"base":   map[string]any{"ref": "main"},
					},
					"pullrequest": map[string]any{"updated_on": at.UTC().Format(time.RFC3339Nano)},
				}
			}
			createdAt := time.Now().Add(-time.Minute)
			require.NoError(t, db.Create(&models.CanvasEvent{
				WorkflowID: canvas.ID, NodeID: "trigger", Channel: "default", State: models.CanvasEventStateRouted,
				Data:     models.NewJSONValue(map[string]any{"type": "bitbucket.pullRequest", "data": payload("abc", updatedAt)}),
				DedupKey: key, CreatedAt: &createdAt,
			}).Error)
			events := NewEventContext(db, &nodes[0], nil, nil)
			count := func() int64 {
				var count int64
				require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
				return count
			}

			require.NoError(t, events.Emit("bitbucket.pullRequest", payload("abc", updatedAt.Add(time.Minute))))
			assert.EqualValues(t, 1, count(), "the same diff must not start a second assessment")
			require.NoError(t, events.Emit("bitbucket.pullRequest", payload("next", updatedAt.Add(2*time.Minute))))
			assert.EqualValues(t, 2, count(), "a new head starts an assessment")
		})
	}
}
