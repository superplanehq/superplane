package cursor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CloudAgent struct{}

func (c *CloudAgent) Name() string {
	return "cursor.cloudAgent"
}

func (c *CloudAgent) Label() string {
	return "Launch Cloud Agent"
}

func (c *CloudAgent) Description() string {
	return "Launches a Cursor Cloud Agent to perform coding tasks asynchronously."
}

func (c *CloudAgent) Documentation() string {
	return `The Launch Cloud Agent component triggers a Cursor AI coding agent and waits for it to complete.

## Use Cases

- **Automated code generation**: Generate code from natural language prompts
- **PR fixes**: Automatically fix issues on existing pull requests
- **Code refactoring**: Refactor code based on instructions
- **Feature implementation**: Implement new features from specifications

## How It Works

1. Launches a Cursor Cloud Agent with the specified prompt and configuration
2. Waits for the agent to complete (monitored via webhook and polling)
3. Routes execution based on agent result:
   - **Passed channel**: Agent completed successfully
   - **Failed channel**: Agent failed or encountered an error

## Configuration

- **Instructions**: The prompt/instruction for the agent (required)
- **Model**: The LLM to use (Claude 3.5 Sonnet, GPT-4o, or o1-mini)
- **Repository URL**: Full URL of the repository to work on (required if PR URL is not set)
- **Base Branch**: Branch to start from (default: main, ignored if PR URL is set)
- **Existing PR URL**: If set, the agent will fix this specific PR instead of creating a new branch
- **Auto Create PR**: Whether to automatically open a Pull Request when finished
- **Act as Cursor Bot**: If true, the PR is opened by the Cursor GitHub App

## Output Channels

- **Passed**: Emitted when the agent completes successfully
- **Failed**: Emitted when the agent fails or encounters an error

## Notes

- Requires a valid Cursor Cloud Agent API key configured in the integration
- The component automatically generates unique branch names to prevent collisions
- Falls back to polling if webhook doesn't arrive`
}

func (c *CloudAgent) Icon() string {
	return "cpu"
}

func (c *CloudAgent) Color() string {
	return "#8B5CF6"
}

func (c *CloudAgent) ExampleOutput() map[string]any {
	return map[string]any{
		"status":     CloudAgentStatusDone,
		"agentId":    "agent_12345",
		"summary":    "Refactored login logic.",
		"prUrl":      "https://github.com/org/repo/pull/42",
		"branchName": "cursor/agent-550e8400",
	}
}

func (c *CloudAgent) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  CloudAgentPassedChannel,
			Label: "Passed",
		},
		{
			Name:  CloudAgentFailedChannel,
			Label: "Failed",
		},
	}
}

func (c *CloudAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "prompt",
			Label:       "Instructions",
			Type:        configuration.FieldTypeText,
			Description: "What should the agent do? (e.g., 'Fix the bug in auth.go')",
			Required:    true,
		},
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Description: "The LLM to use. Auto lets Cursor pick the best model.",
			Required:    false,
			Default:     "",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "model",
				},
			},
		},
		{
			Name:        "sourceMode",
			Label:       "Source",
			Type:        configuration.FieldTypeSelect,
			Description: "Choose how to specify the code to work on",
			Required:    true,
			Default:     "repository",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "New Task (Repository + Branch)", Value: "repository"},
						{Label: "Fix Existing PR", Value: "pr"},
					},
				},
			},
		},
		{
			Name:        "repository",
			Label:       "Repository URL",
			Type:        configuration.FieldTypeString,
			Description: "Full URL (e.g., https://github.com/org/repo)",
			Required:    false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceMode", Values: []string{"repository"}},
			},
		},
		{
			Name:        "branch",
			Label:       "Base Branch",
			Type:        configuration.FieldTypeString,
			Description: "Branch to start from (e.g., main)",
			Required:    false,
			Default:     CloudAgentDefaultBranch,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceMode", Values: []string{"repository"}},
			},
		},
		{
			Name:        "prUrl",
			Label:       "Existing PR URL",
			Type:        configuration.FieldTypeString,
			Description: "The agent will fix this specific PR",
			Required:    false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceMode", Values: []string{"pr"}},
			},
		},
		{
			Name:        "autoCreatePr",
			Label:       "Auto Create PR",
			Type:        configuration.FieldTypeBool,
			Description: "Should the agent open a Pull Request when finished?",
			Required:    false,
			Default:     true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceMode", Values: []string{"repository"}},
			},
		},
		{
			Name:        "useCursorBot",
			Label:       "Act as Cursor Bot",
			Type:        configuration.FieldTypeBool,
			Description: "If true, the PR is opened by the Cursor GitHub App.",
			Required:    false,
			Default:     true,
		},
	}
}

func (c *CloudAgent) Setup(ctx core.SetupContext) error {
	spec := CloudAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if spec.SourceMode == "repository" {
		if spec.Repository == "" {
			return fmt.Errorf("repository URL is required when using repository mode")
		}
		if err := validateURL(spec.Repository, "repository URL"); err != nil {
			return err
		}
	} else if spec.SourceMode == "pr" {
		if spec.PrURL == "" {
			return fmt.Errorf("PR URL is required when using PR mode")
		}
		if err := validateURL(spec.PrURL, "PR URL"); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("invalid source mode: %s", spec.SourceMode)
	}

	return ctx.Integration.RequestWebhook(nil)
}

func (c *CloudAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CloudAgent) Execute(ctx core.ExecutionContext) error {
	spec := CloudAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if spec.Branch == "" {
		spec.Branch = CloudAgentDefaultBranch
	}

	if spec.SourceMode == "repository" {
		if spec.Repository == "" {
			return fmt.Errorf("repository URL is required when using repository mode")
		}
		if err := validateURL(spec.Repository, "repository URL"); err != nil {
			return err
		}
	} else if spec.SourceMode == "pr" {
		if spec.PrURL == "" {
			return fmt.Errorf("PR URL is required when using PR mode")
		}
		if err := validateURL(spec.PrURL, "PR URL"); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("invalid source mode: %s", spec.SourceMode)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create cursor client: %w", err)
	}

	if client.CloudAgentKey == "" {
		return fmt.Errorf("cloud agent API key is not configured in the integration")
	}

	branchName := fmt.Sprintf("%s%s", CloudAgentBranchPrefix, ctx.ID.String()[:8])

	webhookSecret := uuid.New().String()
	webhookURL := fmt.Sprintf("%s/api/v1/integrations/%s/webhook", ctx.BaseURL, ctx.Integration.ID().String())

	source := cloudAgentSource{}
	autoCreatePr := spec.AutoCreatePr
	openAsCursorGithubApp := spec.UseCursorBot
	skipReviewerRequest := CloudAgentSkipReviewerRequest
	target := cloudAgentTarget{
		AutoCreatePr:          &autoCreatePr,
		OpenAsCursorGithubApp: &openAsCursorGithubApp,
		BranchName:            branchName,
		SkipReviewerRequest:   &skipReviewerRequest,
	}

	if spec.SourceMode == "pr" {
		source.PrURL = spec.PrURL
		autoBranch := false
		target.AutoBranch = &autoBranch
	} else {
		source.Repository = spec.Repository
		source.Ref = spec.Branch
	}
	payload := cloudAgentRequest{
		Prompt: cloudAgentPrompt{Text: spec.Prompt},
		Source: source,
		Target: target,
		Webhook: cloudAgentWebhook{
			URL:    webhookURL,
			Secret: webhookSecret,
		},
	}
	if spec.Model != "" {
		payload.Model = spec.Model
	}

	result, err := client.LaunchAgent(payload)
	if err != nil {
		return fmt.Errorf("failed to launch cursor agent: %w", err)
	}
	metadata := CloudAgentExecutionMetadata{
		Agent: &AgentMetadata{
			ID:     result.ID,
			Name:   result.Name,
			Status: result.Status,
		},
		Target: &TargetMetadata{
			BranchName: branchName,
		},
		Source: &SourceMetadata{
			Repository: spec.Repository,
			Ref:        spec.Branch,
		},
		WebhookSecret: webhookSecret,
	}

	if result.Target != nil {
		if result.Target.URL != "" {
			metadata.Agent.URL = result.Target.URL
		}
		if result.Target.PrURL != "" {
			metadata.Target.PrURL = result.Target.PrURL
		}
		if result.Target.BranchName != "" {
			metadata.Target.BranchName = result.Target.BranchName
		}
	}

	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	err = ctx.ExecutionState.SetKV("agent_id", result.ID)
	if err != nil {
		return fmt.Errorf("failed to set agent_id in KV: %w", err)
	}

	ctx.Logger.Infof("Launched Cursor Agent %s. Waiting for completion...", result.ID)
	pollParams := map[string]any{
		"attempt": 1,
		"errors":  0,
	}
	return ctx.Requests.ScheduleActionCall("poll", pollParams, CloudAgentInitialPollInterval)
}

func (c *CloudAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var payload cloudAgentWebhookPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid json body: %w", err)
	}

	if payload.ID == "" {
		return http.StatusBadRequest, fmt.Errorf("id missing from webhook payload")
	}

	executionCtx, err := ctx.FindExecutionByKV("agent_id", payload.ID)
	if err != nil {
		return http.StatusOK, nil
	}

	metadata := CloudAgentExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode metadata: %w", err)
	}
	if metadata.WebhookSecret != "" {
		signature := ctx.Headers.Get(CloudAgentWebhookSignatureHeader)
		if signature == "" {
			executionCtx.Logger.Warnf("Missing webhook signature for Agent %s", payload.ID)
			return http.StatusUnauthorized, fmt.Errorf("missing webhook signature")
		}
		if !verifyWebhookSignature(ctx.Body, signature, metadata.WebhookSecret) {
			executionCtx.Logger.Warnf("Invalid webhook signature for Agent %s", payload.ID)
			return http.StatusUnauthorized, fmt.Errorf("invalid webhook signature")
		}
	}
	if metadata.Agent != nil && isTerminalStatus(metadata.Agent.Status) {
		return http.StatusOK, nil
	}

	executionCtx.Logger.Infof("Received webhook for Agent %s: %s", payload.ID, payload.Status)
	if metadata.Agent == nil {
		metadata.Agent = &AgentMetadata{}
	}
	metadata.Agent.ID = payload.ID
	metadata.Agent.Status = payload.Status
	metadata.Agent.Summary = payload.Summary

	if metadata.Target == nil {
		metadata.Target = &TargetMetadata{}
	}
	if payload.PrURL != "" {
		metadata.Target.PrURL = payload.PrURL
	}

	if err := executionCtx.Metadata.Set(metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to set metadata: %w", err)
	}

	branchName := ""
	if metadata.Target != nil {
		branchName = metadata.Target.BranchName
	}

	outputPayload := buildOutputPayload(payload.Status, payload.ID, payload.PrURL, payload.Summary, branchName)

	if isSuccessStatus(payload.Status) {
		err = executionCtx.ExecutionState.Emit(CloudAgentPassedChannel, CloudAgentPayloadType, []any{outputPayload})
	} else if isFailureStatus(payload.Status) {
		err = executionCtx.ExecutionState.Emit(CloudAgentFailedChannel, CloudAgentPayloadType, []any{outputPayload})
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (c *CloudAgent) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *CloudAgent) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *CloudAgent) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := CloudAgentExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Agent == nil || metadata.Agent.ID == "" {
		return nil
	}

	if isTerminalStatus(metadata.Agent.Status) {
		return nil
	}

	pollAttempt := 1
	pollErrors := 0
	if attempt, ok := ctx.Parameters["attempt"].(float64); ok {
		pollAttempt = int(attempt)
	}
	if errors, ok := ctx.Parameters["errors"].(float64); ok {
		pollErrors = int(errors)
	}

	if pollAttempt > CloudAgentMaxPollAttempts {
		ctx.Logger.Errorf("Agent %s exceeded maximum poll attempts (%d). Failing execution.", metadata.Agent.ID, CloudAgentMaxPollAttempts)

		branchName := ""
		if metadata.Target != nil {
			branchName = metadata.Target.BranchName
		}

		outputPayload := buildOutputPayload("timeout", metadata.Agent.ID, "", "Agent polling timed out after maximum attempts", branchName)
		return ctx.ExecutionState.Emit(CloudAgentFailedChannel, CloudAgentPayloadType, []any{outputPayload})
	}

	ctx.Logger.Infof("Polling for Agent %s status (attempt %d/%d)...", metadata.Agent.ID, pollAttempt, CloudAgentMaxPollAttempts)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Errorf("Failed to create client for polling: %v", err)
		return c.scheduleNextPoll(ctx, pollAttempt+1, pollErrors)
	}

	agentStatus, err := client.GetAgentStatus(metadata.Agent.ID)
	if err != nil {
		ctx.Logger.Errorf("Failed to get agent status: %v", err)

		pollErrors++

		if pollErrors >= CloudAgentMaxPollErrors {
			ctx.Logger.Errorf("Agent %s exceeded maximum consecutive poll errors (%d). Failing execution.", metadata.Agent.ID, CloudAgentMaxPollErrors)

			branchName := ""
			if metadata.Target != nil {
				branchName = metadata.Target.BranchName
			}

			outputPayload := buildOutputPayload("error", metadata.Agent.ID, "", fmt.Sprintf("Failed to poll agent status after %d consecutive errors", pollErrors), branchName)
			return ctx.ExecutionState.Emit(CloudAgentFailedChannel, CloudAgentPayloadType, []any{outputPayload})
		}

		return c.scheduleNextPoll(ctx, pollAttempt+1, pollErrors)
	}

	pollErrors = 0
	metadata.Agent.Status = agentStatus.Status
	metadata.Agent.Summary = agentStatus.Summary
	if agentStatus.Target != nil {
		if metadata.Target == nil {
			metadata.Target = &TargetMetadata{}
		}
		if agentStatus.Target.URL != "" {
			metadata.Agent.URL = agentStatus.Target.URL
		}
		if agentStatus.Target.PrURL != "" {
			metadata.Target.PrURL = agentStatus.Target.PrURL
		}
		if agentStatus.Target.BranchName != "" {
			metadata.Target.BranchName = agentStatus.Target.BranchName
		}
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		ctx.Logger.Errorf("Failed to update metadata: %v", err)
	}

	// Get branch and PR URL for output payload
	branchName := ""
	prURL := ""
	if metadata.Target != nil {
		branchName = metadata.Target.BranchName
		prURL = metadata.Target.PrURL
	}

	// Check if finished
	if isSuccessStatus(agentStatus.Status) {
		outputPayload := buildOutputPayload(agentStatus.Status, metadata.Agent.ID, prURL, agentStatus.Summary, branchName)
		return ctx.ExecutionState.Emit(CloudAgentPassedChannel, CloudAgentPayloadType, []any{outputPayload})
	}

	if isFailureStatus(agentStatus.Status) {
		outputPayload := buildOutputPayload(agentStatus.Status, metadata.Agent.ID, prURL, agentStatus.Summary, branchName)
		return ctx.ExecutionState.Emit(CloudAgentFailedChannel, CloudAgentPayloadType, []any{outputPayload})
	}

	return c.scheduleNextPoll(ctx, pollAttempt+1, pollErrors)
}

// scheduleNextPoll schedules the next poll with exponential backoff
func (c *CloudAgent) scheduleNextPoll(ctx core.ActionContext, nextAttempt, errors int) error {
	interval := CloudAgentInitialPollInterval * time.Duration(1<<uint(min(nextAttempt-1, 8)))
	if interval > CloudAgentMaxPollInterval {
		interval = CloudAgentMaxPollInterval
	}

	pollParams := map[string]any{
		"attempt": nextAttempt,
		"errors":  errors,
	}
	return ctx.Requests.ScheduleActionCall("poll", pollParams, interval)
}

func (c *CloudAgent) Cancel(ctx core.ExecutionContext) error {
	metadata := CloudAgentExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		ctx.Logger.Warnf("Failed to decode metadata for cancellation: %v", err)
		return nil
	}

	if metadata.Agent == nil || metadata.Agent.ID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Warnf("Failed to create client for cancellation: %v", err)
		return nil
	}

	if err := client.CancelAgent(metadata.Agent.ID); err != nil {
		ctx.Logger.Warnf("Failed to cancel Cursor Agent %s: %v", metadata.Agent.ID, err)
		return nil
	}

	ctx.Logger.Infof("Cancelled Cursor Agent %s", metadata.Agent.ID)
	return nil
}

func (c *CloudAgent) Cleanup(ctx core.SetupContext) error {
	return nil
}
