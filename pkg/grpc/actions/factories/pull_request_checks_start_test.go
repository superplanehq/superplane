package factories

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomponents "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/checks"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
	githubmocks "github.com/superplanehq/superplane/test/support/mocks/github"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestPullRequestAttachmentBatchesVerificationTriggers(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	checks := createChecksStartupCanvas(t, r, factory, "acme/app")
	expected := []uuid.UUID{checks.ID}

	type queryCountKey struct{}
	const callback = "test:pull_request_attachment_query_count"
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if count, ok := tx.Statement.Context.Value(queryCountKey{}).(*atomic.Int64); ok {
			count.Add(1)
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callback)) })
	attach := func(number int) int64 {
		t.Helper()
		order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
		require.NoError(t, err)
		var count atomic.Int64
		measured := db.WithContext(context.WithValue(t.Context(), queryCountKey{}, &count))
		pr, err := order.CreatePullRequest(measured, models.FactoryPullRequestParams{URL: fmt.Sprintf("https://github.com/acme/app/pull/%d", number)})
		require.NoError(t, err)
		queries := count.Load()
		requests, err := models.ListPullRequestChecksStartRequests(db, pr.ID)
		require.NoError(t, err)
		canvases := make([]uuid.UUID, 0, len(requests))
		for _, request := range requests {
			canvases = append(canvases, request.WorkflowID)
		}
		assert.ElementsMatch(t, expected, canvases)
		return queries
	}
	singleQueries := attach(41)
	for range 8 {
		support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, support.RandomName("unrelated"), "start")
	}
	for range 3 {
		canvas := createChecksStartupCanvas(t, r, factory, "ACME/App")
		expected = append(expected, canvas.ID)
	}
	for range 2 {
		canvas := createMergeConfidenceStartupCanvas(t, r, factory, "acme/app")
		expected = append(expected, canvas.ID)
	}
	otherFactory := seedHandOffFactory(t, r, db)
	createChecksStartupCanvas(t, r, otherFactory, "acme/app")
	createMergeConfidenceStartupCanvas(t, r, factory, "other/app")
	deleted := createChecksStartupCanvas(t, r, factory, "acme/app")
	require.NoError(t, deleted.SoftDeleteInTransaction(db))
	broken := createChecksStartupCanvas(t, r, factory, "acme/app")
	require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", broken.ID).Update("state", models.CanvasNodeStateError).Error)
	discussion := createMergeConfidenceStartupCanvas(t, r, factory, "acme/app")
	_, err := factory.CreatePRFeedbackHandler(db, discussion.ID, models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion)
	require.NoError(t, err)
	unpublished := createMergeConfidenceStartupCanvas(t, r, factory, "acme/app")
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, unpublished)
	require.NoError(t, err)
	version.Nodes[0].Metadata = nil
	require.NoError(t, db.Save(version).Error)
	batchQueries := attach(42)
	t.Logf("Attachment queries: one canvas = %d, many canvases = %d", singleQueries, batchQueries)
	require.Positive(t, singleQueries)
	assert.Equal(t, singleQueries, batchQueries, "attachment queries must not grow with workspace canvases")
}

func TestAttachedPullRequestChecksUseCurrentResults(t *testing.T) {
	for _, outcome := range []string{"failure", "success", "pending", "missing"} {
		t.Run(outcome, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			_, _, _, canvas, request := checksStartupFixture(t, r)
			useGitHub(t, checksStartupGitHubPullRequest())
			events, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, request.ID)
			require.NoError(t, err)
			require.Len(t, events, 1)
			root := events[0]
			activityCtx := attachedChecksExecution(t, canvas, root, prFeedbackActivityNodeID, root, nil)
			require.NoError(t, (&factorycomponents.AddPullRequestActivity{}).Execute(activityCtx))
			var activityEvent models.CanvasEvent
			require.NoError(t, db.Where("execution_id = ?", activityCtx.ID).First(&activityEvent).Error)
			waitCtx := attachedChecksExecution(t, canvas, root, prFeedbackWaitChecksNodeID, activityEvent, &activityCtx.ID)
			waitCtx.Integration = githubmocks.IntegrationContextForNewSetupFlow()
			// The HTTP fixture supplies GitHub snapshots; execution state, requests,
			// revisions and activities use the real persisted contexts.
			waitCtx.HTTP = attachedChecksHTTP(outcome)
			component := &checks.WaitForPullRequestChecks{}
			require.NoError(t, component.Execute(waitCtx))
			var emitted []models.CanvasEvent
			require.NoError(t, db.Where("execution_id = ?", waitCtx.ID).Find(&emitted).Error)
			if outcome == "pending" || outcome == "missing" {
				assert.Empty(t, emitted)
				value, err := models.FindLatestNodeExecutionKVValueInTransaction(db, waitCtx.ID, "waitChecksRef")
				require.NoError(t, err)
				assert.Equal(t, "acme/app@"+checksStartupGitHubPullRequest().GetHead().GetSHA(), value)
				var poll models.CanvasNodeRequest
				require.NoError(t, db.Where("execution_id = ? AND state = ?", waitCtx.ID, models.NodeExecutionRequestStatePending).First(&poll).Error)
				assert.Equal(t, "evaluate", poll.Spec.Data().InvokeAction.ActionName)
				if outcome == "missing" {
					return
				}
				body := []byte(fmt.Sprintf(`{"repository":{"full_name":"acme/app","name":"app"},"check_run":{"head_sha":%q,"name":"build"}}`, checksStartupGitHubPullRequest().GetHead().GetSHA()))
				mac := hmac.New(sha256.New, []byte("test-secret"))
				mac.Write(body)
				headers := http.Header{"X-Github-Event": []string{"check_run"}, "X-Hub-Signature-256": []string{"sha256=" + fmt.Sprintf("%x", mac.Sum(nil))}}
				code, _, err := component.HandleWebhook(core.WebhookRequestContext{
					Body: body, Headers: headers, Logger: log.NewEntry(log.New()), Configuration: waitCtx.Configuration,
					Webhook: &testcontexts.NodeWebhookContext{Secret: "test-secret"},
					FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
						execution, err := models.FirstNodeExecutionByKVInTransaction(db, canvas.ID, prFeedbackWaitChecksNodeID, key, value)
						if err != nil {
							return nil, err
						}
						require.Equal(t, waitCtx.ID, execution.ID)
						return &waitCtx, nil
					},
				})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, code)
				var wake models.CanvasNodeRequest
				require.NoError(t, db.Where("execution_id = ? AND state = ?", waitCtx.ID, models.NodeExecutionRequestStatePending).First(&wake).Error)
				assert.WithinDuration(t, time.Now().Add(time.Second), wake.RunAt, time.Second)
				require.NoError(t, component.HandleHook(core.ActionHookContext{Name: "evaluate", Configuration: waitCtx.Configuration, HTTP: attachedChecksHTTP("failure"), Integration: waitCtx.Integration, ExecutionState: waitCtx.ExecutionState, Requests: waitCtx.Requests, Metadata: waitCtx.Metadata, Logger: waitCtx.Logger}))
				require.NoError(t, db.Where("execution_id = ?", waitCtx.ID).Find(&emitted).Error)
				outcome = "failure"
			}
			require.Len(t, emitted, 1)
			branch := prFeedbackStartRepairNodeID
			channel := "failed"
			if outcome == "success" {
				branch = prFeedbackMarkPassedNodeID
				channel = "passed"
			}
			assert.Equal(t, channel, emitted[0].Channel)
			branchCtx := attachedChecksExecution(t, canvas, root, branch, emitted[0], &waitCtx.ID)
			require.NoError(t, (&factorycomponents.UpdatePullRequestActivity{}).Execute(branchCtx))
			activity, err := models.FindPullRequestActivityByRunID(db, root.RunID)
			require.NoError(t, err)
			if outcome == "failure" {
				assert.Contains(t, activity.Title, "Fixing failed checks")
				assert.Equal(t, models.FactoryPullRequestAccessExclusive, activity.Access)
			} else {
				assert.Contains(t, activity.Title, "Checks passed")
				assert.Equal(t, models.FactoryPullRequestAccessConcurrent, activity.Access)
			}
		})
	}
}

func attachedChecksExecution(t *testing.T, canvas *models.Canvas, root models.CanvasEvent, nodeID string, input models.CanvasEvent, previous *uuid.UUID) core.ExecutionContext {
	t.Helper()
	db := database.DB(t.Context())
	node, err := models.FindCanvasNode(db, canvas.ID, nodeID)
	require.NoError(t, err)
	execution := support.CreateNextNodeExecution(t, canvas.ID, nodeID, root.ID, input.ID, previous)
	configuration, err := contexts.NewNodeConfigurationBuilder(db, canvas.ID).WithNodeID(nodeID).WithRootEvent(&root.ID).WithIncomingEventID(&input.ID).WithPreviousExecution(previous).Build(node.Configuration.Data())
	require.NoError(t, err)
	execution.Configuration = datatypes.NewJSONType(configuration)
	require.NoError(t, db.Save(execution).Error)
	require.NoError(t, execution.StartInTransaction(db))
	return core.ExecutionContext{
		ID: execution.ID, Configuration: configuration, Logger: log.NewEntry(log.New()),
		Factory:        contexts.NewFactoryContext(db, canvas, execution),
		ExecutionState: contexts.NewExecutionStateContext(db, execution, nil),
		Requests:       contexts.NewExecutionRequestContext(db, execution),
		Metadata:       contexts.NewExecutionMetadataContext(db, execution),
	}
}

func attachedChecksHTTP(outcome string) core.HTTPContext {
	return &attachedChecksSnapshotHTTP{outcome: outcome}
}

type attachedChecksSnapshotHTTP struct{ outcome string }

func (h *attachedChecksSnapshotHTTP) Do(request *http.Request) (*http.Response, error) {
	if strings.HasSuffix(request.URL.Path, "/status") {
		return githubmocks.GitHubResponse(http.StatusOK, `{"total_count":0,"statuses":[]}`), nil
	}
	if !strings.HasSuffix(request.URL.Path, "/check-runs") {
		return nil, fmt.Errorf("unexpected GitHub request %s", request.URL.Path)
	}
	status, conclusion := "completed", h.outcome
	if h.outcome == "pending" {
		status = "in_progress"
		conclusion = ""
	}
	runs := fmt.Sprintf(`{"total_count":1,"check_runs":[{"id":1,"name":"build","status":%q,"conclusion":%q,"head_sha":%q}]}`, status, conclusion, checksStartupGitHubPullRequest().GetHead().GetSHA())
	if h.outcome == "missing" {
		runs = `{"total_count":0,"check_runs":[]}`
	}
	return githubmocks.GitHubResponse(http.StatusOK, runs), nil
}

func TestPullRequestChecksStartupOverlap(t *testing.T) {
	for _, scenario := range []struct{ mergeConfidence, webhookFirst bool }{{false, true}, {false, false}, {true, true}, {true, false}} {
		t.Run(fmt.Sprintf("merge confidence %t webhook first %t", scenario.mergeConfidence, scenario.webhookFirst), func(t *testing.T) {
			webhookFirst := scenario.webhookFirst
			r := support.Setup(t)
			db := database.DB(t.Context())
			_, order, pullRequest, canvas, request := pullRequestStartupFixture(t, r, scenario.mergeConfidence)
			current := checksStartupGitHubPullRequest()
			current.Head.Repo.FullName = github.Ptr("contributor/app")
			useGitHub(t, current)
			payload, err := pullRequestChecksEventData(pullRequest, order, current)
			require.NoError(t, err)
			node, err := models.FindCanvasNode(db, canvas.ID, "renamed-pr-trigger")
			require.NoError(t, err)
			if webhookFirst {
				emitChecksStartupWebhook(t, node, payload)
			}
			events, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, request.ID)
			require.NoError(t, err)
			if webhookFirst {
				assert.Empty(t, events)
			} else {
				require.Len(t, events, 1)
			}
			if !webhookFirst {
				emitChecksStartupWebhook(t, node, payload)
			}
			assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID))
			var persisted models.CanvasEvent
			require.NoError(t, db.Where("workflow_id = ?", canvas.ID).First(&persisted).Error)
			root := models.RootEventSourcePayload(persisted.Data.Data()).(map[string]any)
			head := root["pull_request"].(map[string]any)["head"].(map[string]any)
			assert.Equal(t, "feat/refunds", head["ref"])
			assert.Equal(t, "contributor/app", head["repo"].(map[string]any)["full_name"])
			assert.Equal(t, current.GetHead().GetSHA(), head["sha"])
			assert.Equal(t, order.ID.String(), root["workOrder"].(map[string]any)["id"])
			// A delayed delivery must remain a duplicate after the original run finishes.
			require.NoError(t, db.Model(&persisted).Update("state", models.CanvasEventStateRouted).Error)
			require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", persisted.RunID).Update("state", models.CanvasRunStateFinished).Error)
			emitChecksStartupWebhook(t, node, payload)
			assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID))
			current.Head.SHA = github.Ptr("fedcba9876543210fedcba9876543210fedcba98")
			payload, err = pullRequestChecksEventData(pullRequest, order, current)
			require.NoError(t, err)
			payload["action"] = "synchronize"
			emitChecksStartupWebhook(t, node, payload)
			assert.EqualValues(t, 2, countCanvasEvents(t, canvas.ID))
		})
	}
}

func TestPullRequestChecksStartupRecovery(t *testing.T) {
	for _, scenario := range []struct {
		mergeConfidence bool
		failure         string
	}{{false, "GitHub unavailable"}, {false, "missing head branch"}, {true, "GitHub unavailable"}, {true, "missing head branch"}} {
		t.Run(fmt.Sprintf("merge confidence %t %s", scenario.mergeConfidence, scenario.failure), func(t *testing.T) {
			failure := scenario.failure
			r := support.Setup(t)
			db := database.DB(t.Context())
			_, _, _, canvas, request := pullRequestStartupFixture(t, r, scenario.mergeConfidence)
			api := &fakeFactoryGitHub{pullRequest: checksStartupGitHubPullRequest()}
			if failure == "GitHub unavailable" {
				api.getPullErr = errors.New(failure)
			} else {
				api.pullRequest.Head.Ref = nil
			}
			stubFactoryGitHub(t, api)
			_, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, request.ID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "will retry")
			saved, err := models.FindCanvasNodeRequest(db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStatePending, saved.State)
			assert.Equal(t, 1, saved.Spec.Data().PullRequestChecksStart.Attempts)
			assert.EqualValues(t, 0, countCanvasEvents(t, canvas.ID))
			api.getPullErr = nil
			api.pullRequest = checksStartupGitHubPullRequest()
			api.pullRequest.Head.SHA = github.Ptr("fedcba9876543210fedcba9876543210fedcba98")
			require.NoError(t, saved.Reschedule(db, time.Now().Add(-time.Second)))
			due, err := models.ListDuePullRequestChecksStartRequests(db)
			require.NoError(t, err)
			require.NotEmpty(t, due)
			events, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, saved.ID)
			require.NoError(t, err)
			require.Len(t, events, 1)
			persisted, err := models.FindCanvasEventForCanvas(db, canvas.ID, events[0].ID)
			require.NoError(t, err)
			payload := models.RootEventSourcePayload(persisted.Data.Data()).(map[string]any)
			assert.Equal(t, api.pullRequest.GetHead().GetSHA(), payload["pull_request"].(map[string]any)["head"].(map[string]any)["sha"])
			_, err = ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, saved.ID)
			require.NoError(t, err)
			assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID))
		})
	}
}

func TestPullRequestChecksStartupSkipsClosedPullRequestAfterProviderFetch(t *testing.T) {
	for _, mergeConfidence := range []bool{false, true} {
		t.Run(fmt.Sprintf("merge confidence %t", mergeConfidence), func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			_, _, pullRequest, canvas, request := pullRequestStartupFixture(t, r, mergeConfidence)
			arrivals := make(chan struct{}, 1)
			release := make(chan struct{})
			resume := sync.OnceFunc(func() { close(release) })
			stubFactoryGitHub(t, &concurrentImportGitHub{
				fakeFactoryGitHub: &fakeFactoryGitHub{pullRequest: checksStartupGitHubPullRequest()},
				arrivals:          arrivals,
				release:           release,
			})
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			type outcome struct {
				events []models.CanvasEvent
				err    error
			}
			outcomes := make(chan outcome, 1)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				events, err := ProcessPullRequestChecksStartRequest(ctx, IntakeDependencies{}, request.ID)
				outcomes <- outcome{events, err}
			}()
			t.Cleanup(func() {
				cancel()
				resume()
				<-finished
			})
			select {
			case <-arrivals:
			case <-ctx.Done():
				t.Fatal("checks startup did not reach provider lookup")
			}

			closed := models.FactoryPullRequestStateClosed
			require.NoError(t, pullRequest.Update(db, models.FactoryPullRequestPatch{State: &closed}))
			resume()
			result := <-outcomes
			require.NoError(t, result.err)
			assert.Len(t, result.events, 0)
			assert.Zero(t, countCanvasEvents(t, canvas.ID), "a closed PR must not start a review")
			saved, err := models.FindCanvasNodeRequest(db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStateCompleted, saved.State)
		})
	}
}

func TestPullRequestChecksStartupEligibility(t *testing.T) {
	for _, condition := range []string{"no handler", "discussion handler", "other repository", "Bitbucket", "closed tracked PR", "closed GitHub PR", "draft ignored", "draft accepted", "handler removed", "canvas deleted", "organization deleted", "missing live version", "trigger error"} {
		t.Run(condition, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory := seedHandOffFactory(t, r, db)
			canvas := createChecksStartupCanvas(t, r, factory, "acme/app")
			handler, err := models.FindPRFeedbackHandlerByCanvasID(db, canvas.ID)
			require.NoError(t, err)
			provider, repository, state := "github", "acme/app", "open"
			switch condition {
			case "no handler":
				require.NoError(t, handler.Delete(db))
			case "discussion handler":
				require.NoError(t, db.Model(handler).Update("source", models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion).Error)
			case "other repository":
				repository = "other/app"
			case "Bitbucket":
				provider = "bitbucket"
			case "closed tracked PR":
				state = "closed"
			}
			order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
			require.NoError(t, err)
			_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{Provider: provider, Repository: repository, Number: 42, URL: "https://" + provider + ".com/" + repository + "/pull/42", State: state})
			require.NoError(t, err)
			current := checksStartupGitHubPullRequest()
			switch condition {
			case "closed GitHub PR":
				current.State = github.Ptr("closed")
			case "draft ignored":
				current.Draft = github.Ptr(true)
			case "draft accepted":
				current.Draft = github.Ptr(true)
				require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).Update("configuration", datatypes.NewJSONType(map[string]any{"repository": "acme/app", "ignoreDrafts": false})).Error)
			case "handler removed":
				require.NoError(t, handler.Delete(db))
			case "canvas deleted":
				require.NoError(t, canvas.SoftDeleteInTransaction(db))
			case "organization deleted":
				require.NoError(t, db.Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("deleted_at", time.Now()).Error)
			case "missing live version":
				other, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
				require.NoError(t, db.Model(canvas).Update("live_version_id", other.LiveVersionID).Error)
			case "trigger error":
				require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).Update("state", models.CanvasNodeStateError).Error)
			}
			useGitHub(t, current)
			var requests []models.CanvasNodeRequest
			require.NoError(t, db.Where("workflow_id = ? AND type = ?", canvas.ID, models.NodeRequestTypePullRequestChecksStart).Find(&requests).Error)
			for _, request := range requests {
				_, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, request.ID)
				require.NoError(t, err)
				saved, err := models.FindCanvasNodeRequest(db, request.ID)
				require.NoError(t, err)
				assert.Equal(t, models.NodeExecutionRequestStateCompleted, saved.State)
			}
			want := int64(0)
			if condition == "draft accepted" {
				want = 1
			}
			assert.Equal(t, want, countCanvasEvents(t, canvas.ID))
		})
	}
}

func TestMergeConfidenceStartupEligibility(t *testing.T) {
	for _, condition := range []string{"other repository", "draft", "closed", "canvas deleted", "template removed", "trigger error"} {
		t.Run(condition, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			_, _, _, canvas, request := pullRequestStartupFixture(t, r, true)
			current := checksStartupGitHubPullRequest()
			switch condition {
			case "other repository":
				version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
				require.NoError(t, err)
				version.Nodes[0].Configuration["repository"] = "other/app"
				require.NoError(t, db.Save(version).Error)
			case "draft":
				current.Draft = github.Ptr(true)
			case "closed":
				current.State = github.Ptr("closed")
			case "canvas deleted":
				require.NoError(t, canvas.SoftDeleteInTransaction(db))
			case "template removed":
				require.NoError(t, canvas.StampFactoryAppTemplate(db, "renamed-pr-trigger", "custom-verification", 1))
			case "trigger error":
				require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).Update("state", models.CanvasNodeStateError).Error)
			}
			useGitHub(t, current)
			events, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, request.ID)
			require.NoError(t, err)
			assert.Empty(t, events)
			assert.Zero(t, countCanvasEvents(t, canvas.ID))
			saved, err := models.FindCanvasNodeRequest(db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStateCompleted, saved.State)
		})
	}
}

func TestPullRequestChecksStartupRollsBackWithAttachment(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	canvas := createChecksStartupCanvas(t, r, factory, "acme/app")
	order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
	require.NoError(t, err)
	err = db.Transaction(func(tx *gorm.DB) error {
		_, err := order.CreatePullRequest(tx, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
		if err != nil {
			return err
		}
		var count int64
		require.NoError(t, tx.Model(&models.CanvasNodeRequest{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
		require.EqualValues(t, 1, count)
		return errors.New("attachment failed")
	})
	require.ErrorContains(t, err, "attachment failed")
	var count int64
	require.NoError(t, db.Model(&models.CanvasNodeRequest{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
	assert.Zero(t, count)
	require.NoError(t, db.Model(&models.FactoryPullRequest{}).Where("work_order_id = ?", order.ID).Count(&count).Error)
	assert.Zero(t, count)
	assert.EqualValues(t, 0, countCanvasEvents(t, canvas.ID))
}

func checksStartupGitHubPullRequest() *github.PullRequest {
	pr := openHandOffPullRequest("acme/app", 42, "feat/refunds")
	pr.Head.SHA = github.Ptr("0123456789abcdef0123456789abcdef01234567")
	pr.Base.Ref = github.Ptr("main")
	pr.Body = github.Ptr("Fix refunds")
	return pr
}

func checksStartupFixture(t *testing.T, r *support.ResourceRegistry) (*models.Factory, *models.FactoryWorkOrder, *models.FactoryPullRequest, *models.Canvas, *models.CanvasNodeRequest) {
	t.Helper()
	return pullRequestStartupFixture(t, r, false)
}

func pullRequestStartupFixture(t *testing.T, r *support.ResourceRegistry, mergeConfidence bool) (*models.Factory, *models.FactoryWorkOrder, *models.FactoryPullRequest, *models.Canvas, *models.CanvasNodeRequest) {
	t.Helper()
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	createCanvas := createChecksStartupCanvas
	if mergeConfidence {
		createCanvas = createMergeConfidenceStartupCanvas
	}
	canvas := createCanvas(t, r, factory, "acme/app")
	order, err := factory.CreateWorkOrder(db, "Review", "Fix refunds", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	requests, err := models.ListPullRequestChecksStartRequests(db, pullRequest.ID)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	return factory, order, pullRequest, canvas, &requests[0]
}

// Merge confidence uses the same PR snapshot and recovery contract as checks,
// but is identified by template metadata rather than a feedback handler.
func createMergeConfidenceStartupCanvas(t *testing.T, r *support.ResourceRegistry, factory *models.Factory, repository string) *models.Canvas {
	t.Helper()
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "renamed-pr-trigger", Type: models.NodeTypeTrigger,
		Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onPullRequest"}}),
		Configuration: datatypes.NewJSONType(map[string]any{"repository": repository, "ignoreDrafts": true, "onlyFactoryPullRequests": true}),
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	require.NoError(t, canvas.StampFactoryAppTemplate(db, "renamed-pr-trigger", "risk-score", 1))
	return canvas
}

func emitChecksStartupWebhook(t *testing.T, node *models.CanvasNode, payload map[string]any) {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	mac := hmac.New(sha256.New, []byte("test-secret"))
	mac.Write(body)
	headers := http.Header{}
	headers.Set("X-GitHub-Event", "pull_request")
	headers.Set("X-Hub-Signature-256", "sha256="+fmt.Sprintf("%x", mac.Sum(nil)))
	code, _, err := (&pulls.OnPullRequest{}).HandleWebhook(core.WebhookRequestContext{
		Body: body, Headers: headers, Logger: log.NewEntry(log.New()),
		Configuration: pulls.OnPullRequestConfiguration{Repository: "acme/app", Actions: []string{"opened", "synchronize", "reopened", "ready_for_review"}},
		Webhook:       &testcontexts.NodeWebhookContext{Secret: "test-secret"},
		Events:        contexts.NewEventContext(database.DB(t.Context()), node, nil, nil),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestPullRequestChecksRestartAfterLifecycleChange(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	_, order, pullRequest, canvas, request := checksStartupFixture(t, r)
	attachedAt := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	current := checksStartupGitHubPullRequest()
	current.UpdatedAt = &github.Timestamp{Time: attachedAt}
	useGitHub(t, current)
	events, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, request.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.NoError(t, db.Model(&events[0]).Update("state", models.CanvasEventStateRouted).Error)
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", events[0].RunID).Update("state", models.CanvasRunStateFinished).Error)
	node, err := models.FindCanvasNode(db, canvas.ID, "renamed-pr-trigger")
	require.NoError(t, err)

	emitChecksStartupWebhook(t, node, checksLifecyclePayload(t, pullRequest, order, current, "synchronize", "1111111111111111111111111111111111111111", attachedAt))
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID), "delayed synchronize for the attached head")

	emitChecksStartupWebhook(t, node, checksLifecyclePayload(t, pullRequest, order, current, "reopened", "", attachedAt))
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID), "reopened webhook for the attached snapshot")
	emitChecksStartupWebhook(t, node, checksLifecyclePayload(t, pullRequest, order, current, "reopened", "", attachedAt.Add(-time.Minute)))
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID), "older reopened webhook")
	emitChecksStartupWebhook(t, node, checksLifecyclePayload(t, pullRequest, order, current, "ready_for_review", "", attachedAt))
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID), "ready for review webhook for the attached snapshot")

	reopenedAt := attachedAt.Add(time.Minute)
	emitChecksStartupWebhook(t, node, checksLifecyclePayload(t, pullRequest, order, current, "reopened", "", reopenedAt))
	assert.EqualValues(t, 2, countCanvasEvents(t, canvas.ID))
	emitChecksStartupWebhook(t, node, checksLifecyclePayload(t, pullRequest, order, current, "reopened", "", reopenedAt))
	assert.EqualValues(t, 2, countCanvasEvents(t, canvas.ID), "repeated reopen delivery")

	returnedAt := reopenedAt.Add(time.Minute)
	returned := checksLifecyclePayload(t, pullRequest, order, current, "synchronize", "2222222222222222222222222222222222222222", returnedAt)
	emitChecksStartupWebhook(t, node, returned)
	assert.EqualValues(t, 3, countCanvasEvents(t, canvas.ID))
	emitChecksStartupWebhook(t, node, returned)
	assert.EqualValues(t, 3, countCanvasEvents(t, canvas.ID), "repeated push delivery")
}

func checksLifecyclePayload(t *testing.T, pullRequest *models.FactoryPullRequest, order *models.FactoryWorkOrder, current *github.PullRequest, action, before string, updatedAt time.Time) map[string]any {
	t.Helper()
	revision := *current
	revision.UpdatedAt = &github.Timestamp{Time: updatedAt}
	payload, err := pullRequestChecksEventData(pullRequest, order, &revision)
	require.NoError(t, err)
	payload["action"] = action
	if before != "" {
		payload["before"] = before
		payload["after"] = revision.GetHead().GetSHA()
	}
	return payload
}

func TestPullRequestChecksLegacyEventStaysDuplicate(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	_, order, pullRequest, canvas, request := checksStartupFixture(t, r)
	current := checksStartupGitHubPullRequest()
	current.UpdatedAt = &github.Timestamp{Time: time.Now().Add(-time.Hour).UTC().Truncate(time.Second)}
	useGitHub(t, current)
	events, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, request.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.NoError(t, db.Model(&events[0]).Update("state", models.CanvasEventStateRouted).Error)
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", events[0].RunID).Update("state", models.CanvasRunStateFinished).Error)
	require.NoError(t, db.Exec("UPDATE workflow_events SET dedup_key = NULL WHERE id = ?", events[0].ID).Error)
	node, err := models.FindCanvasNode(db, canvas.ID, "renamed-pr-trigger")
	require.NoError(t, err)
	payload, err := pullRequestChecksEventData(pullRequest, order, current)
	require.NoError(t, err)
	attachedAt := current.GetUpdatedAt().Time

	emitChecksStartupWebhook(t, node, checksLifecyclePayload(t, pullRequest, order, current, "reopened", "", attachedAt))
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID), "legacy reopened webhook for the attached snapshot")
	emitChecksStartupWebhook(t, node, payload)
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID), "legacy event with no dedup key")
	var restored models.CanvasEvent
	require.NoError(t, db.Where("id = ?", events[0].ID).First(&restored).Error)
	require.NotNil(t, restored.DedupKey)
	assert.Equal(t, "acme/app|42|"+current.GetHead().GetSHA(), *restored.DedupKey)

	require.NoError(t, db.Exec("UPDATE workflow_events SET dedup_key = NULL WHERE id = ?", events[0].ID).Error)
	applyChecksDedupBackfill(t, db)
	require.NoError(t, db.Where("id = ?", events[0].ID).First(&restored).Error)
	require.NotNil(t, restored.DedupKey)
	assert.Equal(t, "acme/app|42|"+current.GetHead().GetSHA(), *restored.DedupKey)
	emitChecksStartupWebhook(t, node, payload)
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID), "backfilled legacy event")
}

func applyChecksDedupBackfill(t *testing.T, db *gorm.DB) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "db", "migrations", "20261014121106_index-workflow-event-checks-dedup.up.sql")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	sqlText := string(raw)
	start := strings.Index(sqlText, "UPDATE ")
	require.NotEqual(t, -1, start)
	update := sqlText[start:]
	if end := strings.Index(update, "CREATE INDEX"); end >= 0 {
		update = update[:end]
	}
	require.NoError(t, db.Exec(strings.TrimSpace(update)).Error)
}

func TestPullRequestChecksStartupAttachmentEntries(t *testing.T) {
	for _, entry := range []string{"API", "Implement context"} {
		t.Run(entry, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory := seedHandOffFactory(t, r, db)
			canvas := createChecksStartupCanvas(t, r, factory, "acme/app")
			order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
			require.NoError(t, err)
			useGitHub(t, checksStartupGitHubPullRequest())
			if entry == "API" {
				_, err = CreateFactoryPullRequest(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryPullRequestRequest{FactoryId: factory.ID.String(), WorkOrderId: order.ID.String(), Url: "https://github.com/acme/app/pull/42"})
				require.NoError(t, err)
				assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID))
				return
			}
			var trackedID uuid.UUID
			err = db.Transaction(func(tx *gorm.DB) error {
				created, err := contexts.NewFactoryContext(tx, canvas, nil).AddPullRequest(core.AddPullRequestParams{OrderID: order.ID.String(), URL: "https://github.com/acme/app/pull/42"})
				if err != nil {
					return err
				}
				trackedID = uuid.MustParse(created.ID)
				pending, err := models.ListPullRequestChecksStartRequests(tx, trackedID)
				require.NoError(t, err)
				require.Len(t, pending, 1)
				outside, err := models.ListPullRequestChecksStartRequests(db, trackedID)
				require.NoError(t, err)
				require.Empty(t, outside, "startup must remain inside the caller's uncommitted transaction")
				return nil
			})
			require.NoError(t, err)
			assert.EqualValues(t, 0, countCanvasEvents(t, canvas.ID))
			StartPullRequestChecks(t.Context(), IntakeDependencies{}, trackedID)
			assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID))
		})
	}
}
