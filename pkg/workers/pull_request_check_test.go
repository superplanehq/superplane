package workers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func (f *reviewCheckFixture) applyStatus(state string) error {
	f.ctx.Configuration = map[string]any{
		"repository": "hello", "pullNumber": 42, "assessedSha": "abc",
		"context": "team/review", "state": state, "reason": "Assessment result",
		"resultUrl": "https://superplane.example/run",
	}
	return (&pulls.SetPullRequestCheck{}).Execute(f.ctx)
}

func TestPullRequestCheckPublishesWithoutFactoryOrReviewers(t *testing.T) {
	f := newPullRequestCheckFixture(t, "github.setPullRequestCheck")
	require.NoError(t, f.applyStatus("pending"))
	require.NoError(t, f.processDue())
	require.Equal(t, "pending", f.state())
	f.finishAction()
	require.NoError(t, f.applyStatus("success"))
	require.NoError(t, f.processDue())
	require.Equal(t, "success", f.state())
	require.Equal(t, "https://superplane.example/run", f.http.writes[1].Status.GetTargetURL())
	require.Nil(t, f.record().WakeAt)
	writes := len(f.http.writes)
	require.NoError(t, f.applyStatus("success"))
	require.NoError(t, f.refresh())
	require.Len(t, f.http.writes, writes)
	require.Zero(t, f.http.requestCount)

	// The final state cannot be reopened by a delayed start from this attempt.
	require.NoError(t, f.applyStatus("pending"))
	require.NoError(t, f.refresh())
	require.Equal(t, "success", f.state())
	old := f.root
	f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
	require.NoError(t, f.applyStatus("failure"))
	require.NoError(t, f.refresh())
	require.Equal(t, "failure", f.state())
	generation := f.record().Generation
	f.attempt(old)
	require.NoError(t, f.applyStatus("success"))
	require.Equal(t, generation, f.record().Generation)
}

func TestPullRequestCheckRecoversTerminalDeliveryAfterOwnerEnds(t *testing.T) {
	for _, scenario := range []string{"cancelled", "failed", "finished without output", "removed", "retention"} {
		t.Run(scenario, func(t *testing.T) {
			f := newPullRequestCheckFixture(t, "github.setPullRequestCheck")
			require.NoError(t, f.applyStatus("pending"))
			require.NoError(t, f.refresh())
			switch scenario {
			case "cancelled":
				require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", f.ctx.RunID).Update("state", models.CanvasRunStateCancelling).Error)
			case "failed":
				require.NoError(t, f.ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, "agent failed"))
				// Failed steps resolve pending after the run finalizer ends the run.
				require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", f.ctx.RunID).Update("state", models.CanvasRunStateFinished).Error)
			case "finished without output":
				require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", f.ctx.RunID).Update("state", models.CanvasRunStateFinished).Error)
			case "removed":
				require.NoError(t, f.db.Delete(f.canvas).Error)
			case "retention":
				run, err := models.FindCanvasRunInTransaction(f.db, f.canvas.ID, f.ctx.RunID)
				require.NoError(t, err)
				_, err = run.DeleteChain(f.db)
				require.NoError(t, err)
			}
			f.http.failWrite = true
			require.Error(t, f.processDue())
			require.NotNil(t, f.record().WakeAt)
			f.http.failWrite = false
			require.NoError(t, f.processDue())
			require.Equal(t, "error", f.state())
			require.Nil(t, f.record().WakeAt)
		})
	}
}

func TestPullRequestCheckKeepsIncompleteAssessmentPendingUntilFinalStep(t *testing.T) {
	f := newPullRequestCheckFixture(t, "github.setPullRequestCheck")
	require.NoError(t, f.applyStatus("pending"))
	require.NoError(t, f.processDue())
	f.finishAction()
	run, err := models.FindCanvasRunInTransaction(f.db, f.canvas.ID, f.ctx.RunID)
	require.NoError(t, err)
	require.NoError(t, run.AddError(f.db, "Assessment did not report all checks", 1024))
	require.NoError(t, f.processDue())
	require.Equal(t, "pending", f.state())

	f.attempt(f.root)
	require.NoError(t, f.applyStatus("failure"))
	f.http.failWrite = true
	require.Error(t, f.processDue())
	f.http.failWrite = false
	require.NoError(t, f.processDue())
	require.Equal(t, "failure", f.state())
	require.Nil(t, f.record().WakeAt)
}

func TestPullRequestCheckDoesNotPublishForDeletedCredentialsOrOrganization(t *testing.T) {
	for _, organization := range []bool{true, false} {
		t.Run(map[bool]string{true: "organization", false: "integration"}[organization], func(t *testing.T) {
			f := newPullRequestCheckFixture(t, "github.setPullRequestCheck")
			require.NoError(t, f.applyStatus("pending"))
			if organization {
				require.NoError(t, f.db.Model(&models.Organization{}).Where("id = ?", f.r.Organization.ID).Update("deleted_at", time.Now()).Error)
			} else {
				require.NoError(t, f.db.Delete(f.installation).Error)
			}
			require.NoError(t, f.refresh())
			require.Empty(t, f.http.writes)
			require.Nil(t, f.record().WakeAt)
		})
	}
}

func TestPullRequestCheckRetiresPendingStatusOnAnOlderCommit(t *testing.T) {
	f := newPullRequestCheckFixture(t, "github.setPullRequestCheck")
	require.NoError(t, f.applyStatus("pending"))
	require.NoError(t, f.refresh())
	f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
	f.http.head = "new-head"
	f.ctx.Configuration = map[string]any{"repository": "hello", "pullNumber": 42, "assessedSha": "new-head", "context": "team/review", "state": "pending", "reason": "New assessment"}
	require.NoError(t, (&pulls.SetPullRequestCheck{}).Execute(f.ctx))
	f.http.failWrite = true
	require.Error(t, f.processDue())
	f.http.failWrite = false
	require.NoError(t, f.processDue())
	require.Equal(t, "error", f.state())
	require.Equal(t, "abc", f.http.writes[1].SHA)
	require.NoError(t, f.processDue())
	require.Equal(t, "pending", f.state())
	require.Equal(t, "new-head", f.http.writes[2].SHA)
}

func TestPullRequestCheckKeepsUndeliveredTerminalResultOnAnOlderCommit(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		state string
		push  bool
	}{
		{name: "head changed", state: "error", push: true},
		{name: "final result", state: "success"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newPullRequestCheckFixture(t, "github.setPullRequestCheck")
			require.NoError(t, f.applyStatus("pending"))
			require.NoError(t, f.refresh())
			require.Equal(t, "pending", f.state())
			if scenario.push {
				f.http.head = "new-head"
			} else {
				f.finishAction()
				require.NoError(t, f.applyStatus(scenario.state))
			}
			// The old commit saves its terminal result, but GitHub rejects the write.
			f.http.failWrite = true
			require.Error(t, f.processDue())
			f.http.failWrite = false

			f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
			f.http.head = "new-head"
			f.ctx.Configuration = map[string]any{"repository": "hello", "pullNumber": 42, "assessedSha": "new-head", "context": "team/review", "state": "pending", "reason": "New assessment"}
			require.NoError(t, (&pulls.SetPullRequestCheck{}).Execute(f.ctx))
			require.NoError(t, f.processDue())
			old := f.http.writes[len(f.http.writes)-1]
			require.Equal(t, "abc", old.SHA)
			require.Equal(t, scenario.state, old.Status.GetState())
			require.NoError(t, f.processDue())
			require.Equal(t, "pending", f.state())
			require.Equal(t, "new-head", f.http.writes[len(f.http.writes)-1].SHA)
		})
	}
}

func TestPullRequestCheckStopsUnavailableCredentialsAfterRetries(t *testing.T) {
	f := newPullRequestCheckFixture(t, "github.setPullRequestCheck")
	require.NoError(t, f.applyStatus("pending"))
	integration := f.integration.IntegrationContext.(*testcontexts.IntegrationContext)
	integration.CurrentSecrets = nil
	for range 9 {
		require.Error(t, f.processDue())
	}
	require.NoError(t, f.processDue())
	require.Empty(t, f.http.writes)
	require.Nil(t, f.record().WakeAt)
	require.Empty(t, pendingActionRequests(t, f.installation.ID, pulls.ReviewCheckHook))
}

func TestPullRequestCheckStopsPermissionFailuresAndAllowsExplicitRetry(t *testing.T) {
	f := newPullRequestCheckFixture(t, "github.setPullRequestCheck")
	require.NoError(t, f.applyStatus("success"))
	f.http.writeStatus = 403
	require.NoError(t, f.processDue())
	require.Nil(t, f.record().WakeAt)
	var request models.CanvasNodeRequest
	require.NoError(t, f.db.Where("execution_id = ? AND state = ?", f.ctx.ID, models.NodeExecutionRequestStatePending).First(&request).Error)
	require.NoError(t, NewNodeRequestExecutor(f.r.Encryptor, f.r.Registry, "", nil).RunNow(request))
	var execution models.CanvasNodeExecution
	require.NoError(t, f.db.First(&execution, "id = ?", f.ctx.ID).Error)
	require.Equal(t, models.CanvasNodeExecutionResultFailed, execution.Result)
	require.Contains(t, execution.ResultMessage, "permissions")
	f.http.writeStatus = 0
	// Retry a publication step, rather than the failed agent/run.
	f.attempt(f.root)
	require.NoError(t, f.applyStatus("success"))
	require.NoError(t, f.processDue())
	require.Equal(t, "success", f.state())
}

func TestPullRequestCheckRechecksOwnerAfterProviderRead(t *testing.T) {
	for _, organization := range []bool{true, false} {
		t.Run(map[bool]string{true: "organization", false: "canvas"}[organization], func(t *testing.T) {
			f := newPullRequestCheckFixture(t, "github.setPullRequestCheck")
			require.NoError(t, f.applyStatus("success"))
			f.http.onRead = func() {
				if organization {
					require.NoError(t, f.db.Model(&models.Organization{}).Where("id = ?", f.r.Organization.ID).Update("deleted_at", time.Now()).Error)
					return
				}
				require.NoError(t, f.db.Delete(f.canvas).Error)
			}
			require.NoError(t, f.refresh())
			require.Nil(t, f.record().WakeAt)
			if organization {
				require.Empty(t, f.http.writes, "deletion during provider reads must prevent publication")
				return
			}
			require.Equal(t, "error", f.state(), "removal during provider reads must not publish success")
		})
	}
}
