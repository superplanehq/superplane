package public

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestFactoryPullRequestWebhookReusesLoadedRecords(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review changes", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider: "bitbucket", Repository: "acme/app", Number: 42,
		URL: "https://bitbucket.org/acme/app/pull-requests/42", State: models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)
	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "trigger", Type: models.NodeTypeTrigger,
		Ref:      datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "bitbucket.onPullRequest"}}),
		Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1)),
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	var published []models.CanvasEvent
	context := newLazyWebhookFactory(db, &nodes[0], func(events []models.CanvasEvent) { published = append(published, events...) })
	// Factory-only triggers load the canvas while matching the stored PR.
	_, err = context.FindPullRequest(core.FindPullRequestParams{Provider: "bitbucket", Repository: "acme/app", Number: 42})
	require.NoError(t, err)

	lookups := map[string]int{}
	callback := "test:factory-pr-reuse"
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "workflows" || tx.Statement.Table == "workflow_nodes" {
			lookups[tx.Statement.Table]++
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callback)) })
	producer := context.(core.FactoryPullRequestEventContext)
	for _, revision := range []string{"first", "first", "next"} {
		require.NoError(t, producer.EmitPullRequestEvent(nodes[0].NodeID, "bitbucket.pullRequest", map[string]any{
			"pullRequest":  map[string]any{"id": pullRequest.ID.String()},
			"repository":   map[string]any{"full_name": "acme/app"},
			"pull_request": map[string]any{"number": 42, "head": map[string]any{"sha": revision}, "base": map[string]any{"ref": "main"}},
		}))
	}
	assert.Zero(t, lookups["workflow_nodes"], "emission must reuse the loaded trigger")
	// Creating a run still reads the live canvas under the model admission guard.
	assert.LessOrEqual(t, lookups["workflows"], len(published), "emission must not reload the cached canvas")
	require.Len(t, published, 2, "only new revisions publish events")
	support.VerifyCanvasNodeEventsCount(t, canvas.ID, nodes[0].NodeID, 2)
}
