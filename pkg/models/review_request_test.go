package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type reviewRequestFixture struct {
	db       *gorm.DB
	consumer *models.Canvas
	trigger  *models.CanvasNode
	source   *models.Canvas
	run      *models.CanvasRun
	pr       *models.FactoryPullRequest
	message  map[string]any
}

// These cases protect the trust boundary between app-message delivery and an
// agent run. Component tests cannot establish source ownership or persistence.
func TestReviewRequestAdmitsOnlyPersistedSuccessfulCompletion(t *testing.T) {
	for _, scenario := range []string{"published", "unchanged", "other app", "other organization", "changed report", "missing broadcast", "publication disabled", "publication skipped", "failed source", "stale commit", "source replaced"} {
		t.Run(scenario, func(t *testing.T) {
			f := newReviewRequestFixture(t)
			switch scenario {
			case "unchanged", "publication skipped":
				status := "unchanged"
				if scenario == "publication skipped" {
					status = "skipped"
				}
				require.NoError(t, f.db.Model(&models.CanvasEvent{}).Where("run_id = ? AND node_id = ?", f.run.ID, "publish").Update("data", models.NewJSONValue(map[string]any{"type": "github.updatePullRequestBodySection.finished", "data": map[string]any{"status": status}})).Error)
			case "other app":
				f.message["app"].(map[string]any)["id"] = uuid.New().String()
			case "other organization":
				f.consumer.OrganizationID = uuid.New()
			case "changed report":
				f.message["payload"].(map[string]any)["mergeConfidence"].(map[string]any)["overallLevel"] = "positive"
			case "missing broadcast":
				require.NoError(t, f.db.Where("run_id = ? AND node_id = ?", f.run.ID, "broadcast").Delete(&models.CanvasEvent{}).Error)
			case "publication disabled":
				require.NoError(t, f.db.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", f.source.ID, "publish").Update("configuration", datatypes.NewJSONType(map[string]any{"enabled": false})).Error)
			case "failed source":
				require.NoError(t, f.db.Model(f.run).Update("result", models.CanvasRunResultFailed).Error)
			case "stale commit":
				_, err := f.pr.ObserveRevision(f.db, "next-head")
				require.NoError(t, err)
			case "source replaced":
				version, err := models.FindLiveCanvasVersionByCanvasInTransaction(f.db, f.source)
				require.NoError(t, err)
				next := *version
				next.ID = uuid.New()
				next.Nodes = append(datatypes.JSONSlice[models.Node]{}, version.Nodes...)
				next.Nodes[0].Metadata = models.FactoryAppTemplateMetadata("custom-verification", 1)
				require.NoError(t, f.db.Create(&next).Error)
				require.NoError(t, f.db.Model(f.source).Update("live_version_id", next.ID).Error)
			}
			source, err := models.ValidateReviewRequestSource(f.db, f.consumer, f.trigger, f.message)
			if scenario != "published" && scenario != "unchanged" {
				require.Error(t, err)
				assert.Nil(t, source)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, f.run.ID, source.Run.ID)
			assert.Equal(t, f.pr.ID, source.PullRequest.ID)
			assert.Equal(t, "caution", source.Report["overallLevel"])
		})
	}
}

func TestReviewRequestOrdersReportsBySourceRunAndAttemptsByRootEvent(t *testing.T) {
	f := newReviewRequestFixture(t)
	first := f.selection(t, time.Now().Add(-time.Hour))
	require.NoError(t, models.MaybeLinkReviewRequestRun(f.db, f.consumer, first, f.selectionRun(t, first)))
	_, err := models.CheckReviewRequestAttempt(f.db, f.consumer, f.selectionRun(t, first))
	require.NoError(t, err, "linking the consumer must not supersede its source")

	second := f.selection(t, time.Now())
	_, err = models.CheckReviewRequestAttempt(f.db, f.consumer, f.selectionRun(t, first))
	require.ErrorIs(t, err, models.ErrReviewRequestSuperseded)
	secondRun := f.selectionRun(t, second)
	consumerStarted := time.Now().Add(-2 * time.Hour)
	require.NoError(t, f.db.Model(secondRun).Update("created_at", consumerStarted).Error)
	_, err = models.CheckReviewRequestAttempt(f.db, f.consumer, secondRun)
	require.NoError(t, err, "consumer start order must not order attempts")

	oldRun := f.run
	oldMessage := cloneReviewRequestMessage(t, f.message)
	f.completeReport(t)
	_, err = models.ValidateReviewRequestSource(f.db, f.consumer, f.trigger, oldMessage)
	require.ErrorIs(t, err, models.ErrReviewRequestSuperseded, "a new source report supersedes the same SHA")
	assert.NotEqual(t, oldRun.ID, f.run.ID)
	_, err = models.CheckReviewRequestAttempt(f.db, f.consumer, secondRun)
	require.ErrorIs(t, err, models.ErrReviewRequestSuperseded)

	retry, err := canvases.ReemitTriggerEvent(t.Context(), f.db, f.consumer, f.trigger.NodeID, first.ID)
	require.NoError(t, err)
	retryID, err := uuid.Parse(retry.EventId)
	require.NoError(t, err)
	retried, err := models.FindCanvasEventForCanvas(f.db, f.consumer.ID, retryID)
	require.NoError(t, err)
	assert.NotEqual(t, first.ID, retried.ID)
	assert.NotEqual(t, first.RunID, retried.RunID)
	assert.Nil(t, retried.DedupKey)
	data := models.RootEventSourcePayload(retried.Data.Data()).(map[string]any)
	assert.Equal(t, f.run.ID.String(), data["payload"].(map[string]any)["reportId"])
	assert.Equal(t, f.run.ID.String(), data["reviewRequestSource"].(map[string]any)["reportId"])
	assert.Equal(t, "main", data["pull_request"].(map[string]any)["base"].(map[string]any)["ref"])
	_, err = models.ValidateReviewRequestSource(f.db, f.consumer, f.trigger, data)
	require.NoError(t, err)
}

func TestReviewRequestPauseBlocksRetryButPreservesInFlightAttempts(t *testing.T) {
	f := newReviewRequestFixture(t)
	root := f.selection(t, time.Now())
	configuration := datatypes.NewJSONType(map[string]any{"app": f.source.ID.String(), "enabled": false})
	require.NoError(t, f.db.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", f.consumer.ID, f.trigger.NodeID).Update("configuration", configuration).Error)
	require.Equal(t, true, f.trigger.Configuration.Data()["enabled"])
	_, err := models.ReviewRequestRetryEventData(f.db, f.consumer, f.trigger, root)
	require.ErrorIs(t, err, models.ErrReviewRequestSourceInvalid)
	_, err = models.CheckReviewRequestAttempt(f.db, f.consumer, f.selectionRun(t, root))
	require.NoError(t, err)
}

func TestReviewRequestRetryDoesNotReuseAnOlderPublishedReport(t *testing.T) {
	f := newReviewRequestFixture(t)
	original := f.selection(t, time.Now())
	newer, err := models.CreateCanvasRunInTransaction(f.db, f.source.ID, "source-trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	_, err = f.pr.CreateActivity(f.db, models.FactoryPullRequestActivityParams{RunID: newer.ID, Access: models.FactoryPullRequestAccessConcurrent})
	require.NoError(t, err)
	_, err = models.ReviewRequestRetryEventData(f.db, f.consumer, f.trigger, original)
	require.ErrorIs(t, err, models.ErrReviewRequestSourceInvalid)
}

func newReviewRequestFixture(t *testing.T) *reviewRequestFixture {
	t.Helper()
	r := support.Setup(t)
	db := database.DB(t.Context())
	workspace, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := workspace.CreateWorkOrder(db, "Review this change", "", &r.User, nil, nil)
	require.NoError(t, err)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{Repository: "acme/app", Number: 42, URL: "https://github.com/acme/app/pull/42", State: models.FactoryPullRequestStateOpen})
	require.NoError(t, err)
	node := func(id, component string, config map[string]any) models.CanvasNode {
		return models.CanvasNode{NodeID: id, Name: id, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: component}}), Configuration: datatypes.NewJSONType(config)}
	}
	source, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "source-trigger", Type: models.NodeTypeTrigger, Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1))},
		node("assess", "runnerClaudeCode", nil),
		node("publish", "github.updatePullRequestBodySection", map[string]any{"sectionKey": "merge-confidence", "enabled": true}),
		node("broadcast", "broadcastMessage", nil),
	}, nil)
	require.NoError(t, db.Model(source).Update("factory_id", workspace.ID).Error)
	source.FactoryID = &workspace.ID
	consumer, triggers := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "on-report", Type: models.NodeTypeTrigger,
		Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onBroadcast"}}),
		Configuration: datatypes.NewJSONType(map[string]any{"app": source.ID.String(), "enabled": true}),
		Metadata:      datatypes.NewJSONType(models.FactoryAppTemplateMetadata("review-request", 1)),
	}}, nil)
	require.NoError(t, db.Model(consumer).Update("factory_id", workspace.ID).Error)
	consumer.FactoryID = &workspace.ID
	f := &reviewRequestFixture{db: db, consumer: consumer, trigger: &triggers[0], source: source, pr: pr}
	f.completeReport(t)
	return f
}

func (f *reviewRequestFixture) completeReport(t *testing.T) {
	t.Helper()
	root := support.EmitCanvasEventForNodeWithData(t, f.source.ID, "source-trigger", "default", nil, map[string]any{"type": "github.pullRequest", "data": map[string]any{
		"pullRequest": map[string]any{"id": f.pr.ID.String()}, "pull_request": map[string]any{"head": map[string]any{"sha": "assessed-head"}, "base": map[string]any{"ref": "main"}},
	}})
	run, err := models.FindCanvasRunInTransaction(f.db, f.source.ID, root.RunID)
	require.NoError(t, err)
	_, err = f.pr.CreateActivity(f.db, models.FactoryPullRequestActivityParams{RunID: run.ID, Access: models.FactoryPullRequestAccessConcurrent, RevisionSHA: "assessed-head"})
	require.NoError(t, err)
	report := map[string]any{"pullRequestId": f.pr.ID.String(), "repository": f.pr.Repository, "pullNumber": f.pr.Number, "complete": true, "reviewedCommit": "assessed-head", "overallLevel": "caution", "checks": []any{map[string]any{"id": "security", "score": 3, "maxScore": 5, "summary": "Review access changes.", "level": "caution", "missing": false}}}
	payload := map[string]any{"type": "merge-confidence.completed", "reportId": run.ID.String(), "pullRequestId": f.pr.ID.String(), "repository": f.pr.Repository, "pullNumber": f.pr.Number, "reviewedCommit": "assessed-head", "mergeConfidence": report}
	for _, output := range []struct {
		node, channel, kind string
		data                map[string]any
	}{
		{"assess", "passed", "runnerClaudeCode.finished", map[string]any{"mergeConfidence": report}},
		{"publish", "default", "github.updatePullRequestBodySection.finished", map[string]any{"status": "published"}},
		{"broadcast", "default", "app.broadcast", payload},
	} {
		execution := support.CreateCanvasNodeExecution(t, f.source.ID, output.node, root.ID, root.ID)
		require.NoError(t, f.db.Model(execution).Updates(map[string]any{"run_id": run.ID, "configuration": datatypes.NewJSONType(map[string]any{"expectedHeadSha": "assessed-head"})}).Error)
		execution.RunID = run.ID
		_, err := execution.PassInTransaction(f.db, map[string][]any{output.channel: {map[string]any{"type": output.kind, "data": output.data}}})
		require.NoError(t, err)
	}
	f.run = run
	f.message = cloneReviewRequestMessage(t, map[string]any{"app": map[string]any{"id": f.source.ID.String(), "name": "Merge confidence"}, "node": map[string]any{"id": "broadcast"}, "payload": payload})
}

func (f *reviewRequestFixture) selection(t *testing.T, createdAt time.Time) *models.CanvasEvent {
	t.Helper()
	event := support.EmitCanvasEventForNodeWithData(t, f.consumer.ID, f.trigger.NodeID, "default", nil, map[string]any{"type": "app.broadcast", "data": f.message})
	require.NoError(t, f.db.Model(event).Update("created_at", createdAt).Error)
	return event
}

func (f *reviewRequestFixture) selectionRun(t *testing.T, event *models.CanvasEvent) *models.CanvasRun {
	t.Helper()
	run, err := models.FindCanvasRunInTransaction(f.db, f.consumer.ID, event.RunID)
	require.NoError(t, err)
	return run
}

func cloneReviewRequestMessage(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	var cloned map[string]any
	require.NoError(t, json.Unmarshal(raw, &cloned))
	return cloned
}
