package factories

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/features"
	canvasactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// The fixture substitutes only GitHub responses and the external agent task.
// Both materialized graphs, expressions, component execution, and runner hooks are real.
type reviewRequestHTTP struct {
	statusDenied          bool
	statuses              []github.RepoStatus
	requests              []*http.Request
	reviewerBodies        []map[string]any
	reviewerRemovals      []string
	pendingReviewers      []pulls.ReviewerIdentity
	selectionEnvironments []map[string]string
	approved              bool
	oldApproval           bool
	withdrawnApproval     bool
	requestError          bool
	alreadyAsked          bool
	empty                 bool
	discoveryError        bool
	permissionError       bool
	movedHead             bool
	closed                bool
	lostPermission        bool
	changedIdentity       bool
	// pushDuringLookups moves the head after the permission lookup.
	pushDuringLookups bool
	// description and comments hold the published pull request state.
	description   string
	bodyFails     bool
	bodyWrites    []string
	comments      map[int64]string
	commentWrites int
	// commentFails comment writes fail with commentStatus. When commentApplied
	// is set, the host applies a failed write before the response is lost.
	commentFails   int
	commentStatus  int
	commentApplied bool
	// holdPublications leaves accepted publication requests pending.
	holdPublications bool
}

func (h *reviewRequestHTTP) commentResponse(request *http.Request) (*http.Response, error) {
	path := request.URL.Path
	if h.comments == nil {
		h.comments = map[int64]string{}
	}
	if request.Method == http.MethodGet && path == "/repos/acme/app/issues/42/comments" {
		comments := []map[string]any{}
		for _, id := range slices.Sorted(maps.Keys(h.comments)) {
			comments = append(comments, map[string]any{"id": id, "body": h.comments[id]})
		}
		encoded, err := json.Marshal(comments)
		return publicationHTTPResponse(200, string(encoded)), err
	}
	id, _ := strconv.ParseInt(strings.TrimPrefix(path, "/repos/acme/app/issues/comments/"), 10, 64)
	if request.Method == http.MethodGet {
		body, ok := h.comments[id]
		if !ok {
			return publicationHTTPResponse(404, `{"message":"Not Found"}`), nil
		}
		encoded, err := json.Marshal(map[string]any{"id": id, "body": body})
		return publicationHTTPResponse(200, string(encoded)), err
	}
	var comment struct{ Body string }
	if err := json.NewDecoder(request.Body).Decode(&comment); err != nil {
		return nil, err
	}
	failed := h.commentFails > 0
	if failed {
		h.commentFails--
		if !h.commentApplied {
			return publicationHTTPResponse(h.commentStatus, `{"message":"comment write failed"}`), nil
		}
	}
	h.commentWrites++
	if request.Method == http.MethodPost {
		id = int64(100 + len(h.comments))
	}
	h.comments[id] = comment.Body
	if failed {
		return publicationHTTPResponse(h.commentStatus, `{"message":"comment response was lost"}`), nil
	}
	encoded, err := json.Marshal(map[string]any{"id": id, "body": comment.Body})
	return publicationHTTPResponse(201, string(encoded)), err
}

func (h *reviewRequestHTTP) Do(request *http.Request) (*http.Response, error) {
	h.requests = append(h.requests, request)
	path := request.URL.Path
	if request.Method == http.MethodPost && strings.Contains(path, "/statuses/") {
		if h.statusDenied {
			return publicationHTTPResponse(403, `{"message":"status write denied"}`), nil
		}
		var status github.RepoStatus
		if err := json.NewDecoder(request.Body).Decode(&status); err != nil {
			return nil, err
		}
		h.statuses = append(h.statuses, status)
		return publicationHTTPResponse(201, `{}`), nil
	}
	if strings.HasSuffix(path, "/requested_reviewers") && request.Method == http.MethodDelete {
		var body struct{ Reviewers []string }
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			return nil, err
		}
		h.reviewerRemovals = append(h.reviewerRemovals, body.Reviewers...)
		h.pendingReviewers = slices.DeleteFunc(h.pendingReviewers, func(reviewer pulls.ReviewerIdentity) bool { return slices.Contains(body.Reviewers, reviewer.Login) })
		return publicationHTTPResponse(200, `{}`), nil
	}
	if strings.HasSuffix(path, "/requested_reviewers") && request.Method == http.MethodPost {
		if h.requestError {
			return publicationHTTPResponse(403, `{"message":"review request denied"}`), nil
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			return nil, err
		}
		for _, raw := range body["reviewers"].([]any) {
			login := raw.(string)
			id := int64(2)
			if login == "fallback-user" {
				id = 3
			}
			h.pendingReviewers = append(h.pendingReviewers, pulls.ReviewerIdentity{ID: id, Login: login})
		}
		h.reviewerBodies = append(h.reviewerBodies, body)
		return publicationHTTPResponse(201, `{"number":42,"requested_reviewers":[{"login":"unrelated"},{"id":2,"login":"octocat"}]}`), nil
	}
	if request.Method == http.MethodPatch && path == "/repos/acme/app/pulls/42" {
		if h.bodyFails {
			return publicationHTTPResponse(403, `{"message":"body write denied"}`), nil
		}
		var update struct{ Body string }
		if err := json.NewDecoder(request.Body).Decode(&update); err != nil {
			return nil, err
		}
		h.description = update.Body
		h.bodyWrites = append(h.bodyWrites, update.Body)
		return publicationHTTPResponse(200, `{}`), nil
	}
	if strings.HasPrefix(path, "/repos/acme/app/issues/") && strings.Contains(path, "comments") {
		return h.commentResponse(request)
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
		requestedReviewers := append([]pulls.ReviewerIdentity{{ID: 9, Login: "unrelated"}}, h.pendingReviewers...)
		if h.alreadyAsked {
			requestedReviewers = append(requestedReviewers, pulls.ReviewerIdentity{ID: 2, Login: "octocat"})
		}
		encoded, err := json.Marshal(requestedReviewers)
		if err != nil {
			return nil, err
		}
		requested := string(encoded)
		state := "open"
		if h.closed {
			state = "closed"
		}
		description := h.description
		if description == "" {
			description = "User description"
		}
		return publicationHTTPResponse(200, fmt.Sprintf(`{"number":42,"state":%q,"body":%q,"head":{"sha":%q},"base":{"ref":"main","sha":"base-sha"},"user":{"login":"author"},"requested_reviewers":%s}`, state, description, sha, requested)), nil
	}
	return nil, fmt.Errorf("unexpected GitHub request: %s %s", request.Method, path)
}

type reviewRequestManagedIntegration struct {
	core.IntegrationContext
	store core.PublicationStore
}

func (c reviewRequestManagedIntegration) Publications() core.PublicationStore { return c.store }

type reviewRequestFailedActivity struct{ core.FactoryContext }

func (c reviewRequestFailedActivity) UpdatePullRequestActivity(core.UpdatePullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	return nil, errors.New("activity update unavailable")
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
	return reviewRequestTestCanvasWith(t, f, template, factoryTemplateInput{appName: template, installParams: params})
}

// The GitHub integration is wired like an installation, so ordered
// publication can authorize its persisted source and attempt records.
func reviewRequestTestCanvasWith(t *testing.T, f publicationFixture, template string, input factoryTemplateInput) *models.Canvas {
	t.Helper()
	vcs := f.workspace.OnboardingConfigValue().VCSIntegrationID
	if vcs != "" {
		input.integrations = map[string]factoryTemplateIntegration{"github": {id: vcs, name: "github"}}
	}
	result, err := materializeFactoryTemplate(template, models.ProviderGitHub, input)
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
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(f.db, canvas)
	require.NoError(t, err)
	for i := range version.Nodes {
		if !strings.HasPrefix(version.Nodes[i].ComponentName(), "github.") || vcs == "" {
			continue
		}
		version.Nodes[i].IntegrationID = &vcs
		require.NoError(t, f.db.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, version.Nodes[i].ID).Update("app_installation_id", uuid.MustParse(vcs)).Error)
	}
	require.NoError(t, f.db.Save(version).Error)
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
	require.NoError(t, workercontexts.NewFactoryExecutionStateContext(f.db, assessment, nil).Emit("passed", "runnerClaudeCode.finished", []any{map[string]any{"status": "succeeded"}}))
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
	recorded := map[string]any{}
	var visit func(*models.CanvasEvent, *models.CanvasNodeExecution) error
	visit = func(incoming *models.CanvasEvent, previous *models.CanvasNodeExecution) error {
		var executionError error
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
				ID: execution.ID, WorkflowID: canvas.ID.String(), NodeID: node.NodeID, RunID: run.ID, Configuration: execution.Configuration.Data(), HTTP: host, Integration: reviewRequestIntegration(),
				Publications:   workercontexts.NewPublicationContext(f.db, canvas.OrganizationID, node, execution),
				Expressions:    workercontexts.NewExpressionContext(builder),
				Metadata:       workercontexts.NewExecutionMetadataContext(f.db, execution),
				ExecutionState: workercontexts.NewExecutionStateContext(f.db, execution, nil),
				Factory:        workercontexts.NewFactoryContext(f.db, canvas, execution),
				Apps:           workercontexts.NewAppExecutionContext(f.db, canvas, node, execution),
				Runs:           workercontexts.NewRunExecutionContext(f.db, canvas, node, execution),
				Requests:       &contexts.RequestContext{},
			}
			if scenario == "activity failure" && node.NodeID == "review-outcome-activity" {
				ctx.Factory = reviewRequestFailedActivity{ctx.Factory}
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
				tasks := &contexts.RunnerTaskContext{}
				ctx.RunnerTasks = tasks
				ctx.Secrets = &contexts.SecretsContext{IntegrationKeys: map[string]map[string][]byte{"github": {}, "claude": {"ANTHROPIC_API_KEY": []byte("test-key")}}}
				ctx.Webhook = &contexts.NodeWebhookContext{}
				if scenario == "instructions changed before dispatch" {
					setReviewRequestInstructions(t, f, canvas, "A newer preference for future attempts.")
				}
				if scenario == "hosted agent" {
					ctx.HostedLLM = &contexts.HostedLLMContext{Default: core.DefaultHostedLLMModel{Provider: models.UsageProviderAnthropic, Model: "claude-sonnet-4-6"}, Access: core.HostedLLMAccess{APIKey: "test-key", AllowedModels: []string{"claude-sonnet-4-6"}}}
				}
				require.NoError(t, action.Execute(ctx))
				var dispatched struct {
					Files       []runner.TaskFile                `json:"files"`
					Environment []runner.TaskEnvironmentVariable `json:"environment"`
				}
				require.NoError(t, json.Unmarshal(tasks.Payload, &dispatched))
				selectionEnvironment := map[string]string{}
				for _, variable := range dispatched.Environment {
					selectionEnvironment[variable.Name] = variable.Value
				}
				host.selectionEnvironments = append(host.selectionEnvironments, selectionEnvironment)
				promptIndex := slices.IndexFunc(dispatched.Files, func(file runner.TaskFile) bool { return file.Path == "prompts/02-select-reviewers.txt" })
				require.NotEqual(t, -1, promptIndex)
				dispatchedPrompt := dispatched.Files[promptIndex].Content
				require.Contains(t, dispatchedPrompt, prompt)
				if scenario == "instructions changed before dispatch" {
					require.NotContains(t, dispatchedPrompt, "A newer preference for future attempts.")
				}
				instructions, _ := execution.Configuration.Data()["additionalInstructions"].(string)
				if instructions == "" {
					require.NotContains(t, dispatchedPrompt, "Additional instructions for this automation:")
				} else {
					require.Equal(t, 1, strings.Count(dispatchedPrompt, instructions))
					require.NotContains(t, prompt, instructions, "saved managed prompts must not contain user text")
				}
				if scenario == "hosted agent" {
					require.NoError(t, f.db.Save(execution).Error)
					task := &models.RunnerTask{ID: uuid.MustParse(tasks.CreatedID), OrganizationID: canvas.OrganizationID}
					require.NoError(t, task.RequireModelPolicyAdmission(f.db))
				}
				decision := map[string]any{"reviewRequired": true, "rationale": "Security findings need review.", "noCandidate": false, "reviewers": []any{map[string]any{"id": 2, "login": "octocat", "reason": "Owns the changed security code."}}}
				switch scenario {
				case "replacement":
					decision["rationale"] = "The previous reviewer is unavailable. Select the authorization owner."
					decision["reviewers"] = []any{map[string]any{"id": 3, "login": "fallback-user", "reason": "The previous reviewer is unavailable. Owns authorization changes."}}
				case "fallback", "no candidate", "no suitable candidate", "ineligible fallback":
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
				result, readErr := readReviewDecision(t, steps[2].(map[string]any)["command"].(string), result, selectionEnvironment)
				state, code := "succeeded", 0
				if scenario == "agent failure" || readErr != nil {
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
			} else if isPublicationComponent(node.ComponentName()) {
				ctx.Requests = workercontexts.NewExecutionRequestContext(f.db, execution)
				require.NoError(t, action.Execute(ctx))
				if !host.holdPublications {
					processReviewRequestPublication(t, f, host, execution)
				}
			} else {
				if err := action.Execute(ctx); err != nil {
					require.NoError(t, ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, err.Error()))
					executionError = errors.Join(executionError, err)
					continue
				}
				if node.ComponentName() == "github.setPullRequestReviewCheck" {
					if node.NodeID == "pr-check-pending" && host.statusDenied {
						records, err := ctx.Publications.List("github.review-check/")
						require.NoError(t, err)
						require.Len(t, records, 1)
						integration := reviewRequestManagedIntegration{IntegrationContext: ctx.Integration, store: ctx.Publications}
						require.Error(t, pulls.RefreshReviewCheck(core.IntegrationHookContext{Parameters: map[string]any{"key": records[0].Key}, Integration: integration, HTTP: host}))
					}
					metadata, _ := ctx.Metadata.Get().(map[string]any)
					if key, ok := metadata["key"].(string); ok {
						integration := reviewRequestManagedIntegration{IntegrationContext: ctx.Integration, store: ctx.Publications}
						refreshErr := pulls.RefreshReviewCheck(core.IntegrationHookContext{Parameters: map[string]any{"key": key}, Integration: integration, HTTP: host})
						if host.statusDenied {
							require.Error(t, refreshErr)
						} else {
							require.NoError(t, refreshErr)
						}
						require.NoError(t, action.HandleHook(core.ActionHookContext{Name: "awaitReviewCheck", Metadata: ctx.Metadata, ExecutionState: ctx.ExecutionState, Publications: ctx.Publications, Requests: ctx.Requests}))
					}
				}
			}
			var outputs []models.CanvasEvent
			require.NoError(t, f.db.Where("execution_id = ?", execution.ID).Order("created_at").Find(&outputs).Error)
			for i := range outputs {
				recorded[node.NodeID] = outputs[i].Data.Data().(map[string]any)["data"]
				if err := visit(&outputs[i], execution); err != nil {
					executionError = errors.Join(executionError, err)
				}
			}
		}
		return executionError
	}
	err = visit(root, nil)
	var messageCount int64
	require.NoError(t, f.db.Model(&models.AppMessage{}).Where("canvas_id = ?", canvas.ID).Count(&messageCount).Error)
	require.Zero(t, messageCount, "Review request must not broadcast outgoing app messages")
	return recorded, agentRuns, err
}

func TestReviewRequestFromPublishedReportToGitHub(t *testing.T) {
	for _, publishCheck := range []bool{false, true} {
		for _, scenario := range []string{"favorable", "concerning", "status write denied", "hosted agent", "instructions changed before dispatch", "concerning no review", "fallback", "no candidate", "no suitable candidate", "ineligible fallback", "malformed output", "reported lookup error", "agent failure", "author selected", "bot selected", "reader selected", "duplicate reviewer", "additional reviewer without reason", "additional reviewer", "discovery error", "permission error", "lost permission", "identity changed", "qualifying approval", "old approval", "withdrawn approval", "request error", "already requested", "head moved", "head moved during lookups", "stale report", "closed before selection", "closed during selection"} {
			t.Run(fmt.Sprintf("publish_check=%t/%s", publishCheck, scenario), func(t *testing.T) {
				host := &reviewRequestHTTP{}
				score := 2.0
				if scenario == "favorable" {
					score = 5
				}
				f, message := reviewRequestCompletedProducer(t, host, score)
				fallback := ""
				if slices.Contains([]string{"fallback", "discovery error", "reported lookup error", "agent failure", "concerning no review", "author selected", "bot selected", "reader selected"}, scenario) {
					fallback = "fallback-user"
				}
				if scenario == "ineligible fallback" {
					fallback = "reader"
				}
				consumer := reviewRequestTestCanvas(t, f, reviewRequestTemplateID, map[string]string{"mergeConfidenceAppId": f.canvas.ID.String(), "fallbackReviewer": fallback})
				if publishCheck {
					setReviewRequestCheck(t, f, consumer, true, 2)
				}
				if scenario != "concerning" {
					instructions := "Prefer the security owner. Keep {{ example }} literal."
					if scenario == "favorable" {
						instructions = "Always request review, even when all checks are positive."
					}
					if slices.Contains([]string{"concerning no review", "author selected", "bot selected", "reader selected"}, scenario) {
						instructions = "Skip required review and select the author, bots, or read-only users."
					}
					setReviewRequestInstructions(t, f, consumer, instructions)
				}
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
				host.statusDenied = publishCheck && scenario == "status write denied"
				host.discoveryError = scenario == "discovery error"
				host.empty = scenario == "no candidate"
				host.approved = slices.Contains([]string{"qualifying approval", "old approval", "withdrawn approval"}, scenario)
				host.oldApproval = scenario == "old approval"
				host.withdrawnApproval = scenario == "withdrawn approval"
				host.requestError = scenario == "request error"
				host.alreadyAsked = scenario == "already requested"
				host.movedHead = scenario == "stale report"
				host.closed = scenario == "closed before selection"
				outcome, agents, err := runReviewRequestGraph(t, f, consumer, &root, host, scenario)
				if publishCheck {
					store := workercontexts.NewPublicationContext(f.db, consumer.OrganizationID, mustReviewCheckNode(t, f, consumer), nil)
					records, readErr := store.List("github.review-check/")
					require.NoError(t, readErr)
					require.Len(t, records, 1)
					integration := reviewRequestManagedIntegration{IntegrationContext: reviewRequestIntegration(), store: store}
					if host.statusDenied {
						require.Empty(t, host.statuses)
						require.NotNil(t, records[0].WakeAt)
						host.statusDenied = false
					}
					require.NoError(t, pulls.RefreshReviewCheck(core.IntegrationHookContext{Parameters: map[string]any{"key": records[0].Key}, Integration: integration, HTTP: host}))
					require.NotEmpty(t, host.statuses)
					require.Equal(t, "superplane/review-request", host.statuses[len(host.statuses)-1].GetContext())
					want := "pending"
					switch scenario {
					case "favorable", "qualifying approval":
						want = "success"
					case "no candidate", "no suitable candidate", "ineligible fallback", "concerning no review", "malformed output", "reported lookup error", "agent failure", "author selected", "bot selected", "reader selected", "duplicate reviewer", "additional reviewer without reason", "discovery error", "permission error", "lost permission", "identity changed", "request error":
						want = "error"
					case "withdrawn approval":
						want = "failure"
					}
					if scenario == "closed before selection" || scenario == "closed during selection" {
						want = "error"
					}
					require.Equal(t, want, host.statuses[len(host.statuses)-1].GetState())
				} else {
					require.Empty(t, host.statuses)
				}
				if slices.Contains([]string{"discovery error", "permission error", "lost permission", "identity changed", "request error"}, scenario) {
					require.Error(t, err)
					require.NotContains(t, outcome, "request-reviewers")
					require.Empty(t, host.reviewerBodies)
					return
				}
				require.NoError(t, err)
				if slices.Contains([]string{"concerning no review", "malformed output", "reported lookup error", "agent failure", "author selected", "bot selected", "reader selected", "duplicate reviewer", "additional reviewer without reason"}, scenario) {
					require.NotContains(t, outcome, "request-reviewers")
					require.Empty(t, host.reviewerBodies)
					var run models.CanvasRun
					require.NoError(t, f.db.First(&run, "id = ?", root.RunID).Error)
					require.NotEmpty(t, run.Errors)
					return
				}
				activity, err := models.FindPullRequestActivityByRunID(f.db, root.RunID)
				require.NoError(t, err)
				require.Equal(t, f.pr.ID, activity.PullRequestID)
				expected := "requested"
				wantTitle := "Review requested"
				switch scenario {
				case "favorable":
					wantTitle = "Review unnecessary"
					require.Zero(t, agents)
					require.Contains(t, outcome, "withdraw-reviewers")
				case "stale report", "closed before selection":
					wantTitle = "Stale report"
					require.Zero(t, agents)
				case "head moved", "head moved during lookups", "closed during selection":
					wantTitle = "Stale report"
					expected = "skipped"
				case "no candidate", "no suitable candidate", "ineligible fallback":
					wantTitle = "No eligible reviewer"
					require.Contains(t, outcome, "no-reviewer-activity")
					if scenario == "no candidate" {
						require.Zero(t, agents)
						require.NotContains(t, outcome, reviewRequestAgentNodeID)
					} else {
						require.Equal(t, 1, agents)
						require.Contains(t, activity.Description, "Security findings need review.")
					}
				case "qualifying approval", "already requested":
					wantTitle = "Review already covered"
					expected = "already_covered"
				}
				require.Equal(t, wantTitle, activity.Title)
				require.NotEmpty(t, activity.Description)
				if request, ok := outcome["request-reviewers"].(map[string]any); ok {
					require.Equal(t, expected, request["status"])
				} else {
					expected = ""
				}
				if expected != "requested" {
					require.Empty(t, host.reviewerBodies)
				} else {
					logins := []any{"octocat"}
					if scenario == "fallback" {
						logins = []any{"fallback-user"}
						require.Equal(t, true, outcome[reviewRequestAgentNodeID].(map[string]any)["result"].(map[string]any)["fallbackUsed"])
					}
					if scenario == "additional reviewer" {
						logins = append(logins, "fallback-user")
					}
					actual := []any{}
					for _, body := range host.reviewerBodies {
						actual = append(actual, body["reviewers"].([]any)...)
						require.NotContains(t, body, "requested_reviewers")
					}
					require.Equal(t, logins, actual)
				}
				if !publishCheck {
					for _, request := range host.requests {
						require.NotContains(t, request.URL.Path, "/statuses")
						require.NotContains(t, request.URL.Path, "/check-runs")
					}
				}
			})
		}
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
	setReviewRequestInstructions(t, f, consumer, "Prefer the original security owner.")
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
		require.Contains(t, outcome, "request-reviewers")
	}
	f, newMessage := emitReviewRequestReport(t, f, host, 3)
	require.NotEqual(t, originalReport, newMessage["payload"].(map[string]any)["reportId"])
	outcome, agents, err := runReviewRequestGraph(t, f, consumer, deliver(newMessage), host, "")
	require.NoError(t, err)
	require.Equal(t, 1, agents)
	require.Equal(t, "abc", outcome["request-reviewers"].(map[string]any)["headSha"])
	setReviewRequestInstructions(t, f, consumer, "Prefer the API owner. Explain {{ ownership }} literally.")
	var originalSelection models.CanvasNodeExecution
	require.NoError(t, f.db.Where("root_event_id = ? AND node_id = ?", original.ID, reviewRequestAgentNodeID).First(&originalSelection).Error)
	require.Equal(t, "Prefer the original security owner.", originalSelection.Configuration.Data()["additionalInstructions"], "saved changes do not rewrite past attempts")
	retry, err := canvasactions.ReemitTriggerEvent(t.Context(), f.db, consumer, trigger.NodeID, original.ID)
	require.NoError(t, err)
	retried, err := models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
	require.NoError(t, err)
	require.Equal(t, original.Data.Data(), retried.Data.Data(), "retry preserves the original immutable report after a newer report")
	outcome, agents, err = runReviewRequestGraph(t, f, consumer, retried, host, "")
	require.NoError(t, err)
	require.Equal(t, 1, agents)
	require.Contains(t, outcome, "request-reviewers")
	var retriedSelection models.CanvasNodeExecution
	require.NoError(t, f.db.Where("root_event_id = ? AND node_id = ?", retried.ID, reviewRequestAgentNodeID).First(&retriedSelection).Error)
	require.Equal(t, "Prefer the API owner. Explain {{ ownership }} literally.", retriedSelection.Configuration.Data()["additionalInstructions"])
	setReviewRequestInstructions(t, f, consumer, "")
	retry, err = canvasactions.ReemitTriggerEvent(t.Context(), f.db, consumer, trigger.NodeID, original.ID)
	require.NoError(t, err)
	retried, err = models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
	require.NoError(t, err)
	outcome, agents, err = runReviewRequestGraph(t, f, consumer, retried, host, "")
	require.NoError(t, err)
	require.Equal(t, 1, agents)
	require.Contains(t, outcome, "request-reviewers")
	var clearedSelection models.CanvasNodeExecution
	require.NoError(t, f.db.Where("root_event_id = ? AND node_id = ?", retried.ID, reviewRequestAgentNodeID).First(&clearedSelection).Error)
	require.Equal(t, "", clearedSelection.Configuration.Data()["additionalInstructions"])
	host.movedHead = true
	retry, err = canvasactions.ReemitTriggerEvent(t.Context(), f.db, consumer, trigger.NodeID, original.ID)
	require.NoError(t, err)
	retried, err = models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
	require.NoError(t, err)
	outcome, agents, err = runReviewRequestGraph(t, f, consumer, retried, host, "")
	require.NoError(t, err)
	require.Zero(t, agents)
	require.Contains(t, outcome, "stale-report-activity")
	// Pause affects new deliveries while installed configuration remains intact.
	paused := trigger.Configuration.Data()
	paused["enabled"] = false
	var events []models.CanvasEvent
	require.NoError(t, onBroadcast.(core.AppTrigger).OnAppMessage(core.AppMessageContext{Message: newMessage, Configuration: paused, Events: workercontexts.NewEventContext(f.db, trigger, nil, func(created []models.CanvasEvent) { events = append(events, created...) })}))
	require.Empty(t, events)
}

// processReviewRequestPublication runs the pending request of a publication step
// through the publication worker with substituted GitHub responses.
func processReviewRequestPublication(t *testing.T, f publicationFixture, host *reviewRequestHTTP, execution *models.CanvasNodeExecution) {
	t.Helper()
	previous := newPublicationConnection
	newPublicationConnection = func(*gorm.DB, IntakeDependencies, *models.CanvasNode, *models.Integration) (core.IntegrationContext, core.HTTPContext) {
		return reviewRequestIntegration(), host
	}
	defer func() { newPublicationConnection = previous }()
	var request models.CanvasNodeRequest
	err := f.db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).First(&request).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	require.NoError(t, err)
	_, err = ProcessBodySectionPublicationRequest(t.Context(), f.deps, request.ID, models.LockNodeRequestNow)
	require.NoError(t, err)
}

func setReviewRequestInstructions(t *testing.T, f publicationFixture, canvas *models.Canvas, instructions string) {
	t.Helper()
	node, err := models.FindCanvasNode(f.db, canvas.ID, reviewRequestAgentNodeID)
	require.NoError(t, err)
	configuration := node.Configuration.Data()
	configuration["additionalInstructions"] = instructions
	require.NoError(t, f.db.Model(node).Update("configuration", datatypes.NewJSONType(configuration)).Error)
}

func mustReviewCheckNode(t *testing.T, f publicationFixture, canvas *models.Canvas) *models.CanvasNode {
	t.Helper()
	node, err := models.FindCanvasNode(f.db, canvas.ID, "pr-check-pending")
	require.NoError(t, err)
	return node
}

func setReviewRequestCheck(t *testing.T, f publicationFixture, canvas *models.Canvas, enabled bool, generation int) {
	t.Helper()
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(f.db, canvas)
	require.NoError(t, err)
	for i := range version.Nodes {
		node := &version.Nodes[i]
		switch node.ID {
		case "pr-check-pending":
			node.Configuration["enabled"], node.Configuration["activationGeneration"] = enabled, generation
		case reviewRequestAgentNodeID:
			node.Configuration["publishPRCheck"], node.Configuration["prCheckActivationGeneration"] = enabled, generation
		default:
			continue
		}
		require.NoError(t, f.db.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, node.ID).Update("configuration", datatypes.NewJSONType(node.Configuration)).Error)
	}
	require.NoError(t, f.db.Save(version).Error)
}

func TestReviewRequestResetKeepsActiveCheck(t *testing.T) {
	host := &reviewRequestHTTP{}
	f, message := reviewRequestCompletedProducer(t, host, 2)
	consumer := reviewFindingsConsumer(t, f, reviewFindingsSettings{}, "fallback-user")
	setReviewRequestCheck(t, f, consumer, true, 7)
	root := deliverReviewRequestReport(t, f, consumer, message)
	_, agents, err := runReviewRequestGraph(t, f, consumer, root, host, "")
	require.NoError(t, err)
	require.Equal(t, 1, agents)
	store := workercontexts.NewPublicationContext(f.db, consumer.OrganizationID, mustReviewCheckNode(t, f, consumer), nil)
	before, err := store.List("github.review-check/")
	require.NoError(t, err)
	require.Len(t, before, 1)
	require.Equal(t, "pending", host.statuses[len(host.statuses)-1].GetState())

	f.deps.WebhookBaseURL = "http://localhost:8000"
	reset, err := resetOrganizationBuiltInAutomations(t.Context(), consumer.OrganizationID, reviewRequestTemplateID, f.deps)
	require.NoError(t, err)
	require.Equal(t, 1, reset.Reset)
	after, err := store.Get(before[0].Key)
	require.NoError(t, err)
	require.False(t, after.Disabled)
	require.Equal(t, before[0].Order, after.Order)
	require.Equal(t, before[0].Generation, after.Generation)
	require.Equal(t, before[0].ActivationGeneration, after.ActivationGeneration)
	require.JSONEq(t, string(before[0].Payload), string(after.Payload))

	host.approved = true
	integration := reviewRequestManagedIntegration{IntegrationContext: reviewRequestIntegration(), store: store}
	require.NoError(t, pulls.RefreshReviewCheck(core.IntegrationHookContext{Parameters: map[string]any{"key": after.Key}, Integration: integration, HTTP: host}))
	require.Equal(t, "success", host.statuses[len(host.statuses)-1].GetState(), "reset keeps approval tracking active")
	require.Len(t, host.reviewerBodies, 1)
	require.Len(t, host.selectionEnvironments, 1)
}

func TestManagedReviewerOwnershipSurvivesExecutionRollback(t *testing.T) {
	host := &reviewRequestHTTP{}
	f, message := reviewRequestCompletedProducer(t, host, 2)
	consumer := reviewFindingsConsumer(t, f, reviewFindingsSettings{}, "")
	root := deliverReviewRequestReport(t, f, consumer, message)
	_, err := models.FindOrCreateCanvasRunForRootEventInTransaction(f.db, root)
	require.NoError(t, err)
	node, err := models.FindCanvasNode(f.db, consumer.ID, "request-reviewers")
	require.NoError(t, err)
	execution := support.CreateCanvasNodeExecution(t, consumer.ID, node.NodeID, root.ID, root.ID)
	config := map[string]any{"repository": "acme/app", "pullNumber": 42, "managed": true, "expectedHeadSha": "abc", "reviewers": []pulls.ReviewerIdentity{{ID: 2, Login: "octocat"}}}
	ctx := core.ExecutionContext{ID: execution.ID, WorkflowID: consumer.ID.String(), NodeID: node.NodeID, Configuration: config, Integration: reviewRequestIntegration(), HTTP: host, ExecutionState: &contexts.ExecutionStateContext{}}
	interrupted := errors.New("execution interrupted after GitHub accepted the request")
	err = f.db.Transaction(func(tx *gorm.DB) error {
		ctx.Publications = workercontexts.NewPublicationContext(tx, consumer.OrganizationID, node, execution)
		require.NoError(t, (&pulls.EnsurePullRequestReviewers{}).Execute(ctx))
		return interrupted
	})
	require.ErrorIs(t, err, interrupted)
	require.Len(t, host.pendingReviewers, 1)
	// A later report can still withdraw the request after the execution rolls back.
	ctx.Publications = workercontexts.NewPublicationContext(f.db, consumer.OrganizationID, node, execution)
	config["reviewers"] = []pulls.ReviewerIdentity{}
	require.NoError(t, (&pulls.EnsurePullRequestReviewers{}).Execute(ctx))
	require.Empty(t, host.pendingReviewers)
	require.Equal(t, []string{"octocat"}, host.reviewerRemovals)
}
