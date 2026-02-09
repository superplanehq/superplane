package cursor

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	AgentCompletedPayloadType = "cursor.agent.completed"

	webhookEventStatusChange = "statusChange"
	agentStatusFinished      = "FINISHED"
	agentStatusError         = "ERROR"

	pollActionName    = "poll"
	pollInterval      = 20 * time.Second
	webhookSecretSize = 32
)

type LaunchCloudAgent struct{}

type LaunchCloudAgentSpec struct {
	Repository   string `json:"repository"`
	Ref          string `json:"ref"`
	Prompt       string `json:"prompt"`
	Model        string `json:"model"`
	AutoCreatePR bool   `json:"autoCreatePr"`
}

type LaunchCloudAgentExecutionMetadata struct {
	Agent         *LaunchAgentResponse `json:"agent,omitempty" mapstructure:"agent"`
	Status        *AgentStatusWebhook  `json:"status,omitempty" mapstructure:"status"`
	WebhookSecret string               `json:"webhookSecret,omitempty" mapstructure:"webhookSecret"`
}

type AgentStatusWebhook struct {
	Event     string       `json:"event"`
	Timestamp string       `json:"timestamp"`
	ID        string       `json:"id"`
	Status    string       `json:"status"`
	Source    AgentSource  `json:"source"`
	Target    *AgentTarget `json:"target,omitempty"`
	Summary   string       `json:"summary,omitempty"`
}

type AgentSource struct {
	Repository string `json:"repository"`
	Ref        string `json:"ref,omitempty"`
}

func updateAgentFromWebhook(agent *LaunchAgentResponse, payload AgentStatusWebhook) *LaunchAgentResponse {
	if agent == nil {
		agent = &LaunchAgentResponse{}
	}

	agent.ID = payload.ID
	agent.Status = payload.Status
	agent.Summary = payload.Summary
	agent.Source = &LaunchAgentSource{
		Repository: payload.Source.Repository,
		Ref:        payload.Source.Ref,
	}
	if payload.Target != nil {
		agent.Target = payload.Target
	}

	return agent
}

func (l *LaunchCloudAgent) Name() string {
	return "cursor.launchCloudAgent"
}

func (l *LaunchCloudAgent) Label() string {
	return "Launch Cloud Agent"
}

func (l *LaunchCloudAgent) Description() string {
	return "Launch a Cursor background agent for a repository and wait for completion"
}

func (l *LaunchCloudAgent) Documentation() string {
	return `Launches a Cursor background agent and waits for completion.

## Configuration

- **Repository**: Git repository URL (e.g., https://github.com/org/repo)
- **Ref**: Branch/tag/commit SHA (defaults to main)
- **Prompt**: Task instructions for the agent
- **Model**: Optional Cursor model identifier
- **Auto-create PR**: Create a PR when the agent finishes

## Output

Emits ` + "`cursor.agent.completed`" + ` when the agent reaches FINISHED, including final status payload and target URLs (PR URL when created).

## Notes

- Uses a per-execution webhook secret to verify completion callbacks.
- Falls back to polling agent status if webhook delivery is delayed.`
}

func (l *LaunchCloudAgent) Icon() string {
	return "bot"
}

func (l *LaunchCloudAgent) Color() string {
	return "gray"
}

func (l *LaunchCloudAgent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *LaunchCloudAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "https://github.com/org/repo",
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeGitRef,
			Required:    false,
			Default:     "main",
			Placeholder: "main",
		},
		{
			Name:        "prompt",
			Label:       "Prompt",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Describe the task for the agent",
		},
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Placeholder: "Select a Cursor model (optional)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "model",
				},
			},
		},
		{
			Name:        "autoCreatePr",
			Label:       "Auto-create PR",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     "true",
			Description: "Create a pull request when the agent completes",
		},
	}
}

func (l *LaunchCloudAgent) Setup(ctx core.SetupContext) error {
	// Ensure the node has a webhook record so Cursor can call back with status changes.
	if err := ctx.Integration.RequestWebhook(WebhookConfiguration{}); err != nil {
		return err
	}

	spec := LaunchCloudAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	return l.validateSpec(spec)
}

func (l *LaunchCloudAgent) Execute(ctx core.ExecutionContext) error {
	spec := LaunchCloudAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if err := l.validateSpec(spec); err != nil {
		return err
	}

	webhookURL, err := l.nodeWebhookURL(ctx)
	if err != nil {
		return err
	}

	secret, err := newWebhookSecret()
	if err != nil {
		return err
	}

	client, err := NewCloudAgentsClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	ref := strings.TrimSpace(spec.Ref)
	ref = strings.TrimPrefix(ref, "refs/heads/")
	ref = strings.TrimPrefix(ref, "refs/tags/")
	if ref == "" {
		ref = "main"
	}

	repo := strings.TrimSpace(spec.Repository)
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "http://github.com/")
	repo = strings.TrimSuffix(repo, ".git")

	req := LaunchAgentRequest{
		Prompt: LaunchAgentPrompt{Text: spec.Prompt},
		Model:  strings.TrimSpace(spec.Model),
		Source: LaunchAgentSource{
			Repository: repo,
			Ref:        strings.TrimSpace(ref),
		},
		Target: &LaunchAgentTarget{
			AutoCreatePr: spec.AutoCreatePR,
			AutoBranch:   true,
		},
		Webhook: &LaunchAgentWebhook{
			URL:    webhookURL,
			Secret: secret,
		},
	}

	agent, err := client.LaunchAgent(req)
	if err != nil {
		return err
	}

	if agent == nil || agent.ID == "" {
		return fmt.Errorf("cursor returned empty agent id")
	}

	if err := ctx.ExecutionState.SetKV("agent_id", agent.ID); err != nil {
		return err
	}

	metadata := LaunchCloudAgentExecutionMetadata{
		Agent:         agent,
		WebhookSecret: secret,
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	// Poll as a fallback in case Cursor webhook delivery is delayed.
	return ctx.Requests.ScheduleActionCall(pollActionName, map[string]any{}, pollInterval)
}

func (l *LaunchCloudAgent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *LaunchCloudAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *LaunchCloudAgent) Actions() []core.Action {
	return []core.Action{
		{
			Name:           pollActionName,
			UserAccessible: false,
		},
		{
			Name:           "view",
			UserAccessible: true,
		},
	}
}

func (l *LaunchCloudAgent) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case pollActionName:
		return l.poll(ctx)
	case "view":
		return l.view(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (l *LaunchCloudAgent) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := LaunchCloudAgentExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return err
	}

	if metadata.Agent == nil || metadata.Agent.ID == "" {
		// Wait for initial launch to finish.
		return ctx.Requests.ScheduleActionCall(pollActionName, map[string]any{}, pollInterval)
	}

	client, err := NewCloudAgentsClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	agent, err := client.GetAgent(metadata.Agent.ID)
	if err != nil {
		return err
	}

	if agent != nil {
		metadata.Agent = agent
		_ = ctx.Metadata.Set(metadata)
	}

	if agent == nil || agent.Status == "" {
		return ctx.Requests.ScheduleActionCall(pollActionName, map[string]any{}, pollInterval)
	}

	if agent.Status != agentStatusFinished && agent.Status != agentStatusError {
		return ctx.Requests.ScheduleActionCall(pollActionName, map[string]any{}, pollInterval)
	}

	// Synthesize a status payload for parity with webhook output.
	status := AgentStatusWebhook{
		Event:   webhookEventStatusChange,
		ID:      metadata.Agent.ID,
		Status:  agent.Status,
		Summary: agent.Summary,
	}
	if agent.Source != nil {
		status.Source = AgentSource{
			Repository: agent.Source.Repository,
			Ref:        agent.Source.Ref,
		}
	}
	status.Target = agent.Target

	metadata.Status = &status
	metadata.WebhookSecret = ""
	_ = ctx.Metadata.Set(metadata)

	return l.finishExecution(ctx.ExecutionState, AgentCompletedPayloadType, status, metadata.Agent)
}

func (l *LaunchCloudAgent) view(ctx core.ActionContext) error {
	// Best-effort: this action is used by UI to show helpful links; it should not fail executions.
	metadata := LaunchCloudAgentExecutionMetadata{}
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)

	links := map[string]any{}
	if metadata.Agent != nil {
		if metadata.Agent.Target != nil {
			if metadata.Agent.Target.URL != "" {
				links["agentUrl"] = metadata.Agent.Target.URL
			}
			if metadata.Agent.Target.PRURL != "" {
				links["prUrl"] = metadata.Agent.Target.PRURL
			}
			if metadata.Agent.Target.BranchName != "" {
				links["branchName"] = metadata.Agent.Target.BranchName
			}
		}
	}

	ctx.Logger.WithFields(logrus.Fields{"cursor": links}).Info("Cursor agent links")
	return nil
}

func (l *LaunchCloudAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signatureHeader := ctx.Headers.Get("X-Webhook-Signature")
	if signatureHeader == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	signature := strings.TrimPrefix(signatureHeader, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	payload := AgentStatusWebhook{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if payload.Event != webhookEventStatusChange || payload.ID == "" {
		return http.StatusOK, nil
	}

	executionCtx, err := ctx.FindExecutionByKV("agent_id", payload.ID)
	if err != nil || executionCtx == nil {
		return http.StatusOK, nil
	}

	metadata := LaunchCloudAgentExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %v", err)
	}

	if metadata.WebhookSecret == "" {
		// If the per-execution secret is already cleared (e.g., poll finished first),
		// we can't verify the signature. Ignore to avoid retries.
		return http.StatusOK, nil
	}

	if err := verifyCursorSignature([]byte(metadata.WebhookSecret), ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	// Only after authenticating the request do we check whether the execution is already finished.
	if executionCtx.ExecutionState.IsFinished() {
		return http.StatusOK, nil
	}

	metadata.Status = &payload
	metadata.Agent = updateAgentFromWebhook(metadata.Agent, payload)

	if payload.Status != agentStatusFinished && payload.Status != agentStatusError {
		if err := executionCtx.Metadata.Set(metadata); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error setting metadata: %v", err)
		}
		return http.StatusOK, nil
	}

	metadata.WebhookSecret = ""
	if err := executionCtx.Metadata.Set(metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error setting metadata: %v", err)
	}

	if err := l.finishExecution(executionCtx.ExecutionState, AgentCompletedPayloadType, payload, metadata.Agent); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (l *LaunchCloudAgent) finishExecution(state core.ExecutionStateContext, payloadType string, status AgentStatusWebhook, agent *LaunchAgentResponse) error {
	if status.Status == agentStatusError {
		return state.Fail(models.CanvasNodeExecutionResultReasonError, "Cursor agent reported an error")
	}

	payload := map[string]any{
		"status": status,
	}
	if agent != nil {
		payload["agent"] = agent
	}

	return state.Emit(core.DefaultOutputChannel.Name, payloadType, []any{payload})
}

func (l *LaunchCloudAgent) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (l *LaunchCloudAgent) validateSpec(spec LaunchCloudAgentSpec) error {
	if strings.TrimSpace(spec.Repository) == "" {
		return fmt.Errorf("repository is required")
	}
	if strings.TrimSpace(spec.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	return nil
}

func (l *LaunchCloudAgent) nodeWebhookURL(ctx core.ExecutionContext) (string, error) {
	workflowID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return "", fmt.Errorf("invalid workflow id")
	}

	node, err := models.FindCanvasNode(database.Conn(), workflowID, ctx.NodeID)
	if err != nil {
		return "", fmt.Errorf("failed to load node webhook: %v", err)
	}

	if node.WebhookID == nil {
		return "", fmt.Errorf("webhook not configured for node")
	}

	webhooksBaseURL := ctx.BaseURL
	meta := IntegrationMetadata{}
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &meta)
	if strings.TrimSpace(meta.WebhooksBaseURL) != "" {
		webhooksBaseURL = strings.TrimSpace(meta.WebhooksBaseURL)
	}

	return fmt.Sprintf("%s/api/v1/webhooks/%s", webhooksBaseURL, node.WebhookID.String()), nil
}

func newWebhookSecret() (string, error) {
	b := make([]byte, webhookSecretSize)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate webhook secret: %v", err)
	}
	return hex.EncodeToString(b), nil
}

func verifyCursorSignature(secret []byte, body []byte, signatureHex string) error {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)

	actual, err := hex.DecodeString(signatureHex)
	if err != nil {
		return err
	}

	if !hmac.Equal(expected, actual) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}
