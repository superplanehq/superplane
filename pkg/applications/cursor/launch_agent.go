package cursor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
)

const AgentStatusPayloadType = "cursor.agent.status_change"
const webhookEventStatusChange = "statusChange"
const agentStatusFinished = "FINISHED"
const agentStatusError = "ERROR"

type LaunchAgent struct{}

type LaunchAgentSpec struct {
	Prompt       string `json:"prompt"`
	Repository   string `json:"repository"`
	Ref          string `json:"ref"`
	AutoCreatePr bool   `json:"autoCreatePr"`
	BranchName   string `json:"branchName"`
}

type LaunchAgentExecutionMetadata struct {
	Agent  *LaunchAgentResponse `json:"agent,omitempty"`
	Status *AgentStatusWebhook  `json:"status,omitempty"`
}

type AgentStatusPayload struct {
	Agent  *LaunchAgentResponse `json:"agent,omitempty"`
	Status *AgentStatusWebhook  `json:"status,omitempty"`
}

type AgentStatusWebhook struct {
	Event     string             `json:"event"`
	Timestamp string             `json:"timestamp"`
	ID        string             `json:"id"`
	Status    string             `json:"status"`
	Source    LaunchAgentSource  `json:"source"`
	Target    *AgentStatusTarget `json:"target,omitempty"`
	Summary   string             `json:"summary,omitempty"`
}

type AgentStatusTarget struct {
	URL        string `json:"url,omitempty"`
	BranchName string `json:"branchName,omitempty"`
	PRURL      string `json:"prUrl,omitempty"`
}

func (l *LaunchAgent) Name() string {
	return "cursor.launch_agent"
}

func (l *LaunchAgent) Label() string {
	return "Launch Agent"
}

func (l *LaunchAgent) Description() string {
	return "Start a Cursor Cloud Agent for a repository"
}

func (l *LaunchAgent) Icon() string {
	return "bot"
}

func (l *LaunchAgent) Color() string {
	return "gray"
}

func (l *LaunchAgent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *LaunchAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "prompt",
			Label:       "Prompt",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Describe the task for the agent",
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "https://github.com/org/repo",
		},
		{
			Name:        "ref",
			Label:       "Git ref",
			Type:        configuration.FieldTypeGitRef,
			Required:    true,
			Placeholder: "main",
		},
		{
			Name:        "autoCreatePr",
			Label:       "Auto-create PR",
			Type:        configuration.FieldTypeBool,
			Description: "Create a pull request when the agent completes",
		},
		{
			Name:        "branchName",
			Label:       "Branch name",
			Type:        configuration.FieldTypeString,
			Placeholder: "feature/add-readme",
		},
	}
}

func (l *LaunchAgent) Setup(ctx core.SetupContext) error {
	spec := LaunchAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if err := l.validateSpec(spec); err != nil {
		return err
	}

	if ctx.AppInstallation != nil {
		if err := ctx.AppInstallation.RequestWebhook(WebhookConfiguration{Event: webhookEventStatusChange}); err != nil {
			return err
		}
	}

	return nil
}

func (l *LaunchAgent) Execute(ctx core.ExecutionContext) error {
	spec := LaunchAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if err := l.validateSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return err
	}

	webhook, err := l.lookupWebhook(ctx)
	if err != nil {
		return err
	}

	request := LaunchAgentRequest{
		Prompt: LaunchAgentPrompt{Text: spec.Prompt},
		Source: LaunchAgentSource{
			Repository: spec.Repository,
			Ref:        spec.Ref,
		},
		Target:  buildLaunchAgentTarget(spec),
		Webhook: webhook,
	}

	response, err := client.LaunchAgent(request)
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(LaunchAgentExecutionMetadata{Agent: response}); err != nil {
		return err
	}

	return ctx.ExecutionState.SetKV("agent", response.ID)
}

func (l *LaunchAgent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *LaunchAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *LaunchAgent) Actions() []core.Action {
	return []core.Action{}
}

func (l *LaunchAgent) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (l *LaunchAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Webhook-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	payload := AgentStatusWebhook{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if payload.Event != webhookEventStatusChange || payload.ID == "" {
		return http.StatusOK, nil
	}

	executionCtx, err := ctx.FindExecutionByKV("agent", payload.ID)
	if err != nil {
		return http.StatusOK, nil
	}

	if executionCtx.ExecutionState.IsFinished() {
		return http.StatusOK, nil
	}

	metadata := LaunchAgentExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %v", err)
	}

	metadata.Status = &payload
	if err := executionCtx.Metadata.Set(metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error setting metadata: %v", err)
	}

	switch payload.Status {
	case agentStatusFinished:
		return http.StatusOK, executionCtx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			AgentStatusPayloadType,
			[]any{AgentStatusPayload{Agent: metadata.Agent, Status: &payload}},
		)
	case agentStatusError:
		return http.StatusOK, executionCtx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			"Cursor agent reported an error",
		)
	}

	return http.StatusOK, nil
}

func (l *LaunchAgent) validateSpec(spec LaunchAgentSpec) error {
	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if spec.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if spec.Ref == "" {
		return fmt.Errorf("ref is required")
	}

	return nil
}

func buildLaunchAgentTarget(spec LaunchAgentSpec) *LaunchAgentTarget {
	if !spec.AutoCreatePr && spec.BranchName == "" {
		return nil
	}

	return &LaunchAgentTarget{
		AutoCreatePr: spec.AutoCreatePr,
		BranchName:   spec.BranchName,
	}
}

func (l *LaunchAgent) lookupWebhook(ctx core.ExecutionContext) (*LaunchAgentWebhook, error) {
	if ctx.AppInstallation == nil {
		return nil, fmt.Errorf("app installation is required")
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode app metadata: %v", err)
	}

	if metadata.Webhook == nil || metadata.Webhook.URL == "" || metadata.Webhook.Secret == "" {
		return nil, fmt.Errorf("cursor webhook is not ready yet")
	}

	return &LaunchAgentWebhook{
		URL:    metadata.Webhook.URL,
		Secret: metadata.Webhook.Secret,
	}, nil
}
