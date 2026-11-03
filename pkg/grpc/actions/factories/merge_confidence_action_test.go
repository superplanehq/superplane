package factories

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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

func (f publicationFixture) publicationRequest(t *testing.T, sourceRunID uuid.UUID) (*models.CanvasNodeExecution, *models.CanvasNodeRequest) {
	t.Helper()
	event := &models.CanvasEvent{WorkflowID: f.canvas.ID, NodeID: "publication", State: models.CanvasEventStateRouted, Data: models.NewJSONValue(map[string]any{"data": map[string]any{"runId": sourceRunID.String()}})}
	require.NoError(t, f.db.Create(event).Error)
	execution := support.CreateCanvasNodeExecution(t, f.canvas.ID, "publish", event.ID, event.ID)
	execution.Configuration = datatypes.NewJSONType(map[string]any{"sourceRunId": sourceRunID.String()})
	execution.State = models.CanvasNodeExecutionStateStarted
	execution.Metadata = datatypes.NewJSONType(map[string]any{models.CanvasNodeExecutionFrozenComponentKey: factorycomponent.PublishMergeConfidenceComponentName})
	require.NoError(t, f.db.Save(execution).Error)
	require.NoError(t, (&factorycomponent.PublishMergeConfidence{}).Execute(core.ExecutionContext{
		Configuration: execution.Configuration.Data(),
		Metadata:      workercontexts.NewExecutionMetadataContext(f.db, execution),
		Requests:      workercontexts.NewExecutionRequestContext(f.db, execution),
		BaseURL:       "https://example.com",
	}))
	var request models.CanvasNodeRequest
	require.NoError(t, f.db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).First(&request).Error)
	return execution, &request
}

func TestMergeConfidenceActionRetriesAndRechecksEligibility(t *testing.T) {
	for _, superseded := range []bool{false, true} {
		t.Run(map[bool]string{false: "latest description", true: "superseded assessment"}[superseded], func(t *testing.T) {
			f := newPublicationFixture(t)
			f.finish(t, models.CanvasRunResultPassed)
			f.report(t, f.run, 4)
			host := &contexts.HTTPContext{Responses: []*http.Response{
				publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"},"body":"original"}`),
				publicationHTTPResponse(503, `{"message":"unavailable"}`),
				publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"},"body":"latest user edit"}`),
				publicationHTTPResponse(200, `{}`),
			}}
			installPublicationGitHub(t, host)
			execution, request := f.publicationRequest(t, f.run.ID)
			// Even a linked publication run must not supersede the assessment.
			_, err := f.pr.CreateActivity(f.db, models.FactoryPullRequestActivityParams{RunID: execution.RunID})
			require.NoError(t, err)
			_, err = ProcessMergeConfidencePublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
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
			assert.Equal(t, models.CanvasRunResultPassed, source.Result)
		})
	}
}

func TestMergeConfidenceActionRejectsOtherFactoryRuns(t *testing.T) {
	f := newPublicationFixture(t)
	factory, err := models.CreateFactory(f.db, f.workspace.OrganizationID, support.RandomName("other"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, f.workspace.OrganizationID, *f.canvas.CreatedBy, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, nil)
	require.NoError(t, f.db.Model(canvas).Update("factory_id", factory.ID).Error)
	source, err := models.CreateCanvasRunInTransaction(f.db, canvas.ID, "trigger", models.CanvasRunStateFinished, "")
	require.NoError(t, err)
	execution, request := f.publicationRequest(t, source.ID)
	_, err = ProcessMergeConfidencePublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
	require.NoError(t, err)
	require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
	assert.Equal(t, models.CanvasNodeExecutionResultFailed, execution.Result)
	assert.Equal(t, "rejected", execution.Metadata.Data()["status"])
}

func TestMergeConfidenceActionRecoversInterruptedAttempt(t *testing.T) {
	f := newPublicationFixture(t)
	f.finish(t, models.CanvasRunResultFailed)
	execution, request := f.publicationRequest(t, f.run.ID)
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
}
