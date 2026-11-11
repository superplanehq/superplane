package factories

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
	githubmocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

// Exercise stored event decoding at the component boundary. GitHub identities
// and explicit On Run parameters must resolve the same workspace pull request.
func TestPrepareHumanReviewComponentResolvesEventIdentity(t *testing.T) {
	for _, scenario := range []string{"GitHub event", "On Run parameters"} {
		t.Run(scenario, func(t *testing.T) {
			f := newHumanReviewReconciliationFixture(t)
			version, err := models.FindLiveCanvasVersionByCanvasInTransaction(f.db, f.automation.canvas)
			require.NoError(t, err)
			for i := range version.Nodes {
				if version.Nodes[i].ID == models.HumanReviewPrepareNodeID {
					version.Nodes[i].Configuration["enabled"] = false
				}
			}
			require.NoError(t, f.db.Model(version).Update("nodes", version.Nodes).Error)
			eventType := "github.pullRequest"
			data := map[string]any{"repository": map[string]any{"full_name": "acme/app"}, "pull_request": map[string]any{"number": 42}}
			if scenario == "On Run parameters" {
				eventType = "app.invocation"
				data = map[string]any{"parameters": map[string]any{"repository": "acme/app", "number": 42}}
			}
			root := support.EmitCanvasEventForNodeWithData(t, f.automation.canvas.ID, models.HumanReviewTriggerNodeID, "default", nil, map[string]any{"type": eventType, "data": data})
			execution := support.CreateCanvasNodeExecution(t, f.automation.canvas.ID, models.HumanReviewPrepareNodeID, root.ID, root.ID)
			state := &supportcontexts.ExecutionStateContext{}
			component := &prepareHumanReviewComponent{}
			_, err = component.HandleBackgroundHook(core.BackgroundHookContext{
				WorkflowID: f.automation.canvas.ID, OrganizationID: f.factory.OrganizationID, ExecutionID: execution.ID, RunID: root.RunID, NodeID: models.HumanReviewPrepareNodeID,
				ActionHookContext: core.ActionHookContext{ExecutionState: state, Integration: githubmocks.IntegrationContextForNewSetupFlow(), HTTP: &supportcontexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"current-head"}}`)}}},
			})
			require.NoError(t, err)
			require.Equal(t, "default", state.Channel)
			decision, err := f.pr.CurrentHumanReview(f.db)
			require.NoError(t, err)
			require.NotNil(t, decision)
			require.Equal(t, models.FactoryHumanReviewStateDisabled, decision.State)
			require.Nil(t, decision.RunID)
		})
	}
}
