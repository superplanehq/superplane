package workers

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/checks"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	githubmocks "github.com/superplanehq/superplane/test/support/mocks/github"
	"gorm.io/datatypes"
)

func TestFactoryChecksStartupRoutesCurrentResults(t *testing.T) {
	for _, conclusion := range []string{"failure", "success"} {
		t.Run(conclusion, func(t *testing.T) {
			registry.RegisterAction("testFactoryChecksSnapshot", &factoryChecksSnapshotAction{conclusion: conclusion})
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
			require.NoError(t, err)
			pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
			require.NoError(t, err)
			nodes := []models.CanvasNode{{
				NodeID: "renamed-trigger", Type: models.NodeTypeTrigger,
				Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onPullRequest"}}),
				Configuration: datatypes.NewJSONType(map[string]any{"repository": "acme/app"}),
			}}
			for _, step := range []struct {
				id, component string
				config        map[string]any
			}{
				{"activity", "addPullRequestActivity", map[string]any{"pullRequestId": "{{ root().data.pullRequest.id }}", "revision": "{{ root().data.pull_request.head.sha }}", "access": "concurrent"}},
				{"wait", "testFactoryChecksSnapshot", map[string]any{"repository": "acme/app", "ref": "{{ root().data.pull_request.head.sha }}", "checkNames": []any{"build"}}},
				{"repair", "updatePullRequestActivity", map[string]any{"title": "Fixing failed checks", "access": "exclusive"}},
				{"passed", "updatePullRequestActivity", map[string]any{"title": "Checks passed"}},
				{"agent", "noop", map[string]any{}},
			} {
				nodes = append(nodes, models.CanvasNode{NodeID: step.id, Name: step.id, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: step.component}}), Configuration: datatypes.NewJSONType(step.config)})
			}
			canvas, runtime := support.CreateCanvas(t, r.Organization.ID, r.User, nodes, []models.Edge{
				{SourceID: "renamed-trigger", TargetID: "activity", Channel: "default"},
				{SourceID: "activity", TargetID: "wait", Channel: "default"},
				{SourceID: "wait", TargetID: "repair", Channel: "failed"},
				{SourceID: "wait", TargetID: "passed", Channel: "passed"},
				{SourceID: "repair", TargetID: "agent", Channel: "default"},
			})
			require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
			_, err = factory.CreatePRFeedbackHandler(db, canvas.ID, models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, models.FactoryPRFeedbackHandlerSourcePullRequestChecks)
			require.NoError(t, err)
			require.NoError(t, contexts.NewEventContext(db, &runtime[0], nil, nil).Emit("github.pullRequest", map[string]any{
				"action": "opened", "repository": map[string]any{"full_name": "acme/app"},
				"pull_request": map[string]any{"number": 42, "head": map[string]any{"sha": "attached-head", "ref": "feature"}},
				"pullRequest":  map[string]any{"id": pr.ID.String()},
			}))
			router := NewEventRouter("")
			queue := NewNodeQueueWorker(r.Registry, "")
			executor := newTestNodeExecutor(t, r)
			logger := log.NewEntry(log.New())
			for tick := 0; tick < 10; tick++ {
				var pendingEvents []models.CanvasEvent
				require.NoError(t, db.Where("workflow_id = ? AND state = ?", canvas.ID, models.CanvasEventStatePending).Find(&pendingEvents).Error)
				for _, event := range pendingEvents {
					require.NoError(t, router.LockAndProcessEvent(logger, event, time.Now()))
				}
				for _, node := range runtime {
					require.NoError(t, queue.LockAndProcessNode(logger, node, time.Now()))
				}
				var executions []models.CanvasNodeExecution
				require.NoError(t, db.Where("workflow_id = ? AND state = ?", canvas.ID, models.CanvasNodeExecutionStatePending).Find(&executions).Error)
				for _, execution := range executions {
					require.NoError(t, executor.LockAndProcessNodeExecution(execution.ID))
				}
				if len(pendingEvents) == 0 && len(executions) == 0 {
					break
				}
			}
			var agentRuns, passedRuns int64
			require.NoError(t, db.Model(&models.CanvasNodeExecution{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "agent").Count(&agentRuns).Error)
			require.NoError(t, db.Model(&models.CanvasNodeExecution{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "passed").Count(&passedRuns).Error)
			var root models.CanvasEvent
			require.NoError(t, db.Where("workflow_id = ? AND node_id = ?", canvas.ID, "renamed-trigger").First(&root).Error)
			activity, err := models.FindPullRequestActivityByRunID(db, root.RunID)
			require.NoError(t, err)
			if conclusion == "failure" {
				assert.EqualValues(t, 1, agentRuns)
				assert.Zero(t, passedRuns)
				assert.Equal(t, models.FactoryPullRequestAccessExclusive, activity.Access)
			} else {
				assert.Zero(t, agentRuns)
				assert.EqualValues(t, 1, passedRuns)
				assert.Equal(t, "Checks passed", activity.Title)
			}
		})
	}
}

// Replace only GitHub I/O; the worker invokes the real checks evaluation.
type factoryChecksSnapshotAction struct {
	checks.WaitForPullRequestChecks
	conclusion string
}

func (a *factoryChecksSnapshotAction) Execute(ctx core.ExecutionContext) error {
	ctx.HTTP = a
	ctx.Integration = githubmocks.IntegrationContextForNewSetupFlow()
	return a.WaitForPullRequestChecks.Execute(ctx)
}

func (a *factoryChecksSnapshotAction) Do(request *http.Request) (*http.Response, error) {
	if strings.HasSuffix(request.URL.Path, "/status") {
		return githubmocks.GitHubResponse(http.StatusOK, `{"total_count":0,"statuses":[]}`), nil
	}
	if strings.HasSuffix(request.URL.Path, "/check-runs") {
		return githubmocks.GitHubResponse(http.StatusOK, fmt.Sprintf(`{"total_count":1,"check_runs":[{"id":1,"name":"build","head_sha":"attached-head","status":"completed","conclusion":%q}]}`, a.conclusion)), nil
	}
	return nil, fmt.Errorf("unexpected GitHub request %s", request.URL.Path)
}
