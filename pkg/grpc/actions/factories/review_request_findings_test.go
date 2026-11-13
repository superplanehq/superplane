package factories

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	canvasactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
)

type reviewFindingsSettings struct {
	publish, body, comment bool
}

func (s reviewFindingsSettings) agent() map[string]any {
	return map[string]any{"publishReviewFindings": s.publish, "reviewFindingsBody": s.body, "reviewFindingsComment": s.comment}
}

func reviewFindingsConsumer(t *testing.T, f publicationFixture, settings reviewFindingsSettings, fallback string) *models.Canvas {
	t.Helper()
	return reviewRequestTestCanvasWith(t, f, reviewRequestTemplateID, factoryTemplateInput{
		appName:       reviewRequestTemplateID,
		installParams: map[string]string{"mergeConfidenceAppId": f.canvas.ID.String(), "fallbackReviewer": fallback},
		agentSettings: map[string]*factoryAgentSettings{reviewRequestAgentNodeID: {component: "runnerClaudeCode", configuration: settings.agent()}},
	})
}

func deliverReviewRequestReport(t *testing.T, f publicationFixture, consumer *models.Canvas, message map[string]any) *models.CanvasEvent {
	t.Helper()
	trigger, err := models.FindCanvasNode(f.db, consumer.ID, reviewRequestTriggerNodeID)
	require.NoError(t, err)
	onBroadcast, err := f.deps.Registry.GetTrigger("onBroadcast")
	require.NoError(t, err)
	var emitted []models.CanvasEvent
	sourceRun := uuid.MustParse(message["payload"].(map[string]any)["reportId"].(string))
	events := workercontexts.NewEventContext(f.db, trigger, nil, func(events []models.CanvasEvent) { emitted = append(emitted, events...) }).WithSourceRun(&sourceRun)
	require.NoError(t, onBroadcast.(core.AppTrigger).OnAppMessage(core.AppMessageContext{Message: message, Configuration: trigger.Configuration.Data(), Events: events}))
	require.Len(t, emitted, 1)
	root, err := models.FindCanvasEventInTransaction(f.db, emitted[0].ID)
	require.NoError(t, err)
	return root
}

func reviewFindingsExecution(t *testing.T, f publicationFixture, root *models.CanvasEvent, nodeID string) *models.CanvasNodeExecution {
	t.Helper()
	var execution models.CanvasNodeExecution
	require.NoError(t, f.db.Where("root_event_id = ? AND node_id = ?", root.ID, nodeID).First(&execution).Error)
	return &execution
}

// reviewSection returns the managed review section of the description.
func reviewSection(description string) string {
	_, section, found := strings.Cut(description, "<!-- superplane:review-request:start -->")
	if !found {
		return ""
	}
	section, _, _ = strings.Cut(section, "<!-- superplane:review-request:end -->")
	return strings.TrimSpace(section)
}

func onlyComment(t *testing.T, host *reviewRequestHTTP) (int64, string) {
	t.Helper()
	require.Len(t, host.comments, 1, "each pull request keeps one managed comment for the automation")
	for id, body := range host.comments {
		return id, body
	}
	return 0, ""
}

func TestReviewRequestFindingsPublishToSelectedDestinations(t *testing.T) {
	for name, settings := range map[string]reviewFindingsSettings{
		"off":          {publish: false, body: true, comment: true},
		"body":         {publish: true, body: true},
		"comment":      {publish: true, comment: true},
		"both":         {publish: true, body: true, comment: true},
		"off and none": {},
	} {
		t.Run(name, func(t *testing.T) {
			host := &reviewRequestHTTP{}
			f, message := reviewRequestCompletedProducer(t, host, 2)
			mergeConfidence := host.description
			require.Contains(t, mergeConfidence, "superplane:merge-confidence:start")
			consumer := reviewFindingsConsumer(t, f, settings, "")
			root := deliverReviewRequestReport(t, f, consumer, message)
			outcome, agents, err := runReviewRequestGraph(t, f, consumer, root, host, "")
			require.NoError(t, err)
			require.Equal(t, 1, agents)
			require.Equal(t, "requested", outcome["reviewRequest"].(map[string]any)["status"])
			require.Len(t, host.reviewerBodies, 1, "reviewer requests do not depend on findings")

			publishBody, publishComment := settings.publish && settings.body, settings.publish && settings.comment
			section := reviewSection(host.description)
			if !publishBody {
				require.Empty(t, section)
				require.Equal(t, mergeConfidence, host.description)
			} else {
				require.True(t, strings.HasPrefix(host.description, "User description"), "user text is preserved")
				require.Contains(t, host.description, strings.TrimSpace(strings.TrimPrefix(mergeConfidence, "User description")), "the merge confidence section is preserved")
				require.Contains(t, section, "**Reviewers selected**")
				require.Contains(t, section, "| octocat | Owns the changed security code. |")
				require.Contains(t, section, "Assessed commit: abc")
				require.Contains(t, section, outcome["runId"].(string))
			}
			if !publishComment {
				require.Empty(t, host.comments)
			} else {
				_, comment := onlyComment(t, host)
				marker := fmt.Sprintf("<!-- superplane:comment:review-request:%s -->\n", consumer.ID)
				require.True(t, strings.HasPrefix(comment, marker))
				require.Contains(t, comment, "**Reviewers selected**")
				if publishBody {
					require.Equal(t, section, strings.TrimSpace(strings.TrimPrefix(comment, marker)), "both destinations publish the same decision")
				}
			}
			for node, enabled := range map[string]bool{reviewFindingsBodyNodeID: publishBody, reviewFindingsCommentNode: publishComment} {
				execution := reviewFindingsExecution(t, f, root, node)
				require.Equal(t, models.CanvasNodeExecutionResultPassed, execution.Result)
				expected := "skipped"
				if enabled {
					expected = "published"
				}
				require.Equal(t, expected, execution.Metadata.Data()["status"], node)
			}
		})
	}
}

// Findings and Publish PR check share the decision branches, so each decision
// is checked with the PR check off and on.
func TestReviewRequestFindingsExplainEachDecision(t *testing.T) {
	for _, publishCheck := range []bool{false, true} {
		for _, scenario := range []string{"concerning", "fallback", "favorable", "no candidate", "malformed output", "agent failure", "stale report"} {
			t.Run(fmt.Sprintf("publish_check=%t/%s", publishCheck, scenario), func(t *testing.T) {
				host := &reviewRequestHTTP{}
				score := 2.0
				if scenario == "favorable" {
					score = 5
				}
				f, message := reviewRequestCompletedProducer(t, host, score)
				fallback := ""
				if scenario == "fallback" || scenario == "malformed output" || scenario == "agent failure" {
					fallback = "fallback-user"
				}
				consumer := reviewFindingsConsumer(t, f, reviewFindingsSettings{publish: true, body: true, comment: true}, fallback)
				if publishCheck {
					setReviewRequestCheck(t, f, consumer, true, 2)
				}
				root := deliverReviewRequestReport(t, f, consumer, message)
				host.empty = scenario == "no candidate"
				host.movedHead = scenario == "stale report"
				_, _, err := runReviewRequestGraph(t, f, consumer, root, host, scenario)
				require.NoError(t, err)
				section := reviewSection(host.description)
				switch scenario {
				case "malformed output", "agent failure", "stale report":
					// Errors and stale reports must never read as a valid no-review decision.
					require.Empty(t, section)
					require.Empty(t, host.comments)
					return
				}
				_, comment := onlyComment(t, host)
				require.Contains(t, comment, section)
				require.Contains(t, section, "Assessed commit: abc")
				switch scenario {
				case "concerning":
					require.Contains(t, section, "**Reviewers selected**")
					require.Contains(t, section, "Security findings need review.")
					require.Contains(t, section, "Fallback reviewer: not used")
				case "fallback":
					require.Contains(t, section, "**Reviewers selected**")
					require.Contains(t, section, "| fallback-user | Use the configured fallback because selection found no suitable candidate. |")
					require.Contains(t, section, "Fallback reviewer: used because selection found no suitable candidate.")
				case "favorable":
					require.Contains(t, section, "**No review needed**")
					require.Contains(t, section, "All checks are positive.")
					require.NotContains(t, section, "No reviewer was requested.")
				case "no candidate":
					require.Contains(t, section, "**No eligible reviewer**")
					require.Contains(t, section, "No eligible candidate or fallback reviewer is available.")
					require.Contains(t, section, "No reviewer was requested.")
				}
				if publishCheck {
					require.Equal(t, models.CanvasNodeExecutionResultPassed, reviewFindingsExecution(t, f, root, "pr-check-pending").Result)
				} else {
					require.Empty(t, host.statuses)
				}
			})
		}
	}
}

func TestReviewRequestFindingsRetryReconcilesManagedComment(t *testing.T) {
	host := &reviewRequestHTTP{}
	f, message := reviewRequestCompletedProducer(t, host, 2)
	consumer := reviewFindingsConsumer(t, f, reviewFindingsSettings{publish: true, comment: true}, "")
	root := deliverReviewRequestReport(t, f, consumer, message)
	// GitHub creates the comment, but the response is lost.
	host.commentFails, host.commentStatus, host.commentApplied = 1, 502, true
	_, agents, err := runReviewRequestGraph(t, f, consumer, root, host, "")
	require.NoError(t, err)
	require.Equal(t, 1, agents)
	execution := reviewFindingsExecution(t, f, root, reviewFindingsCommentNode)
	require.Equal(t, "retrying", execution.Metadata.Data()["status"])
	require.Contains(t, execution.Metadata.Data()["lastError"], "pull request comment")
	id, created := onlyComment(t, host)

	processReviewRequestPublication(t, f, host, execution)
	require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
	require.Equal(t, models.CanvasNodeExecutionResultPassed, execution.Result)
	require.Equal(t, "unchanged", execution.Metadata.Data()["status"], "the retry finds the comment through its marker")
	require.EqualValues(t, id, execution.Metadata.Data()["commentId"])
	reconciledID, reconciled := onlyComment(t, host)
	require.Equal(t, id, reconciledID)
	require.Equal(t, created, reconciled)
	require.Len(t, host.reviewerBodies, 1, "publication retries do not request reviewers again")

	// A newer accepted decision updates the same comment in place.
	f, newer := emitReviewRequestReport(t, f, host, 3)
	newerRoot := deliverReviewRequestReport(t, f, consumer, newer)
	outcome, _, err := runReviewRequestGraph(t, f, consumer, newerRoot, host, "")
	require.NoError(t, err)
	updatedID, updated := onlyComment(t, host)
	require.Equal(t, id, updatedID)
	require.Contains(t, updated, outcome["runId"].(string))
	require.Equal(t, 2, host.commentWrites)
}

func TestReviewRequestFindingsShowPartialFailure(t *testing.T) {
	host := &reviewRequestHTTP{}
	f, message := reviewRequestCompletedProducer(t, host, 2)
	consumer := reviewFindingsConsumer(t, f, reviewFindingsSettings{publish: true, body: true, comment: true}, "")
	root := deliverReviewRequestReport(t, f, consumer, message)
	host.commentFails, host.commentStatus = 1, 403
	_, _, err := runReviewRequestGraph(t, f, consumer, root, host, "")
	require.NoError(t, err)
	comment := reviewFindingsExecution(t, f, root, reviewFindingsCommentNode)
	require.Equal(t, models.CanvasNodeExecutionResultFailed, comment.Result)
	require.Contains(t, comment.ResultMessage, "Could not publish to the pull request comment")
	body := reviewFindingsExecution(t, f, root, reviewFindingsBodyNodeID)
	require.Equal(t, models.CanvasNodeExecutionResultPassed, body.Result)
	require.Equal(t, "published", body.Metadata.Data()["status"])
	require.Contains(t, reviewSection(host.description), "**Reviewers selected**")
	require.Len(t, host.reviewerBodies, 1, "the saved reviewer request is kept")
	require.Empty(t, host.comments)
}

func TestReviewRequestFindingsKeepNewestSourceAndAttempt(t *testing.T) {
	host := &reviewRequestHTTP{}
	f, older := reviewRequestCompletedProducer(t, host, 2)
	consumer := reviewFindingsConsumer(t, f, reviewFindingsSettings{publish: true, body: true, comment: true}, "")
	olderRoot := deliverReviewRequestReport(t, f, consumer, older)
	f, newer := emitReviewRequestReport(t, f, host, 3)
	newerRoot := deliverReviewRequestReport(t, f, consumer, newer)

	newerOutcome, _, err := runReviewRequestGraph(t, f, consumer, newerRoot, host, "")
	require.NoError(t, err)
	newest := reviewSection(host.description)
	require.Contains(t, newest, newerOutcome["runId"].(string))
	_, newestComment := onlyComment(t, host)

	// The older report completes later on the same commit, with PR checks off.
	_, _, err = runReviewRequestGraph(t, f, consumer, olderRoot, host, "")
	require.NoError(t, err)
	for _, node := range []string{reviewFindingsBodyNodeID, reviewFindingsCommentNode} {
		execution := reviewFindingsExecution(t, f, olderRoot, node)
		require.Equal(t, "skipped", execution.Metadata.Data()["status"], node)
	}
	require.Equal(t, newest, reviewSection(host.description))
	_, comment := onlyComment(t, host)
	require.Equal(t, newestComment, comment)

	// A whole-run retry of the newer report is a newer attempt. An earlier
	// attempt that was accepted first cannot write after it.
	trigger, err := models.FindCanvasNode(f.db, consumer.ID, reviewRequestTriggerNodeID)
	require.NoError(t, err)
	firstRetry := retryReviewRequest(t, f, consumer, trigger, newerRoot)
	host.holdPublications = true
	_, _, err = runReviewRequestGraph(t, f, consumer, firstRetry, host, "")
	require.NoError(t, err)
	host.holdPublications = false
	held := reviewFindingsExecution(t, f, firstRetry, reviewFindingsBodyNodeID)
	retried := retryReviewRequest(t, f, consumer, trigger, newerRoot)
	retriedOutcome, _, err := runReviewRequestGraph(t, f, consumer, retried, host, "")
	require.NoError(t, err)
	latest := reviewSection(host.description)
	require.Contains(t, latest, retriedOutcome["runId"].(string))
	processReviewRequestPublication(t, f, host, held)
	require.NoError(t, f.db.First(held, "id = ?", held.ID).Error)
	require.Equal(t, "skipped", held.Metadata.Data()["status"])
	require.Equal(t, latest, reviewSection(host.description))
}

func TestReviewRequestFindingsSettingsChangesFencePendingWrites(t *testing.T) {
	for _, change := range []string{"disabled", "enabled again"} {
		t.Run(change, func(t *testing.T) {
			host := &reviewRequestHTTP{}
			f, message := reviewRequestCompletedProducer(t, host, 2)
			consumer := reviewFindingsConsumer(t, f, reviewFindingsSettings{publish: true, body: true}, "")
			root := deliverReviewRequestReport(t, f, consumer, message)
			host.holdPublications = true
			_, _, err := runReviewRequestGraph(t, f, consumer, root, host, "")
			require.NoError(t, err)
			host.holdPublications = false
			setReviewFindingsNode(t, f, consumer, false, 1)
			if change == "enabled again" {
				setReviewFindingsNode(t, f, consumer, true, 2)
			}
			execution := reviewFindingsExecution(t, f, root, reviewFindingsBodyNodeID)
			processReviewRequestPublication(t, f, host, execution)
			require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
			require.Equal(t, "skipped", execution.Metadata.Data()["status"], "obsolete attempts never write")
			require.Empty(t, reviewSection(host.description))
			if change == "disabled" {
				return
			}
			// Future attempts use the new activation.
			f, newer := emitReviewRequestReport(t, f, host, 3)
			outcome, _, err := runReviewRequestGraph(t, f, consumer, deliverReviewRequestReport(t, f, consumer, newer), host, "")
			require.NoError(t, err)
			require.Contains(t, reviewSection(host.description), outcome["runId"].(string))
		})
	}
}

func retryReviewRequest(t *testing.T, f publicationFixture, consumer *models.Canvas, trigger *models.CanvasNode, original *models.CanvasEvent) *models.CanvasEvent {
	t.Helper()
	retry, err := canvasactions.ReemitTriggerEvent(t.Context(), f.db, consumer, trigger.NodeID, original.ID)
	require.NoError(t, err)
	root, err := models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
	require.NoError(t, err)
	return root
}

// setReviewFindingsNode saves a new live version like the settings editor.
func setReviewFindingsNode(t *testing.T, f publicationFixture, consumer *models.Canvas, enabled bool, generation int) {
	t.Helper()
	node, err := models.FindCanvasNode(f.db, consumer.ID, reviewFindingsBodyNodeID)
	require.NoError(t, err)
	configuration := node.Configuration.Data()
	configuration["enabled"], configuration["activationGeneration"] = enabled, generation
	require.NoError(t, f.db.Model(node).Update("configuration", datatypes.NewJSONType(configuration)).Error)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(f.db, consumer)
	require.NoError(t, err)
	version.ID = uuid.New()
	findIntakeNode(version.Nodes, reviewFindingsBodyNodeID).Configuration = configuration
	require.NoError(t, f.db.Create(version).Error)
	require.NoError(t, f.db.Model(consumer).Update("live_version_id", version.ID).Error)
	consumer.LiveVersionID = &version.ID
}
