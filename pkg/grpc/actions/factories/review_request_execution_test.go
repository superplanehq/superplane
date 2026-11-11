package factories

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

// The fixture substitutes only GitHub responses and the external agent task.
// Both materialized graphs, output validation, and all action hooks are real.
type reviewRequestHTTP struct {
	requests        []*http.Request
	reviewerBodies  []map[string]any
	approved        bool
	alreadyAsked    bool
	empty           bool
	discoveryError  bool
	permissionError bool
	staleFinalRead  bool
	reads           int
}

func (h *reviewRequestHTTP) Do(request *http.Request) (*http.Response, error) {
	h.requests = append(h.requests, request)
	path := request.URL.Path
	if strings.HasSuffix(path, "/requested_reviewers") && request.Method == http.MethodPost {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			return nil, err
		}
		h.reviewerBodies = append(h.reviewerBodies, body)
		return publicationHTTPResponse(201, `{"number":42,"requested_reviewers":[{"login":"unrelated"},{"login":"octocat"}]}`), nil
	}
	if request.Method == http.MethodPatch && path == "/repos/acme/app/pulls/42" {
		return publicationHTTPResponse(200, `{}`), nil
	}
	if strings.HasSuffix(path, "/collaborators") {
		if h.discoveryError {
			return publicationHTTPResponse(503, `{"message":"Discovery unavailable"}`), nil
		}
		if h.empty {
			return publicationHTTPResponse(200, `[]`), nil
		}
		return publicationHTTPResponse(200, `[{"id":1,"login":"author","type":"User","permissions":{"push":true}},{"id":2,"login":"octocat","type":"User","permissions":{"push":true}},{"id":3,"login":"fallback-user","type":"User","permissions":{"admin":true}},{"id":4,"login":"agent[bot]","type":"Bot","permissions":{"push":true}},{"id":5,"login":"reader","type":"User","permissions":{"pull":true}}]`), nil
	}
	if strings.HasSuffix(path, "/permission") {
		if h.permissionError {
			return publicationHTTPResponse(403, `{"message":"Forbidden"}`), nil
		}
		return publicationHTTPResponse(200, `{"permission":"write","role_name":"write"}`), nil
	}
	if strings.HasSuffix(path, "/reviews") {
		if !h.approved {
			return publicationHTTPResponse(200, `[]`), nil
		}
		now := time.Now()
		body := fmt.Sprintf(`[{"id":1,"user":{"login":"octocat"},"state":"APPROVED","commit_id":"abc","submitted_at":%q},{"id":2,"user":{"login":"octocat"},"state":"COMMENTED","commit_id":"abc","submitted_at":%q}]`, now.Add(-time.Minute).Format(time.RFC3339), now.Format(time.RFC3339))
		return publicationHTTPResponse(200, body), nil
	}
	if request.Method == http.MethodGet && path == "/repos/acme/app/pulls/42" {
		h.reads++
		sha := "abc"
		if h.staleFinalRead && h.reads > 1 {
			sha = "new-head"
		}
		requested := `[{"login":"unrelated"}]`
		if h.alreadyAsked {
			requested = `[{"login":"unrelated"},{"login":"octocat"}]`
		}
		return publicationHTTPResponse(200, fmt.Sprintf(`{"number":42,"state":"open","body":"User description","head":{"sha":%q},"base":{"ref":"main"},"user":{"login":"author"},"requested_reviewers":%s}`, sha, requested)), nil
	}
	return nil, fmt.Errorf("unexpected GitHub request: %s %s", request.Method, path)
}

func reviewRequestIntegration() core.IntegrationContext {
	return &contexts.IntegrationContext{
		NewSetupFlow:      true,
		CurrentProperties: map[string]any{common.PropertyAuthMethod: common.AuthMethodPAT, common.PropertyOwner: "acme", common.PropertyOwnerType: common.OwnerTypeOrganization},
		CurrentSecrets:    map[string]core.IntegrationSecret{common.SecretPAT: {Name: common.SecretPAT, Value: []byte("test-token")}},
	}
}

func reviewRequestTestCanvas(t *testing.T, f publicationFixture, template string, params map[string]string) *models.Canvas {
	t.Helper()
	result, err := materializeFactoryTemplate(template, models.ProviderGitHub, factoryTemplateInput{appName: template, installParams: params})
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	nodes := make([]models.CanvasNode, 0, len(doc.Spec.Nodes))
	for _, node := range doc.Nodes() {
		nodes = append(nodes, models.CanvasNode{
			NodeID: node.ID, Name: node.Name, Type: node.Type,
			Ref: datatypes.NewJSONType(node.Ref), Configuration: datatypes.NewJSONType(node.Configuration), Metadata: datatypes.NewJSONType(node.Metadata),
		})
	}
	canvas, _ := support.CreateCanvas(t, f.canvas.OrganizationID, *f.canvas.CreatedBy, nodes, doc.Edges())
	require.NoError(t, f.db.Model(canvas).Update("factory_id", f.workspace.ID).Error)
	canvas.FactoryID = &f.workspace.ID
	return canvas
}

func reviewRequestTestExecution(t *testing.T, f publicationFixture, canvas *models.Canvas, run *models.CanvasRun, nodeID string, root, incoming *models.CanvasEvent, previous *models.CanvasNodeExecution) (*models.CanvasNodeExecution, core.Action) {
	t.Helper()
	node, err := models.FindCanvasNode(f.db, canvas.ID, nodeID)
	require.NoError(t, err)
	action, err := f.deps.Registry.GetAction(node.ComponentName())
	require.NoError(t, err)
	var previousID *uuid.UUID
	if previous != nil {
		previousID = &previous.ID
	}
	execution := support.CreateNextNodeExecution(t, canvas.ID, nodeID, root.ID, incoming.ID, previousID)
	configuration, err := workercontexts.NewNodeConfigurationBuilder(f.db, canvas.ID).
		WithRunID(run.ID).WithNodeID(nodeID).WithRootEvent(&root.ID).
		WithPreviousExecution(previousID).WithIncomingEventID(&incoming.ID).
		WithInput(map[string]any{incoming.NodeID: incoming.Data.Data()}).
		WithConfigurationFields(action.Configuration()).Build(node.Configuration.Data())
	require.NoError(t, err)
	execution.Configuration = datatypes.NewJSONType(configuration)
	execution.State = models.CanvasNodeExecutionStateStarted
	execution.Metadata = datatypes.NewJSONType(map[string]any{models.CanvasNodeExecutionFrozenComponentKey: node.ComponentName()})
	require.NoError(t, f.db.Save(execution).Error)
	return execution, action
}

func reviewRequestCompletedProducer(t *testing.T, host *reviewRequestHTTP, score float64) (publicationFixture, map[string]any) {
	t.Helper()
	f := newPublicationFixture(t)
	support.AssignOpenSystemTrial(t, f.canvas.OrganizationID)
	require.NoError(t, models.EnableExperimentalFeature(f.canvas.OrganizationID, features.FeatureOrganizationBYOK))
	integration, err := models.CreateIntegration(uuid.New(), f.canvas.OrganizationID, "claude", "claude", nil)
	require.NoError(t, err)
	require.NoError(t, f.db.Model(integration).Update("state", models.IntegrationStateReady).Error)
	integrationID := integration.ID.String()
	require.NoError(t, f.workspace.UpdateOnboarding(f.db, models.FactoryOnboardingPatch{AgentIntegrationID: &integrationID}))
	_, err = models.UpsertOrganizationBYOKModelAllowlist(f.db, f.canvas.OrganizationID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{"claude-sonnet-4-6"})
	require.NoError(t, err)
	permitted, err := models.ResolveSelectableLLMModels(f.db, f.canvas.OrganizationID, &f.workspace.ID, models.UsageProviderAnthropic, models.UsageFundingSourceBYOK)
	require.NoError(t, err)
	require.Contains(t, permitted, "claude-sonnet-4-6", "the fixture must configure an allowed agent before executing its graphs")
	f.canvas = reviewRequestTestCanvas(t, f, "risk-score", map[string]string{"appRepository": "acme/app", "enabledChecks": "security"})
	f.run, err = models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "on-pr-risk", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	root := &models.CanvasEvent{WorkflowID: f.canvas.ID, NodeID: "on-pr-risk", RunID: f.run.ID, State: models.CanvasEventStateRouted, Data: models.NewJSONValue(map[string]any{"data": map[string]any{
		"repository":   map[string]any{"full_name": "acme/app"},
		"pull_request": map[string]any{"number": 42, "head": map[string]any{"sha": "abc"}, "base": map[string]any{"ref": "main"}, "user": map[string]any{"login": "author"}},
		"workOrder":    map[string]any{"id": f.order.ID.String(), "title": f.order.Title},
	}})}
	require.NoError(t, f.db.Create(root).Error)
	_, err = f.pr.CreateActivity(f.db, models.FactoryPullRequestActivityParams{RunID: f.run.ID})
	require.NoError(t, err)
	_, err = factorycomponent.ReportMergeConfidenceCheck(f.db, f.order, f.run.ID, &factoryevents.AutomationRef{AppID: f.canvas.ID, NodeID: "assess-risk"}, "acme/app", "security", score, "Security assessment evidence.", []string{"security"}, nil)
	require.NoError(t, err)
	assessment, _ := reviewRequestTestExecution(t, f, f.canvas, f.run, "assess-risk", root, root, nil)
	require.NoError(t, workercontexts.NewExecutionStateContext(f.db, assessment, nil).Emit("passed", "runnerClaudeCode.finished", []any{map[string]any{"status": "succeeded"}}))
	var assessed models.CanvasEvent
	require.NoError(t, f.db.Where("execution_id = ?", assessment.ID).First(&assessed).Error)
	publication, action := reviewRequestTestExecution(t, f, f.canvas, f.run, "publish-merge-confidence", root, &assessed, assessment)
	state := workercontexts.NewExecutionStateContext(f.db, publication, nil)
	require.NoError(t, action.Execute(core.ExecutionContext{RunID: f.run.ID, Configuration: publication.Configuration.Data(), Metadata: workercontexts.NewExecutionMetadataContext(f.db, publication), Requests: workercontexts.NewExecutionRequestContext(f.db, publication)}))
	require.NoError(t, action.HandleHook(core.ActionHookContext{Name: prbody.HookName, Configuration: publication.Configuration.Data(), Integration: reviewRequestIntegration(), HTTP: host, Metadata: workercontexts.NewExecutionMetadataContext(f.db, publication), ExecutionState: state}))
	var events []models.CanvasEvent
	require.NoError(t, f.db.Where("execution_id = ?", publication.ID).Find(&events).Error)
	payloads := f.completePublication(t, publication, events)
	require.Len(t, payloads, 1)
	return f, map[string]any{
		"app":     map[string]any{"id": f.canvas.ID.String(), "name": f.canvas.Name},
		"node":    map[string]any{"id": "broadcast-merge-confidence", "name": "Broadcast Merge Confidence"},
		"payload": payloads[0],
	}
}

func TestReviewRequestFromPublishedReportToGitHub(t *testing.T) {
	for _, scenario := range []string{"favorable", "concerning", "concerning no review", "fallback", "no candidate", "malformed output", "reported lookup error", "agent failure", "author selected", "bot selected", "reader selected", "additional reviewer without reason", "additional reviewer", "discovery error", "permission error", "qualifying approval", "already requested", "stale before mutation", "paused after load"} {
		t.Run(scenario, func(t *testing.T) {
			host := &reviewRequestHTTP{}
			score := 2.0
			if scenario == "favorable" {
				score = 5
			}
			f, message := reviewRequestCompletedProducer(t, host, score)
			fallback := ""
			if scenario == "fallback" {
				fallback = "fallback-user"
			}
			consumer := reviewRequestTestCanvas(t, f, reviewRequestTemplateID, map[string]string{"mergeConfidenceAppId": f.canvas.ID.String(), "fallbackReviewer": fallback})
			trigger, err := models.FindCanvasNode(f.db, consumer.ID, reviewRequestTriggerNodeID)
			require.NoError(t, err)
			if scenario == "paused after load" {
				paused := datatypes.NewJSONType(map[string]any{"app": f.canvas.ID.String(), "enabled": false})
				require.NoError(t, f.db.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", consumer.ID, trigger.NodeID).Update("configuration", paused).Error)
			}
			var roots []models.CanvasEvent
			onBroadcast, err := f.deps.Registry.GetTrigger("onBroadcast")
			require.NoError(t, err)
			broadcastContext := core.AppMessageContext{Message: message, Events: workercontexts.NewEventContext(f.db, trigger, nil, func(events []models.CanvasEvent) { roots = append(roots, events...) })}
			require.NoError(t, onBroadcast.(core.AppTrigger).OnAppMessage(broadcastContext))
			require.NoError(t, onBroadcast.(core.AppTrigger).OnAppMessage(broadcastContext))
			if scenario == "paused after load" {
				count, err := models.CountCanvasEvents(f.db, consumer.ID, trigger.NodeID)
				require.NoError(t, err)
				require.Zero(t, count, "admission must use the current pause setting")
				return
			}
			require.Len(t, roots, 1, "the real publication must qualify for automatic selection")
			root, err := models.FindCanvasEventInTransaction(f.db, roots[0].ID)
			require.NoError(t, err)
			run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(f.db, root)
			require.NoError(t, err)
			require.NoError(t, models.MaybeLinkReviewRequestRun(f.db, consumer, root, run))
			activity, err := models.FindPullRequestActivityByRunID(f.db, run.ID)
			require.NoError(t, err)
			require.Equal(t, f.pr.ID, activity.PullRequestID)
			agent, agentAction := reviewRequestTestExecution(t, f, consumer, run, reviewRequestAgentNodeID, root, root, nil)
			host.discoveryError = scenario == "discovery error"
			host.empty = scenario == "no candidate"
			agentContext := core.ExecutionContext{RunID: run.ID, Configuration: agent.Configuration.Data(), Integration: reviewRequestIntegration(), HTTP: host}
			handled, err := workercontexts.NewFactoryContext(f.db, consumer, agent).PrepareReviewRequest(&agentContext)
			if host.discoveryError {
				require.ErrorContains(t, err, "Discover Review request reviewers")
				require.Empty(t, host.reviewerBodies)
				return
			}
			require.NoError(t, err)
			require.False(t, handled)
			require.NoError(t, agentAction.(core.NodeConfigurationValidator).ValidateNodeConfiguration(agentContext.Configuration.(map[string]any)), "the discovered context must remain a valid runner configuration")
			var reportContext string
			for _, raw := range agentContext.Configuration.(map[string]any)["environment"].([]any) {
				variable := raw.(map[string]any)
				if variable["name"] == "SUPERPLANE_REVIEW_REQUEST_REPORT" {
					reportContext = variable["value"].(string)
				}
			}
			immutable, err := json.Marshal(message["payload"].(map[string]any)["mergeConfidence"])
			require.NoError(t, err)
			require.JSONEq(t, string(immutable), reportContext)
			decision := map[string]any{"reviewRequired": true, "rationale": "Security findings need review.", "noCandidate": false, "reviewers": []any{map[string]any{"login": "octocat", "reason": "Owns the changed security code."}}}
			if scenario == "favorable" {
				decision["reviewRequired"], decision["rationale"], decision["reviewers"] = false, "All findings are favorable.", []any{}
			}
			if scenario == "concerning no review" {
				decision["reviewRequired"], decision["reviewers"] = false, []any{}
			}
			if scenario == "fallback" || scenario == "no candidate" {
				decision["noCandidate"], decision["reviewers"] = true, []any{}
			}
			if scenario == "malformed output" {
				decision = map[string]any{"reviewRequired": true}
			}
			if scenario == "reported lookup error" {
				decision["error"] = "GitHub contribution lookup failed."
			}
			for selection, login := range map[string]string{"author selected": "author", "bot selected": "agent[bot]", "reader selected": "reader"} {
				if scenario == selection {
					decision["reviewers"] = []any{map[string]any{"login": login, "reason": "Relevant history."}}
				}
			}
			if strings.HasPrefix(scenario, "additional reviewer") {
				reason := "Owns the separate authorization changes."
				if scenario == "additional reviewer without reason" {
					reason = ""
				}
				decision["reviewers"] = append(decision["reviewers"].([]any), map[string]any{"login": "fallback-user", "reason": reason})
			}
			result, err := json.Marshal(decision)
			require.NoError(t, err)
			taskState, exitCode := "succeeded", 0
			if scenario == "agent failure" {
				taskState, exitCode = "failed", 1
			}
			require.NoError(t, agentAction.HandleHook(core.ActionHookContext{Name: runner.HookPoll, Configuration: agentContext.Configuration, Parameters: map[string]any{"task_id": "review-selection"}, Metadata: workercontexts.NewExecutionMetadataContext(f.db, agent), ExecutionState: workercontexts.NewExecutionStateContext(f.db, agent, nil), RunnerTasks: &contexts.RunnerTaskContext{Task: &core.RunnerTask{ID: "review-selection", State: taskState, ExitCode: &exitCode, Result: result}}}))
			if slices.Contains([]string{"agent failure", "malformed output", "reported lookup error", "concerning no review", "author selected", "bot selected", "reader selected", "additional reviewer without reason"}, scenario) {
				require.NoError(t, f.db.First(agent, "id = ?", agent.ID).Error)
				require.Equal(t, models.CanvasNodeExecutionResultFailed, agent.Result)
				require.Empty(t, host.reviewerBodies)
				return
			}
			var selected models.CanvasEvent
			require.NoError(t, f.db.Where("execution_id = ?", agent.ID).First(&selected).Error)
			selection := selected.Data.Data().(map[string]any)["data"].(map[string]any)["reviewRequest"].(map[string]any)
			require.Equal(t, f.run.ID.String(), selection["reportId"])
			require.Equal(t, f.canvas.ID.String(), selection["sourceAppId"])
			require.NotEmpty(t, selection["rationale"])
			request, requestAction := reviewRequestTestExecution(t, f, consumer, run, "request-reviewers", root, &selected, agent)
			host.reads = 0
			host.approved = scenario == "qualifying approval"
			host.alreadyAsked = scenario == "already requested"
			host.permissionError = scenario == "permission error"
			host.staleFinalRead = scenario == "stale before mutation"
			requestContext := core.ExecutionContext{RunID: run.ID, Configuration: request.Configuration.Data(), Integration: reviewRequestIntegration(), HTTP: host, ExecutionState: workercontexts.NewExecutionStateContext(f.db, request, nil)}
			handled, err = workercontexts.NewFactoryContext(f.db, consumer, request).PrepareReviewRequest(&requestContext)
			if host.permissionError || host.staleFinalRead {
				require.Error(t, err)
				require.Empty(t, host.reviewerBodies)
				return
			}
			require.NoError(t, err)
			if !handled {
				require.NoError(t, requestAction.Execute(requestContext))
			}
			if scenario == "favorable" || scenario == "no candidate" || host.approved || host.alreadyAsked {
				require.True(t, handled)
				require.Empty(t, host.reviewerBodies)
				if scenario == "favorable" || scenario == "no candidate" {
					require.Equal(t, false, selection["reviewRequired"])
				}
			} else {
				require.Len(t, host.reviewerBodies, 1)
				expectedReviewers := []any{"octocat"}
				if scenario == "fallback" {
					expectedReviewers = []any{fallback}
				}
				if scenario == "additional reviewer" {
					expectedReviewers = append(expectedReviewers, "fallback-user")
				}
				require.Equal(t, expectedReviewers, host.reviewerBodies[0]["reviewers"])
				require.NotContains(t, host.reviewerBodies[0], "requested_reviewers", "the request must add selected reviewers without replacing unrelated requests")
			}
			for _, outbound := range host.requests {
				require.NotContains(t, outbound.URL.Path, "/statuses", "Phase 1 must publish no PR check")
				require.NotContains(t, outbound.URL.Path, "/check-runs")
				if outbound.Method == http.MethodPatch {
					body, err := io.ReadAll(outbound.Body)
					require.NoError(t, err)
					require.NotContains(t, string(body), "reviewers")
				}
			}
		})
	}
}
