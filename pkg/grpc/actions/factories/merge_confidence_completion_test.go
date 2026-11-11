package factories

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

// Use the materialized graph and real actions so a disconnected edge or an
// expression that reads the wrong execution cannot silently lose completion.
func completionFixture(t *testing.T) publicationFixture {
	t.Helper()
	f := newPublicationFixture(t)
	result, err := materializeFactoryTemplate("risk-score", models.ProviderGitHub, factoryTemplateInput{
		appID: f.canvas.ID.String(), appName: "Merge confidence",
		installParams: map[string]string{"appRepository": "acme/app", "defaultBranch": "main"},
	})
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	f.publishVersion(t, func(version *models.CanvasVersion) {
		for _, node := range doc.Nodes() {
			if node.ComponentName() == "filter" || node.ComponentName() == "broadcastMessage" {
				version.Nodes = append(version.Nodes, node)
			}
		}
		for _, edge := range doc.Edges() {
			if edge.SourceID == "on-pr-risk" || edge.SourceID == "assess-risk" {
				continue
			}
			if edge.SourceID == "publish-merge-confidence" {
				edge.SourceID = "publish"
			}
			version.Edges = append(version.Edges, edge)
		}
	})
	f.startRun(t)
	return f
}

func (f publicationFixture) completePublication(t *testing.T, execution *models.CanvasNodeExecution, events []models.CanvasEvent) []map[string]any {
	t.Helper()
	version, err := models.FindCanvasVersionInTransaction(f.db, f.canvas.ID, f.run.VersionID)
	require.NoError(t, err)
	for _, event := range events {
		// The executor reloads persisted input; emitted events still hold raw JSON.
		incoming, err := models.FindCanvasEventInTransaction(f.db, event.ID)
		require.NoError(t, err)
		for _, edge := range version.Edges {
			if edge.SourceID != execution.NodeID || edge.Channel != event.Channel {
				continue
			}
			node := findIntakeNode(version.Nodes, edge.TargetID)
			require.NotNil(t, node)
			action, err := f.deps.Registry.GetAction(node.ComponentName())
			require.NoError(t, err)
			next := support.CreateNextNodeExecution(t, f.canvas.ID, node.ID, execution.RootEventID, event.ID, &execution.ID)
			builder := workercontexts.NewNodeConfigurationBuilder(f.db, f.canvas.ID).
				WithRunID(f.run.ID).WithNodeID(node.ID).WithRootEvent(&execution.RootEventID).
				WithPreviousExecution(&execution.ID).WithIncomingEventID(&event.ID).WithInput(map[string]any{event.NodeID: incoming.Data.Data()}).
				WithConfigurationFields(action.Configuration())
			configuration, err := builder.Build(node.Configuration)
			require.NoError(t, err)
			next.Configuration = datatypes.NewJSONType(configuration)
			require.NoError(t, f.db.Save(next).Error)
			current, err := models.FindUnscopedCanvasNode(f.db, f.canvas.ID, node.ID)
			require.NoError(t, err)
			var emitted []models.CanvasEvent
			err = action.Execute(core.ExecutionContext{
				RunID: f.run.ID, Configuration: configuration,
				Expressions: workercontexts.NewExpressionContext(builder),
				ExecutionState: workercontexts.NewExecutionStateContext(f.db, next, func(events []models.CanvasEvent) {
					emitted = append(emitted, events...)
				}),
				Apps: workercontexts.NewAppExecutionContext(f.db, f.canvas, current, next),
			})
			require.NoError(t, err)
			f.completePublication(t, next, emitted)
		}
	}
	var messages []models.AppMessage
	require.NoError(t, f.db.Where("canvas_id = ?", f.canvas.ID).Find(&messages).Error)
	payloads := make([]map[string]any, len(messages))
	for i, message := range messages {
		require.NoError(t, json.Unmarshal(message.Payload, &payloads[i]))
	}
	return payloads
}

func TestMergeConfidenceCompletionPublicationOutcomes(t *testing.T) {
	for _, scenario := range []string{"published", "unchanged", "incomplete", "failed", "cancelled", "disabled", "closed", "stale", "superseded", "rejected", "invalid markers"} {
		t.Run(scenario, func(t *testing.T) {
			f := completionFixture(t)
			if scenario != "incomplete" {
				f.report(t, f.run, 4)
			}
			channel := "passed"
			if scenario == "failed" {
				channel = "failed"
			}
			execution, request := f.publicationRequest(t, channel)
			description := "User text"
			if scenario == "unchanged" {
				description = "<!-- superplane:merge-confidence:start -->\n" + execution.Configuration.Data()["body"].(string) + "\n<!-- superplane:merge-confidence:end -->"
			}
			if scenario == "invalid markers" {
				description = "<!-- superplane:merge-confidence:start -->"
			}
			read := descriptionResponse(t, models.ProviderGitHub, description)
			switch scenario {
			case "cancelled":
				require.NoError(t, f.run.MarkAsCancelling(f.db, nil))
			case "disabled":
				f.publishVersion(t, func(version *models.CanvasVersion) {
					findIntakeNode(version.Nodes, "publish").Configuration["enabled"] = false
				})
			case "closed":
				read = publicationHTTPResponse(200, `{"state":"closed","head":{"sha":"abc"}}`)
			case "stale":
				read = publicationHTTPResponse(200, `{"state":"open","head":{"sha":"new"}}`)
			case "superseded":
				f.startRun(t)
			case "rejected":
				read = publicationHTTPResponse(403, `{"message":"Forbidden"}`)
			}
			host := &contexts.HTTPContext{Responses: []*http.Response{read, publicationHTTPResponse(200, `{}`)}}
			installPublicationGitHub(t, host)
			events, err := ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
			expectedStatus := "skipped"
			switch scenario {
			case "published", "incomplete", "failed":
				expectedStatus = "published"
			case "unchanged":
				expectedStatus = "unchanged"
			case "cancelled":
				expectedStatus = "rejected"
			}
			assert.Equal(t, expectedStatus, execution.Metadata.Data()["status"])
			// Continue the producer's graph even when another run superseded it.
			f.run, err = models.FindCanvasRunInTransaction(f.db, f.canvas.ID, execution.RunID)
			require.NoError(t, err)
			payloads := f.completePublication(t, execution, events)
			if scenario == "published" || scenario == "unchanged" {
				require.Len(t, payloads, 1)
				assert.Equal(t, "merge-confidence.completed", payloads[0]["type"])
				assert.Equal(t, f.run.ID.String(), payloads[0]["reportId"])
				assert.Equal(t, f.pr.ID.String(), payloads[0]["pullRequestId"])
				assert.Equal(t, "acme/app", payloads[0]["repository"])
				assert.EqualValues(t, 42, payloads[0]["pullNumber"])
				assert.Equal(t, "abc", payloads[0]["reviewedCommit"])
				var assessment models.CanvasEvent
				require.NoError(t, f.db.Where("run_id = ? AND node_id = ? AND execution_id IS NOT NULL", f.run.ID, "agent").First(&assessment).Error)
				savedReport, err := json.Marshal(assessment.Data.Data().(map[string]any)["data"].(map[string]any)["mergeConfidence"])
				require.NoError(t, err)
				emittedReport, err := json.Marshal(payloads[0]["mergeConfidence"])
				require.NoError(t, err)
				assert.JSONEq(t, string(savedReport), string(emittedReport))
				return
			}
			assert.Empty(t, payloads)
			if scenario == "failed" || scenario == "incomplete" {
				require.Len(t, host.Requests, 2, "incomplete reports still reach the PR")
				assert.Contains(t, execution.Configuration.Data()["body"], "**Incomplete**")
			}
			if scenario == "failed" {
				result, err := f.run.CalculateResult(f.db)
				require.NoError(t, err)
				assert.Equal(t, models.CanvasRunResultFailed, result)
			}
		})
	}
}

func TestMergeConfidenceCompletionKeepsIdentityAndSnapshotOnRetry(t *testing.T) {
	f := completionFixture(t)
	f.report(t, f.run, 4)
	execution, request := f.publicationRequest(t, "passed")
	body := execution.Configuration.Data()["body"].(string)
	host := &contexts.HTTPContext{Responses: []*http.Response{
		descriptionResponse(t, models.ProviderGitHub, "User text"),
		publicationHTTPResponse(503, `{"message":"unavailable"}`),
		descriptionResponse(t, models.ProviderGitHub, "<!-- superplane:merge-confidence:start -->\n"+body+"\n<!-- superplane:merge-confidence:end -->"),
	}}
	installPublicationGitHub(t, host)
	events, err := ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
	require.NoError(t, err)
	assert.Empty(t, f.completePublication(t, execution, events))
	var retry models.CanvasNodeRequest
	require.NoError(t, f.db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).First(&retry).Error)
	// Another report changes the mutable latest results without superseding this activity.
	other, err := models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	f.report(t, other, 1)
	events, err = ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, retry.ID, models.LockNodeRequestNow)
	require.NoError(t, err)
	payloads := f.completePublication(t, execution, events)
	require.Len(t, payloads, 1)
	first := payloads[0]
	assert.Equal(t, f.run.ID.String(), first["reportId"])
	report := first["mergeConfidence"].(map[string]any)
	assert.EqualValues(t, 4, report["checks"].([]any)[0].(map[string]any)["score"])
	require.Len(t, host.Requests, 3, "retry recognizes the same published report")
	events, err = ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, retry.ID, models.LockNodeRequestNow)
	require.NoError(t, err)
	assert.Equal(t, payloads, f.completePublication(t, execution, events), "consumed requests do not emit again")

	f.startRun(t)
	f.report(t, f.run, 4)
	execution, request = f.publicationRequest(t, "passed")
	host.Responses = []*http.Response{descriptionResponse(t, models.ProviderGitHub, "<!-- superplane:merge-confidence:start -->\n"+body+"\n<!-- superplane:merge-confidence:end -->")}
	events, err = ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
	require.NoError(t, err)
	payloads = f.completePublication(t, execution, events)
	require.Len(t, payloads, 2)
	var second map[string]any
	for _, payload := range payloads {
		if payload["reportId"] == first["reportId"] {
			assert.Equal(t, first, payload)
			continue
		}
		second = payload
		assert.Equal(t, f.run.ID.String(), payload["reportId"])
		assert.Equal(t, first["reviewedCommit"], payload["reviewedCommit"])
		assert.Equal(t, first["mergeConfidence"], payload["mergeConfidence"])
	}
	require.NotNil(t, second, "a new same-SHA run must emit a new report identity")
}
