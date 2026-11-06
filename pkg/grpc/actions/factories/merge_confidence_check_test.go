package factories

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

func TestMergeConfidenceCheckTracksRunToCompletionOnItsOriginalHead(t *testing.T) {
	t.Setenv("BASE_URL", "https://superplane.example")
	f, request := newMergeConfidenceCheckFixture(t)
	host := &contexts.HTTPContext{Responses: []*http.Response{
		publicationHTTPResponse(200, `{"check_runs":[]}`),
		publicationHTTPResponse(201, `{"id":123}`),
		mergeConfidenceCheckListResponse(f.run.ID, "in_progress"),
		publicationHTTPResponse(200, `{"id":123}`),
	}}
	installPublicationGitHub(t, host)

	require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
	require.Len(t, host.Requests, 2)
	started := mergeConfidenceCheckRequestBody(t, host.Requests[1])
	assert.Equal(t, http.MethodPost, host.Requests[1].Method)
	assert.Equal(t, "/repos/acme/app/check-runs", host.Requests[1].URL.Path)
	assert.Equal(t, "SuperPlane merge confidence", started["name"])
	assert.Equal(t, "abc", started["head_sha"])
	assert.Equal(t, f.run.ID.String(), started["external_id"])
	assert.Equal(t, "in_progress", started["status"])
	assert.Empty(t, started["conclusion"])
	assert.Equal(t, "https://superplane.example"+f.order.URLPath(f.workspace.RouteSegment()), started["details_url"])
	stored, err := models.FindCanvasNodeRequest(f.db, request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStatePending, stored.State)
	assert.True(t, stored.RunAt.After(time.Now()))

	_, err = f.pr.ObserveRevision(f.db, "newer-head")
	require.NoError(t, err)
	// A low score is advisory. Successful assessment completion is the check result.
	emitMergeConfidenceCheckAssessment(t, f, "passed", true)
	f.finish(t, models.CanvasRunResultPassed)
	require.NoError(t, request.Reschedule(f.db, time.Now().Add(-time.Second)))
	require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
	require.Len(t, host.Requests, 4)
	assert.Equal(t, "/repos/acme/app/commits/abc/check-runs", host.Requests[2].URL.Path)
	assert.Equal(t, http.MethodPatch, host.Requests[3].Method)
	assert.Equal(t, "/repos/acme/app/check-runs/123", host.Requests[3].URL.Path)
	finished := mergeConfidenceCheckRequestBody(t, host.Requests[3])
	assert.Equal(t, f.run.ID.String(), finished["external_id"])
	assert.Equal(t, "completed", finished["status"])
	assert.Equal(t, "success", finished["conclusion"])
	stored, err = models.FindCanvasNodeRequest(f.db, request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, stored.State)

	require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
	assert.Len(t, host.Requests, 4, "a completed delivery must not publish again")
}

func TestMergeConfidenceCheckReportsTerminalAssessmentFailures(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		channel    string
		report     bool
		result     string
		conclusion string
	}{
		{name: "missing result", result: models.CanvasRunResultPassed, conclusion: "failure"},
		{name: "incomplete result", channel: "passed", result: models.CanvasRunResultPassed, conclusion: "failure"},
		{name: "failed run", channel: "passed", report: true, result: models.CanvasRunResultFailed, conclusion: "failure"},
		{name: "cancelled run", result: models.CanvasRunResultCancelled, conclusion: "cancelled"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f, request := newMergeConfidenceCheckFixture(t)
			if scenario.channel != "" {
				emitMergeConfidenceCheckAssessment(t, f, scenario.channel, scenario.report)
			}
			f.finish(t, scenario.result)
			host := &contexts.HTTPContext{Responses: []*http.Response{
				publicationHTTPResponse(200, `{"check_runs":[]}`),
				publicationHTTPResponse(201, `{"id":123}`),
			}}
			installPublicationGitHub(t, host)
			require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
			require.Len(t, host.Requests, 2)
			body := mergeConfidenceCheckRequestBody(t, host.Requests[1])
			assert.Equal(t, "completed", body["status"])
			assert.Equal(t, scenario.conclusion, body["conclusion"])
			stored, err := models.FindCanvasNodeRequest(f.db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStateCompleted, stored.State)
		})
	}
}

func TestMergeConfidenceCheckRetriesPublicationAndRecoversAcceptedCreate(t *testing.T) {
	for _, accepted := range []bool{false, true} {
		t.Run(fmt.Sprintf("GitHub accepted create %t", accepted), func(t *testing.T) {
			f, request := newMergeConfidenceCheckFixture(t)
			emitMergeConfidenceCheckAssessment(t, f, "passed", true)
			f.finish(t, models.CanvasRunResultPassed)
			nextList := publicationHTTPResponse(200, `{"check_runs":[]}`)
			if accepted {
				// GitHub can persist a check before a gateway loses its response.
				nextList = mergeConfidenceCheckListResponse(f.run.ID, "in_progress")
			}
			host := &contexts.HTTPContext{Responses: []*http.Response{
				publicationHTTPResponse(200, `{"check_runs":[]}`),
				publicationHTTPResponse(502, `{"message":"response lost"}`),
				nextList,
				publicationHTTPResponse(200, `{"id":123}`),
			}}
			installPublicationGitHub(t, host)
			require.Error(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
			stored, err := models.FindCanvasNodeRequest(f.db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStatePending, stored.State)
			assert.True(t, stored.RunAt.After(time.Now().Add(45*time.Second)))
			require.NoError(t, request.Reschedule(f.db, time.Now().Add(-time.Second)))
			require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
			require.Len(t, host.Requests, 4)
			method, path := http.MethodPost, "/repos/acme/app/check-runs"
			if accepted {
				method, path = http.MethodPatch, "/repos/acme/app/check-runs/123"
			}
			assert.Equal(t, method, host.Requests[3].Method)
			assert.Equal(t, path, host.Requests[3].URL.Path)
			for _, index := range []int{1, 3} {
				body := mergeConfidenceCheckRequestBody(t, host.Requests[index])
				assert.Equal(t, f.run.ID.String(), body["external_id"])
				assert.Equal(t, "success", body["conclusion"])
			}
			stored, err = models.FindCanvasNodeRequest(f.db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStateCompleted, stored.State)
		})
	}
}

func TestMergeConfidenceCheckDoesNotReplaceNewerAssessmentForTheSameHead(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("older check exists %t", existing), func(t *testing.T) {
			f, request := newMergeConfidenceCheckFixture(t)
			newer := f
			newer.startRun(t)
			emitMergeConfidenceCheckAssessment(t, newer, "passed", true)
			newer.finish(t, models.CanvasRunResultPassed)
			list := publicationHTTPResponse(200, `{"check_runs":[]}`)
			if existing {
				list = publicationHTTPResponse(200, fmt.Sprintf(`{"check_runs":[
					{"id":123,"external_id":%q,"status":"in_progress"},
					{"id":456,"external_id":%q,"status":"completed","conclusion":"success"}
				]}`, f.run.ID.String(), newer.run.ID.String()))
			}
			host := &contexts.HTTPContext{Responses: []*http.Response{list, publicationHTTPResponse(200, `{"id":123}`)}}
			installPublicationGitHub(t, host)
			require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
			for _, sent := range host.Requests {
				assert.NotEqual(t, http.MethodPost, sent.Method, "a delayed old run must not become the latest named check")
			}
			if existing {
				require.Len(t, host.Requests, 2)
				assert.Equal(t, http.MethodPatch, host.Requests[1].Method)
				assert.Equal(t, "/repos/acme/app/check-runs/123", host.Requests[1].URL.Path)
				body := mergeConfidenceCheckRequestBody(t, host.Requests[1])
				assert.Equal(t, "completed", body["status"])
				assert.Equal(t, "cancelled", body["conclusion"])
			}
			stored, err := models.FindCanvasNodeRequest(f.db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStateCompleted, stored.State)
		})
	}
}

func TestMergeConfidenceCheckClosesPublishedCheckAfterCanvasDeletion(t *testing.T) {
	f, request := newMergeConfidenceCheckFixture(t)
	require.NoError(t, f.db.Model(f.canvas).Update("deleted_at", time.Now()).Error)
	host := &contexts.HTTPContext{Responses: []*http.Response{
		mergeConfidenceCheckListResponse(f.run.ID, "in_progress"),
		publicationHTTPResponse(200, `{"id":123}`),
	}}
	installPublicationGitHub(t, host)
	require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
	require.Len(t, host.Requests, 2)
	assert.Equal(t, http.MethodPatch, host.Requests[1].Method)
	body := mergeConfidenceCheckRequestBody(t, host.Requests[1])
	assert.Equal(t, "completed", body["status"])
	assert.Equal(t, "cancelled", body["conclusion"])
	stored, err := models.FindCanvasNodeRequest(f.db, request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, stored.State)
}

func newMergeConfidenceCheckFixture(t *testing.T) (publicationFixture, *models.CanvasNodeRequest) {
	t.Helper()
	f := newPublicationFixture(t)
	f.publishVersion(t, func(version *models.CanvasVersion) {
		findIntakeNode(version.Nodes, "agent").ID = "assess-risk"
	})
	f.startRun(t)
	request := &models.CanvasNodeRequest{
		ID: uuid.New(), WorkflowID: f.canvas.ID, NodeID: "trigger",
		State: models.NodeExecutionRequestStatePending, Type: models.NodeRequestTypeMergeConfidenceCheck,
		RunAt: time.Now().Add(-time.Second),
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{MergeConfidenceCheck: &models.MergeConfidenceCheckRequest{
			RunID: f.run.ID, PullRequestID: f.pr.ID,
		}}),
	}
	require.NoError(t, f.db.Create(request).Error)
	return f, request
}

func emitMergeConfidenceCheckAssessment(t *testing.T, f publicationFixture, channel string, report bool) {
	t.Helper()
	if report {
		_, err := factorycomponent.ReportMergeConfidenceCheck(f.db, f.order, f.run.ID,
			&factoryevents.AutomationRef{AppID: f.canvas.ID, NodeID: "assess-risk"},
			f.pr.Repository, "security", 1, "A security issue needs attention.", []string{"security"}, nil)
		require.NoError(t, err)
	}
	root, err := f.run.RootEvent(f.db)
	require.NoError(t, err)
	execution := support.CreateCanvasNodeExecution(t, f.canvas.ID, "assess-risk", root.ID, root.ID)
	state := workercontexts.NewExecutionStateContext(f.db, execution, nil)
	require.NoError(t, state.Emit(channel, "runnerClaudeCode.finished", []any{map[string]any{"status": "finished"}}))
}

func mergeConfidenceCheckListResponse(runID uuid.UUID, status string) *http.Response {
	return publicationHTTPResponse(200, fmt.Sprintf(`{"check_runs":[{"id":123,"external_id":%q,"status":%q}]}`, runID.String(), status))
}

func mergeConfidenceCheckRequestBody(t *testing.T, request *http.Request) map[string]any {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	return payload
}
