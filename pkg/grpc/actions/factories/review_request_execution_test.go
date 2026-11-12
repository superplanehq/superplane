package factories

import (
	"encoding/json"
	"fmt"
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
	canvasactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
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
// Both materialized graphs, expressions, component execution, and runner hooks are real.
type reviewRequestHTTP struct {
	requests          []*http.Request
	reviewerBodies    []map[string]any
	approved          bool
	oldApproval       bool
	withdrawnApproval bool
	requestError      bool
	alreadyAsked      bool
	empty             bool
	discoveryError    bool
	permissionError   bool
	movedHead         bool
	closed            bool
	lostPermission    bool
	changedIdentity   bool
	// pushDuringLookups moves the head after the permission lookup.
	pushDuringLookups bool
}

func (h *reviewRequestHTTP) Do(request *http.Request) (*http.Response, error) {
	h.requests = append(h.requests, request)
	path := request.URL.Path
	if strings.HasSuffix(path, "/requested_reviewers") && request.Method == http.MethodPost {
		if h.requestError {
			return publicationHTTPResponse(403, `{"message":"review request denied"}`), nil
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			return nil, err
		}
		h.reviewerBodies = append(h.reviewerBodies, body)
		return publicationHTTPResponse(201, `{"number":42,"requested_reviewers":[{"login":"unrelated"},{"id":2,"login":"octocat"}]}`), nil
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
		if h.changedIdentity {
			return publicationHTTPResponse(200, `[{"id":99,"login":"octocat","type":"User","permissions":{"push":true}}]`), nil
		}
		return publicationHTTPResponse(200, `[{"id":1,"login":"author","type":"User","permissions":{"push":true}},{"id":2,"login":"octocat","type":"User","permissions":{"push":true}},{"id":3,"login":"fallback-user","type":"User","permissions":{"admin":true}},{"id":4,"login":"agent[bot]","type":"Bot","permissions":{"push":true}},{"id":5,"login":"reader","type":"User","permissions":{"pull":true}}]`), nil
	}
	if strings.HasSuffix(path, "/permission") {
		if h.permissionError {
			return publicationHTTPResponse(403, `{"message":"Forbidden"}`), nil
		}
		if h.lostPermission {
			return publicationHTTPResponse(200, `{"permission":"read"}`), nil
		}
		h.movedHead = h.movedHead || h.pushDuringLookups
		return publicationHTTPResponse(200, `{"permission":"write","role_name":"write"}`), nil
	}
	if strings.HasSuffix(path, "/reviews") {
		if !h.approved {
			return publicationHTTPResponse(200, `[]`), nil
		}
		now := time.Now()
		commit := "abc"
		if h.oldApproval {
			commit = "previous-commit"
		}
		body := fmt.Sprintf(`[{"id":1,"user":{"id":2,"login":"octocat"},"state":"APPROVED","commit_id":%q,"submitted_at":%q},{"id":2,"user":{"id":2,"login":"octocat"},"state":"COMMENTED","commit_id":"abc","submitted_at":%q}]`, commit, now.Add(-time.Minute).Format(time.RFC3339), now.Format(time.RFC3339))
		if h.withdrawnApproval {
			body = strings.TrimSuffix(body, "]") + fmt.Sprintf(`,{"id":3,"user":{"id":2,"login":"octocat"},"state":"CHANGES_REQUESTED","commit_id":"abc","submitted_at":%q}]`, now.Add(time.Minute).Format(time.RFC3339))
		}
		return publicationHTTPResponse(200, body), nil
	}
	if request.Method == http.MethodGet && path == "/repos/acme/app/pulls/42" {
		sha := "abc"
		if h.movedHead {
			sha = "new-head"
		}
		requested := `[{"login":"unrelated"}]`
		if h.alreadyAsked {
			requested = `[{"login":"unrelated"},{"id":2,"login":"octocat"}]`
		}
		state := "open"
		if h.closed {
			state = "closed"
		}
		return publicationHTTPResponse(200, fmt.Sprintf(`{"number":42,"state":%q,"body":"User description","head":{"sha":%q},"base":{"ref":"main","sha":"base-sha"},"user":{"login":"author"},"requested_reviewers":%s}`, state, sha, requested)), nil
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
	vcs, err := models.CreateIntegration(uuid.New(), f.canvas.OrganizationID, "github", "github", nil)
	require.NoError(t, err)
	require.NoError(t, f.db.Model(vcs).Update("state", models.IntegrationStateReady).Error)
	vcsID := vcs.ID.String()
	require.NoError(t, f.workspace.UpdateOnboarding(f.db, models.FactoryOnboardingPatch{AgentIntegrationID: &integrationID, VCSIntegrationID: &vcsID}))
	_, err = models.UpsertOrganizationBYOKModelAllowlist(f.db, f.canvas.OrganizationID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{"claude-sonnet-4-6"})
	require.NoError(t, err)
	permitted, err := models.ResolveSelectableLLMModels(f.db, f.canvas.OrganizationID, &f.workspace.ID, models.UsageProviderAnthropic, models.UsageFundingSourceBYOK)
	require.NoError(t, err)
	require.Contains(t, permitted, "claude-sonnet-4-6", "the fixture must configure an allowed agent before executing its graphs")
	f.canvas = reviewRequestTestCanvas(t, f, "risk-score", map[string]string{"appRepository": "acme/app", "enabledChecks": "security"})
	return emitReviewRequestReport(t, f, host, score)
}

func emitReviewRequestReport(t *testing.T, f publicationFixture, host *reviewRequestHTTP, score float64) (publicationFixture, map[string]any) {
	t.Helper()
	var err error

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
	payloads := slices.DeleteFunc(f.completePublication(t, publication, events), func(payload map[string]any) bool { return payload["reportId"] != f.run.ID.String() })
	require.Len(t, payloads, 1)
	return f, map[string]any{
		"app":     map[string]any{"id": f.canvas.ID.String(), "name": f.canvas.Name},
		"node":    map[string]any{"id": "broadcast-merge-confidence", "name": "Broadcast Merge Confidence"},
		"payload": payloads[0],
	}
}

// Follow the published graph using persisted events and ordinary configuration
// resolution. Only the remote agent task and GitHub HTTP responses are replaced.
func runReviewRequestGraph(t *testing.T, f publicationFixture, canvas *models.Canvas, root *models.CanvasEvent, host *reviewRequestHTTP, scenario string) (map[string]any, int, error) {
	t.Helper()
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(f.db, root)
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(f.db, canvas)
	require.NoError(t, err)
	agentRuns := 0
	var visit func(*models.CanvasEvent, *models.CanvasNodeExecution) error
	visit = func(incoming *models.CanvasEvent, previous *models.CanvasNodeExecution) error {
		for _, edge := range version.Edges {
			if edge.SourceID != incoming.NodeID || edge.Channel != incoming.Channel {
				continue
			}
			execution, action := reviewRequestTestExecution(t, f, canvas, run, edge.TargetID, root, incoming, previous)
			node, err := models.FindCanvasNode(f.db, canvas.ID, execution.NodeID)
			require.NoError(t, err)
			var previousID *uuid.UUID
			if previous != nil {
				previousID = &previous.ID
			}
			builder := workercontexts.NewNodeConfigurationBuilder(f.db, canvas.ID).WithRunID(run.ID).WithNodeID(node.NodeID).WithRootEvent(&root.ID).WithPreviousExecution(previousID).WithIncomingEventID(&incoming.ID).WithInput(map[string]any{incoming.NodeID: incoming.Data.Data()})
			ctx := core.ExecutionContext{
				RunID: run.ID, Configuration: execution.Configuration.Data(), HTTP: host, Integration: reviewRequestIntegration(),
				Expressions:    workercontexts.NewExpressionContext(builder),
				Metadata:       workercontexts.NewExecutionMetadataContext(f.db, execution),
				ExecutionState: workercontexts.NewExecutionStateContext(f.db, execution, nil),
				Factory:        workercontexts.NewFactoryContext(f.db, canvas, execution),
				Apps:           workercontexts.NewAppExecutionContext(f.db, canvas, node, execution),
				Runs:           workercontexts.NewRunExecutionContext(f.db, canvas, node, execution),
				Requests:       &contexts.RequestContext{},
			}
			if node.NodeID == reviewRequestAgentNodeID {
				agentRuns++
				steps := execution.Configuration.Data()["steps"].([]any)
				prompt := steps[1].(map[string]any)["prompt"].(string)
				require.Contains(t, prompt, "Pull request author: author")
				environment := execution.Configuration.Data()["environment"].([]any)
				for _, raw := range environment {
					variable := raw.(map[string]any)
					if variable["name"] == "SUPERPLANE_REVIEW_REQUEST_REPORT" {
						encoded, err := json.Marshal(root.Data.Data().(map[string]any)["data"].(map[string]any)["payload"].(map[string]any)["mergeConfidence"])
						require.NoError(t, err)
						require.JSONEq(t, string(encoded), variable["value"].(string))
					}
				}
				if scenario == "hosted agent" {
					tasks := &contexts.RunnerTaskContext{}
					ctx.RunnerTasks = tasks
					ctx.Secrets = &contexts.SecretsContext{IntegrationKeys: map[string]map[string][]byte{"github": {}}}
					ctx.Webhook = &contexts.NodeWebhookContext{}
					ctx.HostedLLM = &contexts.HostedLLMContext{Default: core.DefaultHostedLLMModel{Provider: models.UsageProviderAnthropic, Model: "claude-sonnet-4-6"}, Access: core.HostedLLMAccess{APIKey: "test-key", AllowedModels: []string{"claude-sonnet-4-6"}}}
					require.NoError(t, action.Execute(ctx))
					require.NoError(t, f.db.Save(execution).Error)
					task := &models.RunnerTask{ID: uuid.MustParse(tasks.CreatedID), OrganizationID: canvas.OrganizationID}
					require.NoError(t, task.RequireModelPolicyAdmission(f.db))
				}
				decision := map[string]any{"reviewRequired": true, "rationale": "Security findings need review.", "noCandidate": false, "reviewers": []any{map[string]any{"id": 2, "login": "octocat", "reason": "Owns the changed security code."}}}
				switch scenario {
				case "fallback", "no candidate", "ineligible fallback":
					decision["noCandidate"], decision["reviewers"] = true, []any{}
				case "malformed output":
					decision = map[string]any{"reviewRequired": true}
				case "reported lookup error":
					decision["error"] = "GitHub contribution lookup failed."
				case "concerning no review":
					decision["reviewRequired"], decision["reviewers"] = false, []any{}
				case "author selected":
					decision["reviewers"] = []any{map[string]any{"id": 1, "login": "author", "reason": "Author"}}
				case "bot selected":
					decision["reviewers"] = []any{map[string]any{"id": 4, "login": "agent[bot]", "reason": "Bot"}}
				case "reader selected":
					decision["reviewers"] = []any{map[string]any{"id": 5, "login": "reader", "reason": "Read access"}}
				case "duplicate reviewer":
					decision["reviewers"] = append(decision["reviewers"].([]any), decision["reviewers"].([]any)[0])
				case "additional reviewer", "additional reviewer without reason":
					reason := "Owns separate authorization changes."
					if scenario == "additional reviewer without reason" {
						reason = ""
					}
					decision["reviewers"] = append(decision["reviewers"].([]any), map[string]any{"id": 3, "login": "fallback-user", "reason": reason})
				}
				result, err := json.Marshal(decision)
				require.NoError(t, err)
				state, code := "succeeded", 0
				if scenario == "agent failure" {
					state, code = "failed", 1
				}
				err = action.HandleHook(core.ActionHookContext{Name: runner.HookPoll, Configuration: ctx.Configuration, Parameters: map[string]any{"task_id": "review-selection"}, Metadata: ctx.Metadata, ExecutionState: ctx.ExecutionState, RunnerTasks: &contexts.RunnerTaskContext{Task: &core.RunnerTask{ID: "review-selection", State: state, ExitCode: &code, Result: result}}})
				require.NoError(t, err)
				// Changes during selection must be detected by the reusable mutation component.
				host.permissionError = scenario == "permission error"
				host.lostPermission = scenario == "lost permission"
				host.changedIdentity = scenario == "identity changed"
				host.movedHead = scenario == "head moved"
				host.pushDuringLookups = scenario == "head moved during lookups"
				host.closed = scenario == "closed during selection"
			} else if err := action.Execute(ctx); err != nil {
				return err
			}
			var outputs []models.CanvasEvent
			require.NoError(t, f.db.Where("execution_id = ?", execution.ID).Order("created_at").Find(&outputs).Error)
			for i := range outputs {
				if err := visit(&outputs[i], execution); err != nil {
					return err
				}
			}
		}
		return nil
	}
	err = visit(root, nil)
	var messages []models.AppMessage
	require.NoError(t, f.db.Where("canvas_id = ? AND payload->>'runId' = ?", canvas.ID, run.ID.String()).Find(&messages).Error)
	if len(messages) == 0 {
		return nil, agentRuns, err
	}
	require.Len(t, messages, 1)
	var outcome map[string]any
	require.NoError(t, json.Unmarshal(messages[0].Payload, &outcome))
	return outcome, agentRuns, err
}

func TestReviewRequestFromPublishedReportToGitHub(t *testing.T) {
	for _, scenario := range []string{"favorable", "concerning", "hosted agent", "concerning no review", "fallback", "no candidate", "ineligible fallback", "malformed output", "reported lookup error", "agent failure", "author selected", "bot selected", "reader selected", "duplicate reviewer", "additional reviewer without reason", "additional reviewer", "discovery error", "permission error", "lost permission", "identity changed", "qualifying approval", "old approval", "withdrawn approval", "request error", "already requested", "head moved", "head moved during lookups", "stale report", "closed before selection", "closed during selection"} {
		t.Run(scenario, func(t *testing.T) {
			host := &reviewRequestHTTP{}
			score := 2.0
			if scenario == "favorable" {
				score = 5
			}
			f, message := reviewRequestCompletedProducer(t, host, score)
			fallback := ""
			if scenario == "fallback" || scenario == "discovery error" {
				fallback = "fallback-user"
			}
			if scenario == "ineligible fallback" {
				fallback = "reader"
			}
			consumer := reviewRequestTestCanvas(t, f, reviewRequestTemplateID, map[string]string{"mergeConfidenceAppId": f.canvas.ID.String(), "fallbackReviewer": fallback})
			if scenario == "hosted agent" {
				seedHostedModels(t, f.db, models.UsageProviderAnthropic, "claude-sonnet-4-6")
				node, err := models.FindCanvasNode(f.db, consumer.ID, reviewRequestAgentNodeID)
				require.NoError(t, err)
				config := node.Configuration.Data()
				delete(config, "model")
				delete(config, "credentials")
				require.NoError(t, f.db.Model(node).Updates(map[string]any{"ref": datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: models.SuperPlaneRunnerComponent}}), "configuration": datatypes.NewJSONType(config)}).Error)
			}
			trigger, err := models.FindCanvasNode(f.db, consumer.ID, reviewRequestTriggerNodeID)
			require.NoError(t, err)
			onBroadcast, err := f.deps.Registry.GetTrigger("onBroadcast")
			require.NoError(t, err)
			require.NoError(t, onBroadcast.(core.AppTrigger).OnAppMessage(core.AppMessageContext{Message: message, Configuration: trigger.Configuration.Data(), Events: workercontexts.NewEventContext(f.db, trigger, nil, nil)}))
			var root models.CanvasEvent
			require.NoError(t, f.db.Where("workflow_id = ? AND node_id = ?", consumer.ID, trigger.NodeID).First(&root).Error)
			host.discoveryError = scenario == "discovery error"
			host.empty = scenario == "no candidate"
			host.approved = slices.Contains([]string{"qualifying approval", "old approval", "withdrawn approval"}, scenario)
			host.oldApproval = scenario == "old approval"
			host.withdrawnApproval = scenario == "withdrawn approval"
			host.requestError = scenario == "request error"
			host.alreadyAsked = scenario == "already requested"
			host.movedHead = scenario == "stale report"
			host.closed = scenario == "closed before selection"
			readsBefore := len(host.requests)
			outcome, agents, err := runReviewRequestGraph(t, f, consumer, &root, host, scenario)
			if slices.Contains([]string{"discovery error", "permission error", "lost permission", "identity changed", "request error"}, scenario) {
				require.Error(t, err)
				require.Nil(t, outcome)
				require.Empty(t, host.reviewerBodies)
				return
			}
			require.NoError(t, err)
			if slices.Contains([]string{"concerning no review", "malformed output", "reported lookup error", "agent failure", "author selected", "bot selected", "reader selected", "duplicate reviewer", "additional reviewer without reason"}, scenario) {
				require.Nil(t, outcome)
				require.Empty(t, host.reviewerBodies)
				var run models.CanvasRun
				require.NoError(t, f.db.First(&run, "id = ?", root.RunID).Error)
				require.NotEmpty(t, run.Errors)
				return
			}
			require.NotNil(t, outcome)
			require.Equal(t, f.run.ID.String(), outcome["reportId"])
			require.Equal(t, f.canvas.ID.String(), outcome["sourceAppId"])
			require.Equal(t, "abc", outcome["reviewedCommit"])
			require.NotEmpty(t, outcome["runId"])
			decision := outcome["reviewRequest"].(map[string]any)
			require.NotEmpty(t, decision["rationale"])
			expected := "requested"
			switch scenario {
			case "favorable":
				expected = "review_unnecessary"
				require.Zero(t, agents)
				require.Len(t, host.requests, readsBefore)
			case "stale report", "closed before selection":
				expected = "stale"
				require.Zero(t, agents)
			case "head moved", "head moved during lookups", "closed during selection":
				expected = "stale"
			case "no candidate", "ineligible fallback":
				expected = "no_eligible_reviewer"
			case "qualifying approval", "already requested":
				expected = "already_covered"
			}
			require.Equal(t, expected, decision["status"])
			if expected != "requested" {
				require.Empty(t, host.reviewerBodies)
			} else {
				require.Len(t, host.reviewerBodies, 1)
				logins := []any{"octocat"}
				if scenario == "fallback" {
					logins = []any{"fallback-user"}
					require.Equal(t, true, decision["fallbackUsed"])
				}
				if scenario == "additional reviewer" {
					logins = append(logins, "fallback-user")
				}
				require.Equal(t, logins, host.reviewerBodies[0]["reviewers"])
				require.NotContains(t, host.reviewerBodies[0], "requested_reviewers")
			}
			for _, request := range host.requests {
				require.NotContains(t, request.URL.Path, "/statuses")
				require.NotContains(t, request.URL.Path, "/check-runs")
			}
			activity, err := models.FindPullRequestActivityByRunID(f.db, uuid.MustParse(outcome["runId"].(string)))
			require.NoError(t, err)
			require.Equal(t, f.pr.ID, activity.PullRequestID)
		})
	}
}

func TestReviewComponentsInOrdinaryCanvas(t *testing.T) {
	host := &reviewRequestHTTP{}
	f := newPublicationFixture(t)
	canvas, _ := support.CreateCanvas(t, f.canvas.OrganizationID, *f.canvas.CreatedBy, []models.CanvasNode{
		{NodeID: "ordinary-trigger", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onBroadcast"}})},
		{NodeID: "list", Name: "List candidates", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "github.listPullRequestReviewCandidates"}}), Configuration: datatypes.NewJSONType(map[string]any{"repository": "acme/app", "pullNumber": 42})},
		{NodeID: "ensure", Name: "Ensure reviewers", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "github.ensurePullRequestReviewers"}}), Configuration: datatypes.NewJSONType(map[string]any{"repository": "acme/app", "pullNumber": 42, "reviewers": `{{ $["List candidates"].data.candidates }}`})},
	}, []models.Edge{{SourceID: "ordinary-trigger", TargetID: "list", Channel: "default"}, {SourceID: "list", TargetID: "ensure", Channel: "default"}})
	require.Nil(t, canvas.FactoryID)
	root := &models.CanvasEvent{WorkflowID: canvas.ID, NodeID: "ordinary-trigger", Channel: "default", State: models.CanvasEventStatePending, Data: models.NewJSONValue(map[string]any{"data": map[string]any{}})}
	require.NoError(t, f.db.Create(root).Error)
	_, agents, err := runReviewRequestGraph(t, f, canvas, root, host, "")
	require.NoError(t, err)
	require.Zero(t, agents)
	require.Len(t, host.reviewerBodies, 1)
	require.Equal(t, []any{"octocat", "fallback-user"}, host.reviewerBodies[0]["reviewers"])
	var output models.CanvasEvent
	require.NoError(t, f.db.Where("workflow_id = ? AND node_id = ?", canvas.ID, "ensure").First(&output).Error)
	result := output.Data.Data().(map[string]any)["data"].(map[string]any)
	require.Equal(t, "requested", result["status"])
	require.Len(t, result["requested"], 2)
}

func TestReviewRequestRepeatsReportsAndRetriesOriginalReport(t *testing.T) {
	host := &reviewRequestHTTP{}
	f, message := reviewRequestCompletedProducer(t, host, 2)
	originalReport := message["payload"].(map[string]any)["reportId"]
	consumer := reviewRequestTestCanvas(t, f, reviewRequestTemplateID, map[string]string{"mergeConfidenceAppId": f.canvas.ID.String()})
	trigger, err := models.FindCanvasNode(f.db, consumer.ID, reviewRequestTriggerNodeID)
	require.NoError(t, err)
	onBroadcast, err := f.deps.Registry.GetTrigger("onBroadcast")
	require.NoError(t, err)
	deliver := func(message map[string]any) *models.CanvasEvent {
		var emitted []models.CanvasEvent
		require.NoError(t, onBroadcast.(core.AppTrigger).OnAppMessage(core.AppMessageContext{Message: message, Configuration: trigger.Configuration.Data(), Events: workercontexts.NewEventContext(f.db, trigger, nil, func(events []models.CanvasEvent) { emitted = append(emitted, events...) })}))
		require.Len(t, emitted, 1)
		root, err := models.FindCanvasEventInTransaction(f.db, emitted[0].ID)
		require.NoError(t, err)
		return root
	}
	original := deliver(message)
	for _, root := range []*models.CanvasEvent{original, deliver(message)} {
		outcome, agents, err := runReviewRequestGraph(t, f, consumer, root, host, "")
		require.NoError(t, err)
		require.Equal(t, 1, agents, "separate broadcasts may repeat selection")
		require.Equal(t, originalReport, outcome["reportId"])
	}
	f, newMessage := emitReviewRequestReport(t, f, host, 3)
	require.NotEqual(t, originalReport, newMessage["payload"].(map[string]any)["reportId"])
	outcome, agents, err := runReviewRequestGraph(t, f, consumer, deliver(newMessage), host, "")
	require.NoError(t, err)
	require.Equal(t, 1, agents)
	require.Equal(t, "abc", outcome["reviewedCommit"])
	retry, err := canvasactions.ReemitTriggerEvent(t.Context(), f.db, consumer, trigger.NodeID, original.ID)
	require.NoError(t, err)
	retried, err := models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
	require.NoError(t, err)
	require.Equal(t, original.Data.Data(), retried.Data.Data(), "retry preserves the original immutable report after a newer report")
	outcome, agents, err = runReviewRequestGraph(t, f, consumer, retried, host, "")
	require.NoError(t, err)
	require.Equal(t, 1, agents)
	require.Equal(t, originalReport, outcome["reportId"])
	host.movedHead = true
	retry, err = canvasactions.ReemitTriggerEvent(t.Context(), f.db, consumer, trigger.NodeID, original.ID)
	require.NoError(t, err)
	retried, err = models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
	require.NoError(t, err)
	outcome, agents, err = runReviewRequestGraph(t, f, consumer, retried, host, "")
	require.NoError(t, err)
	require.Zero(t, agents)
	require.Equal(t, "stale", outcome["reviewRequest"].(map[string]any)["status"])
	// Pause affects new deliveries while installed configuration remains intact.
	paused := trigger.Configuration.Data()
	paused["enabled"] = false
	var events []models.CanvasEvent
	require.NoError(t, onBroadcast.(core.AppTrigger).OnAppMessage(core.AppMessageContext{Message: newMessage, Configuration: paused, Events: workercontexts.NewEventContext(f.db, trigger, nil, func(created []models.CanvasEvent) { events = append(events, created...) })}))
	require.Empty(t, events)
}
