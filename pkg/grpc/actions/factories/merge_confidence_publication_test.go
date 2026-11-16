package factories

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
)

func TestMergeConfidenceDestinationsDeliverCompletionAndStartReviewRequest(t *testing.T) {
	for _, check := range []bool{false, true} {
		for name, settings := range map[string]reviewFindingsSettings{
			"body":    {publish: true, body: true},
			"comment": {publish: true, comment: true},
			"both":    {publish: true, body: true, comment: true},
			"off":     {body: true, comment: true},
		} {
			t.Run(fmt.Sprintf("check=%t/%s", check, name), func(t *testing.T) {
				host := &reviewRequestHTTP{description: "User description", comments: map[int64]string{99: "User comment"}}
				f, message := reviewRequestCompletedProducer(t, host, 2, map[string]any{
					"publishPRCheck": check, "publishResults": settings.publish, "resultsBody": settings.body, "resultsComment": settings.comment,
				})
				var comment models.CanvasNodeExecution
				require.NoError(t, f.db.Where("run_id = ? AND node_id = ?", f.run.ID, "comment-merge-confidence").First(&comment).Error)
				var incoming models.CanvasEvent
				require.NoError(t, f.db.Where("execution_id = ?", comment.ID).First(&incoming).Error)
				root, err := f.run.RootEvent(f.db)
				require.NoError(t, err)
				finish, action := reviewRequestTestExecution(t, f, f.canvas, f.run, "finish-assessment-check", root, &incoming, &comment)
				node, err := models.FindCanvasNode(f.db, f.canvas.ID, finish.NodeID)
				require.NoError(t, err)
				publications := workercontexts.NewPublicationContext(f.db, f.canvas.OrganizationID, node, finish)
				require.NoError(t, action.Execute(core.ExecutionContext{
					ID: finish.ID, NodeID: node.NodeID, WorkflowID: f.canvas.ID.String(), RunID: f.run.ID,
					Configuration: finish.Configuration.Data(), Publications: publications, Integration: reviewRequestIntegration(), HTTP: host,
					Metadata: workercontexts.NewExecutionMetadataContext(f.db, finish), Requests: workercontexts.NewExecutionRequestContext(f.db, finish), ExecutionState: workercontexts.NewExecutionStateContext(f.db, finish, nil),
				}))
				records, err := publications.List("github.review-check/")
				require.NoError(t, err)
				if check {
					require.Len(t, records, 1)
					var payload struct{ Configuration struct{ State string } }
					require.NoError(t, json.Unmarshal(records[0].Payload, &payload))
					require.Equal(t, "success", payload.Configuration.State, "complete assessments stay advisory with results off")
				} else {
					require.Empty(t, records)
				}
				require.Equal(t, settings.publish && settings.body, strings.Contains(host.description, "superplane:merge-confidence:start"))
				require.True(t, strings.HasPrefix(host.description, "User description"))
				require.Equal(t, "User comment", host.comments[99])
				wantComments := 1
				if settings.publish && settings.comment {
					wantComments++
				}
				require.Len(t, host.comments, wantComments)
				consumer := reviewFindingsConsumer(t, f, reviewFindingsSettings{}, "")
				outcome, agents, err := runReviewRequestGraph(t, f, consumer, deliverReviewRequestReport(t, f, consumer, message), host, "")
				require.NoError(t, err)
				require.Equal(t, 1, agents)
				require.Equal(t, "requested", outcome["reviewRequest"].(map[string]any)["status"])
				require.Len(t, host.reviewerBodies, 1)
			})
		}
	}
}

func TestMergeConfidenceCommentRetryKeepsAssessmentAndManagedComment(t *testing.T) {
	for _, mode := range []string{"retry", "disabled", "enabled again"} {
		t.Run(mode, func(t *testing.T) {
			host := &reviewRequestHTTP{}
			f, _ := reviewRequestCompletedProducer(t, host, 2, map[string]any{"publishResults": true, "resultsBody": false, "resultsComment": true})
			_, original := onlyComment(t, host)
			host.commentFails, host.commentStatus, host.commentApplied = 1, 502, true
			f, body := startMergeConfidencePublication(t, f, host, 3)
			var events []models.CanvasEvent
			require.NoError(t, f.db.Where("execution_id = ?", body.ID).Find(&events).Error)
			f.completePublication(t, body, events)
			var comment models.CanvasNodeExecution
			require.NoError(t, f.db.Where("run_id = ? AND node_id = ?", f.run.ID, "comment-merge-confidence").First(&comment).Error)
			require.Equal(t, "retrying", comment.Metadata.Data()["status"])
			id, updated := onlyComment(t, host)
			require.NotEqual(t, original, updated)
			require.Empty(t, host.bodyWrites)
			if mode != "retry" {
				save := func(enabled bool, generation int) {
					f.publishVersion(t, func(version *models.CanvasVersion) {
						config := findIntakeNode(version.Nodes, "comment-merge-confidence").Configuration
						config["enabled"], config["activationGeneration"] = enabled, generation
						require.NoError(t, f.db.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", f.canvas.ID, "comment-merge-confidence").Update("configuration", datatypes.NewJSONType(config)).Error)
					})
				}
				save(false, 1)
				if mode == "enabled again" {
					save(true, 2)
				}
			}
			writes := host.commentWrites
			processReviewRequestPublication(t, f, host, &comment)
			require.Equal(t, writes, host.commentWrites)
			require.NoError(t, f.db.First(&comment, "id = ?", comment.ID).Error)
			expected := map[string]string{"retry": "unchanged", "disabled": "disabled", "enabled again": "skipped"}
			require.Equal(t, expected[mode], comment.Metadata.Data()["status"])
			reconciledID, reconciled := onlyComment(t, host)
			require.Equal(t, id, reconciledID)
			require.Equal(t, updated, reconciled)
			require.NoError(t, f.db.Where("execution_id = ?", comment.ID).Find(&events).Error)
			payloads := f.completePublication(t, &comment, events)
			wantMessages := 2
			if mode == "enabled again" {
				wantMessages = 1
			}
			require.Len(t, payloads, wantMessages)
			var assessments int64
			require.NoError(t, f.db.Model(&models.CanvasNodeExecution{}).Where("run_id = ? AND node_id = ?", f.run.ID, "assess-risk").Count(&assessments).Error)
			require.EqualValues(t, 1, assessments, "publication retries keep the completed assessment")
		})
	}
}
