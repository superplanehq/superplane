package factories

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (f publicationFixture) publicationRequest(t *testing.T, channel string) (*models.CanvasNodeExecution, *models.CanvasNodeRequest) {
	t.Helper()
	root, err := f.run.RootEvent(f.db)
	require.NoError(t, err)
	agent := support.CreateCanvasNodeExecution(t, f.canvas.ID, "agent", root.ID, root.ID)
	agent.Metadata = datatypes.NewJSONType(map[string]any{models.CanvasNodeExecutionFrozenComponentKey: "runnerClaudeCode"})
	require.NoError(t, f.db.Save(agent).Error)
	status := "succeeded"
	if channel == "failed" {
		status = "failed"
	}
	outputs, err := agent.PassInTransaction(f.db, map[string][]any{channel: {map[string]any{"type": "runnerClaudeCode.finished", "data": map[string]any{"status": status}}}})
	require.NoError(t, err)
	require.Len(t, outputs, 1)
	require.NoError(t, f.db.Model(&outputs[0]).Update("state", models.CanvasEventStateRouted).Error)
	execution := support.CreateNextNodeExecution(t, f.canvas.ID, "publish", root.ID, outputs[0].ID, &agent.ID)
	execution.State = models.CanvasNodeExecutionStateStarted
	execution.Metadata = datatypes.NewJSONType(map[string]any{models.CanvasNodeExecutionFrozenComponentKey: factorycomponent.PublishMergeConfidenceComponentName})
	require.NoError(t, f.db.Save(execution).Error)
	require.NoError(t, (&factorycomponent.PublishMergeConfidence{}).Execute(core.ExecutionContext{
		RunID:    execution.RunID,
		Metadata: workercontexts.NewExecutionMetadataContext(f.db, execution),
		Requests: workercontexts.NewExecutionRequestContext(f.db, execution),
		BaseURL:  "https://example.com",
	}))
	var request models.CanvasNodeRequest
	require.NoError(t, f.db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).First(&request).Error)
	return execution, &request
}

func TestMergeConfidenceActionRetriesAndRechecksEligibility(t *testing.T) {
	for _, superseded := range []bool{false, true} {
		t.Run(map[bool]string{false: "latest description", true: "superseded assessment"}[superseded], func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 4)
			host := &contexts.HTTPContext{Responses: []*http.Response{
				publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"},"body":"original"}`),
				publicationHTTPResponse(503, `{"message":"unavailable"}`),
				publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"},"body":"latest user edit"}`),
				publicationHTTPResponse(200, `{}`),
			}}
			installPublicationGitHub(t, host)
			execution, request := f.publicationRequest(t, "passed")
			assert.Equal(t, f.run.ID, execution.RunID)
			_, err := ProcessMergeConfidencePublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
			assert.Equal(t, "retrying", execution.Metadata.Data()["status"])
			assert.EqualValues(t, 1, execution.Metadata.Data()["attempts"])
			assert.NotEmpty(t, execution.Metadata.Data()["lastError"])
			assert.Equal(t, factorycomponent.PublishMergeConfidenceComponentName, execution.FrozenComponentName())
			assert.NotEqual(t, models.CanvasNodeExecutionStateFinished, execution.State)
			var retry models.CanvasNodeRequest
			require.NoError(t, f.db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).First(&retry).Error)
			assert.True(t, retry.RunAt.After(time.Now()))
			if superseded {
				newer, err := models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "trigger", models.CanvasRunStateStarted, "")
				require.NoError(t, err)
				_, err = f.pr.CreateActivity(f.db, models.FactoryPullRequestActivityParams{RunID: newer.ID})
				require.NoError(t, err)
			}
			_, err = ProcessMergeConfidencePublicationRequest(t.Context(), IntakeDependencies{}, retry.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
			assert.Equal(t, models.CanvasNodeExecutionStateFinished, execution.State)
			assert.Equal(t, models.CanvasNodeExecutionResultPassed, execution.Result)
			assert.EqualValues(t, 2, execution.Metadata.Data()["attempts"])
			if superseded {
				assert.Equal(t, "skipped", execution.Metadata.Data()["status"])
				assert.NotEmpty(t, execution.Metadata.Data()["reason"])
				require.Len(t, host.Requests, 2)
			} else {
				assert.Equal(t, "published", execution.Metadata.Data()["status"])
				require.Len(t, host.Requests, 4)
				body, err := io.ReadAll(host.Requests[3].Body)
				require.NoError(t, err)
				var update map[string]string
				require.NoError(t, json.Unmarshal(body, &update))
				assert.True(t, strings.HasPrefix(update["body"], "latest user edit"))
			}
			_, err = ProcessMergeConfidencePublicationRequest(t.Context(), IntakeDependencies{}, retry.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			var source models.CanvasRun
			require.NoError(t, f.db.First(&source, "id = ?", f.run.ID).Error)
			assert.Equal(t, models.CanvasRunStateStarted, source.State)
			result, err := source.CalculateResult(f.db)
			require.NoError(t, err)
			assert.Equal(t, models.CanvasRunResultPassed, result)
		})
	}
}

func TestMergeConfidenceActionRecoversInterruptedAttempt(t *testing.T) {
	f := newPublicationFixture(t)
	f.report(t, f.run, 5)
	execution, request := f.publicationRequest(t, "failed")
	expired := time.Now().Add(-time.Minute)
	require.NoError(t, setMergeConfidencePublicationState(f.db, execution, mergeConfidencePublicationState{SourceRunID: f.run.ID.String(), Status: "publishing", Attempts: 1, LeaseUntil: &expired}))
	require.NoError(t, request.Reschedule(f.db, expired))
	host := &publicationHTTPContext{
		HTTPContext: contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"}}`), publicationHTTPResponse(200, `{}`)}},
		beforeRequest: func(*http.Request) {
			// A duplicate delivery while the recovered attempt owns the lease
			// cannot start another host request.
			_, err := ProcessMergeConfidencePublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
		},
	}
	previous := newFactoryGitHubAPI
	t.Cleanup(func() { newFactoryGitHubAPI = previous })
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return publicationGitHubClient(t, host), nil
	}
	_, err := ProcessMergeConfidencePublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequest)
	require.NoError(t, err)
	require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, execution.Result)
	assert.Equal(t, "published", execution.Metadata.Data()["status"])
	assert.EqualValues(t, 2, execution.Metadata.Data()["attempts"])
	require.Len(t, host.Requests, 2)
	body, err := io.ReadAll(host.Requests[1].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "**Incomplete**")
	result, err := f.run.CalculateResult(f.db)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultFailed, result)
}

func TestMergeConfidenceActionSkipsWriteAfterCancellationDuringDescriptionRead(t *testing.T) {
	cancellations := map[string]func(t *testing.T, f publicationFixture, execution *models.CanvasNodeExecution){
		"run": func(t *testing.T, f publicationFixture, _ *models.CanvasNodeExecution) {
			run, err := models.FindCanvasRunInTransaction(f.db, f.canvas.ID, f.run.ID)
			require.NoError(t, err)
			require.NoError(t, run.MarkAsCancelling(f.db, nil))
		},
		"execution": func(t *testing.T, f publicationFixture, execution *models.CanvasNodeExecution) {
			current, err := models.FindNodeExecutionInTransaction(f.db, execution.WorkflowID, execution.ID)
			require.NoError(t, err)
			require.NoError(t, current.RequestCancellation(f.db, nil))
		},
	}
	for name, cancel := range cancellations {
		t.Run(name, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 4)
			execution, request := f.publicationRequest(t, "passed")
			host := &publicationHTTPContext{
				HTTPContext: contexts.HTTPContext{Responses: []*http.Response{
					publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"},"body":"original"}`),
					publicationHTTPResponse(200, `{}`),
				}},
				beforeRequest: func(request *http.Request) {
					if request.Method == http.MethodGet {
						cancel(t, f, execution)
					}
				},
			}
			previous := newFactoryGitHubAPI
			t.Cleanup(func() { newFactoryGitHubAPI = previous })
			newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
				return publicationGitHubClient(t, host), nil
			}
			_, err := ProcessMergeConfidencePublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			require.Len(t, host.Requests, 1, "a cancellation during the description read must stop the write")
		})
	}
}
