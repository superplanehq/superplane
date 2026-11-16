package factories

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Use the materialized graph and real actions so a disconnected edge or an
// expression that reads the wrong execution cannot silently lose completion.
func completionFixture(t *testing.T, provider string) publicationFixture {
	t.Helper()
	f := newPublicationFixture(t)
	require.NoError(t, f.db.Model(f.pr).Update("provider", provider).Error)
	require.NoError(t, f.workspace.UpdateOnboarding(f.db, models.FactoryOnboardingPatch{VCSProvider: &provider}))
	integration, err := models.CreateIntegration(uuid.New(), f.canvas.OrganizationID, provider, provider, nil)
	require.NoError(t, err)
	result, err := materializeFactoryTemplate("risk-score", provider, factoryTemplateInput{
		appID: f.canvas.ID.String(), appName: "Merge confidence",
		installParams: map[string]string{"appRepository": "acme/app", "defaultBranch": "main"},
		integrations:  map[string]factoryTemplateIntegration{provider: {id: integration.ID.String(), name: provider}},
	})
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	f.publishVersion(t, func(version *models.CanvasVersion) {
		for _, node := range doc.Nodes() {
			if node.ID == "publish-merge-confidence" {
				node.ID = "publish"
				*findIntakeNode(version.Nodes, "publish") = node
				require.NoError(t, f.db.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", f.canvas.ID, "publish").Updates(map[string]any{
					"name": node.Name, "ref": datatypes.NewJSONType(node.Ref), "configuration": datatypes.NewJSONType(node.Configuration),
				}).Error)
			}
			if node.ID == "comment-merge-confidence" || node.ComponentName() == "filter" || node.ComponentName() == "broadcastMessage" {
				version.Nodes = append(version.Nodes, node)
				if node.ID == "comment-merge-confidence" {
					require.NoError(t, f.db.Create(&models.CanvasNode{WorkflowID: f.canvas.ID, NodeID: node.ID, Name: node.Name, Type: node.Type, AppInstallationID: &integration.ID, Ref: datatypes.NewJSONType(node.Ref), Configuration: datatypes.NewJSONType(node.Configuration)}).Error)
				}
			}
		}
		for _, edge := range doc.Edges() {
			// This fixture owns completion broadcasts. Status delivery has its
			// own execution tests and runs on a separate branch of the graph.
			if edge.SourceID == "on-pr-risk" || edge.SourceID == "assess-risk" || edge.SourceID == "start-assessment-check" || edge.TargetID == "finish-assessment-check" {
				continue
			}
			if edge.SourceID == "publish-merge-confidence" {
				edge.SourceID = "publish"
			}
			version.Edges = append(version.Edges, edge)
		}
	})
	require.NoError(t, f.db.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", f.canvas.ID, "publish").Update("app_installation_id", integration.ID).Error)
	f.startRun(t)
	return f
}

func installCompletionConnection(t *testing.T, host core.HTTPContext) {
	t.Helper()
	host = completionHTTP{host}
	installPublicationGitHub(t, host)
	previous := newPublicationConnection
	t.Cleanup(func() { newPublicationConnection = previous })
	newPublicationConnection = func(_ *gorm.DB, _ IntakeDependencies, _ *models.CanvasNode, integration *models.Integration) (core.IntegrationContext, core.HTTPContext) {
		if integration.AppName == models.ProviderGitHub {
			return gitHubPermissionTestIntegration(), host
		}
		return &contexts.IntegrationContext{Configuration: map[string]any{"token": "token"}, Metadata: bitbucket.Metadata{
			AuthType: bitbucket.AuthTypeWorkspaceAccessToken, Workspace: &bitbucket.WorkspaceMetadata{Slug: "acme"},
		}}, host
	}
}

// Commit-title enrichment is optional and has separate coverage. Keep the
// response queue here focused on publication and completion delivery.
type completionHTTP struct{ core.HTTPContext }

func (h completionHTTP) Do(request *http.Request) (*http.Response, error) {
	if strings.Contains(request.URL.Path, "/commits/") {
		return publicationHTTPResponse(http.StatusNotFound, `{}`), nil
	}
	return h.HTTPContext.Do(request)
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
			// Follow completion delivery only; status delivery is an independent
			// branch covered by the assessment and generic publication tests.
			if edge.TargetID == "finish-assessment-check" {
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
				ID: next.ID, NodeID: node.ID, WorkflowID: f.canvas.ID.String(),
				RunID: f.run.ID, Configuration: configuration,
				Publications: workercontexts.NewPublicationContext(f.db, f.canvas.OrganizationID, current, next),
				Metadata:     workercontexts.NewExecutionMetadataContext(f.db, next),
				Requests:     workercontexts.NewExecutionRequestContext(f.db, next),
				Expressions:  workercontexts.NewExpressionContext(builder),
				ExecutionState: workercontexts.NewExecutionStateContext(f.db, next, func(events []models.CanvasEvent) {
					emitted = append(emitted, events...)
				}),
				Apps: workercontexts.NewAppExecutionContext(f.db, f.canvas, current, next),
			})
			require.NoError(t, err)
			if isPublicationComponent(node.ComponentName()) {
				var request models.CanvasNodeRequest
				require.NoError(t, f.db.Where("execution_id = ? AND state = ?", next.ID, models.NodeExecutionRequestStatePending).First(&request).Error)
				emitted, err = ProcessBodySectionPublicationRequest(t.Context(), f.deps, request.ID, models.LockNodeRequestNow)
				require.NoError(t, err)
			}
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
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		for _, scenario := range []string{"published", "unchanged", "incomplete", "failed", "cancelled", "cancelled during unchanged read", "disabled", "closed", "stale", "superseded", "rejected", "invalid markers", "off", "off incomplete", "off failed", "off cancelled", "off closed", "off stale", "off superseded"} {
			t.Run(provider+"/"+scenario, func(t *testing.T) {
				f := completionFixture(t, provider)
				if strings.HasPrefix(scenario, "off") {
					f.publishVersion(t, func(version *models.CanvasVersion) {
						findIntakeNode(version.Nodes, "publish").Configuration["enabled"] = false
					})
					f.startRun(t)
				}
				if scenario != "incomplete" && scenario != "off incomplete" {
					f.report(t, f.run, 4)
				}
				channel := "passed"
				if scenario == "failed" || scenario == "off failed" {
					channel = "failed"
				}
				execution, request := f.publicationRequest(t, channel)
				description := "User text"
				if scenario == "unchanged" || scenario == "cancelled during unchanged read" {
					description = "<!-- superplane:merge-confidence:start -->\n" + execution.Configuration.Data()["body"].(string) + "\n<!-- superplane:merge-confidence:end -->"
				}
				if scenario == "invalid markers" {
					description = "<!-- superplane:merge-confidence:start -->"
				}
				read := descriptionResponse(t, provider, description)
				switch scenario {
				case "cancelled", "off cancelled":
					require.NoError(t, f.run.MarkAsCancelling(f.db, nil))
				case "disabled":
					f.publishVersion(t, func(version *models.CanvasVersion) {
						findIntakeNode(version.Nodes, "publish").Configuration["enabled"] = false
					})
				case "closed", "off closed":
					read = publicationHTTPResponse(200, `{"state":"closed","head":{"sha":"abc"}}`)
					if provider == models.ProviderBitbucket {
						read = publicationHTTPResponse(200, `{"state":"DECLINED","source":{"commit":{"hash":"abc"}}}`)
					}
				case "stale", "off stale":
					read = publicationHTTPResponse(200, `{"state":"open","head":{"sha":"new"}}`)
					if provider == models.ProviderBitbucket {
						read = publicationHTTPResponse(200, `{"state":"OPEN","source":{"commit":{"hash":"new"}}}`)
					}
				case "superseded", "off superseded":
					f.startRun(t)
				case "rejected":
					read = publicationHTTPResponse(403, `{"message":"Forbidden"}`)
				}
				host := &publicationHTTPContext{HTTPContext: contexts.HTTPContext{Responses: []*http.Response{read, publicationHTTPResponse(200, `{}`)}}, beforeRequest: func(request *http.Request) {
					if scenario == "cancelled during unchanged read" && request.Method == http.MethodGet {
						require.NoError(t, f.run.MarkAsCancelling(f.db, nil))
					}
				}}
				installCompletionConnection(t, host)
				events, err := ProcessBodySectionPublicationRequest(t.Context(), f.deps, request.ID, models.LockNodeRequestNow)
				require.NoError(t, err)
				require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
				expectedStatus := "skipped"
				switch scenario {
				case "published", "incomplete", "failed":
					expectedStatus = "published"
				case "disabled", "off", "off incomplete", "off failed":
					expectedStatus = "disabled"
				case "unchanged":
					expectedStatus = "unchanged"
				case "cancelled", "off cancelled":
					expectedStatus = "rejected"
				case "rejected":
					if provider == models.ProviderGitHub {
						expectedStatus = "failed"
					}
				}
				assert.Equal(t, expectedStatus, execution.Metadata.Data()["status"], "%#v", execution.Metadata.Data())
				// Continue the producer's graph even when another run superseded it.
				f.run, err = models.FindCanvasRunInTransaction(f.db, f.canvas.ID, execution.RunID)
				require.NoError(t, err)
				payloads := f.completePublication(t, execution, events)
				if scenario == "cancelled during unchanged read" {
					require.Len(t, host.Requests, 1)
				}
				if scenario == "published" || scenario == "unchanged" || scenario == "off" || scenario == "disabled" {
					require.Len(t, payloads, 1)
					assert.Equal(t, "merge-confidence.completed", payloads[0]["type"])
					assert.Equal(t, f.run.ID.String(), payloads[0]["reportId"])
					assert.Equal(t, f.pr.ID.String(), payloads[0]["pullRequestId"])
					assert.Equal(t, "acme/app", payloads[0]["repository"])
					assert.EqualValues(t, 42, payloads[0]["pullNumber"])
					assert.Equal(t, "abc", payloads[0]["reviewedCommit"])
					var message models.AppMessage
					require.NoError(t, f.db.Where("canvas_id = ?", f.canvas.ID).First(&message).Error)
					assert.Equal(t, "broadcast-merge-confidence", message.NodeID)
					require.NotNil(t, message.SourceRunID)
					assert.Equal(t, f.run.ID, *message.SourceRunID)
					assert.Equal(t, f.order.ID, f.pr.WorkOrderID)
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
				if scenario == "failed" || scenario == "off failed" {
					result, err := f.run.CalculateResult(f.db)
					require.NoError(t, err)
					assert.Equal(t, models.CanvasRunResultFailed, result)
				}
			})
		}
	}
}

func TestMergeConfidenceCompletionKeepsIdentityAndSnapshotOnRetry(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			f := completionFixture(t, provider)
			f.report(t, f.run, 4)
			execution, request := f.publicationRequest(t, "passed")
			body := execution.Configuration.Data()["body"].(string)
			host := &contexts.HTTPContext{Responses: []*http.Response{
				descriptionResponse(t, provider, "User text"),
				publicationHTTPResponse(503, `{"message":"unavailable"}`),
				descriptionResponse(t, provider, "<!-- superplane:merge-confidence:start -->\n"+body+"\n<!-- superplane:merge-confidence:end -->"),
			}}
			installCompletionConnection(t, host)
			events, err := ProcessBodySectionPublicationRequest(t.Context(), f.deps, request.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			assert.Empty(t, f.completePublication(t, execution, events))
			var retry models.CanvasNodeRequest
			require.NoError(t, f.db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).First(&retry).Error)
			// Another report changes the mutable latest results without superseding this activity.
			other, err := models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "trigger", models.CanvasRunStateStarted, "")
			require.NoError(t, err)
			f.report(t, other, 1)
			events, err = ProcessBodySectionPublicationRequest(t.Context(), f.deps, retry.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			payloads := f.completePublication(t, execution, events)
			require.Len(t, payloads, 1)
			first := payloads[0]
			assert.Equal(t, f.run.ID.String(), first["reportId"])
			report := first["mergeConfidence"].(map[string]any)
			assert.EqualValues(t, 4, report["checks"].([]any)[0].(map[string]any)["score"])
			require.Len(t, host.Requests, 3, "retry recognizes the same published report")
			events, err = ProcessBodySectionPublicationRequest(t.Context(), f.deps, retry.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			assert.Equal(t, payloads, f.completePublication(t, execution, events), "consumed requests do not emit again")

			f.startRun(t)
			f.report(t, f.run, 4)
			execution, request = f.publicationRequest(t, "passed")
			host.Responses = []*http.Response{descriptionResponse(t, provider, "<!-- superplane:merge-confidence:start -->\n"+body+"\n<!-- superplane:merge-confidence:end -->")}
			events, err = ProcessBodySectionPublicationRequest(t.Context(), f.deps, request.ID, models.LockNodeRequestNow)
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
		})
	}
}
