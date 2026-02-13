package cursor

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type LaunchAgent struct{}

func (c *LaunchAgent) Name() string { return "cursor.launchAgent" }

func (c *LaunchAgent) Label() string { return "Launch Cloud Agent" }

func (c *LaunchAgent) Description() string {
	return "Launches a Cursor Cloud Agent to perform coding tasks asynchronously."
}

func (c *LaunchAgent) Documentation() string {
	return `The Launch Cloud Agent component triggers a Cursor AI coding agent and waits for it to complete.

## Use Cases
- **Automated code generation**: Generate code from natural language prompts
- **PR fixes**: Automatically fix issues on existing pull requests
- **Code refactoring**: Refactor code based on instructions
- **Feature implementation**: Implement new features from specifications

## How It Works
1. Launches a Cursor Cloud Agent with the specified prompt and configuration
2. Waits for the agent to complete (monitored via webhook and polling)
3. Emits output with the agent result (success or failure)`
}

func (c *LaunchAgent) Icon() string { return "cpu" }

func (c *LaunchAgent) Color() string { return "#8B5CF6" }

func (c *LaunchAgent) ExampleOutput() map[string]any {
	return map[string]any{
		"status":     LaunchAgentStatusDone,
		"agentId":    "agent_12345",
		"summary":    "Refactored login logic.",
		"prUrl":      "https://github.com/org/repo/pull/42",
		"branchName": "cursor/agent-550e8400",
	}
}

func (c *LaunchAgent) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{{Name: LaunchAgentDefaultChannel, Label: "Default"}}
}

func (c *LaunchAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name: "prompt", Label: "Instructions", Type: configuration.FieldTypeText, Description: "What should the agent do?", Required: true,
		},
		{
			Name: "model", Label: "Model", Type: configuration.FieldTypeIntegrationResource, Required: false,
			TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: "model"}},
		},
		{
			Name: "sourceMode", Label: "Source", Type: configuration.FieldTypeSelect, Required: true, Default: "repository",
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
			Name: "repository", Label: "Repository URL", Type: configuration.FieldTypeString, Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "sourceMode", Values: []string{"repository"}}},
		},
		{
			Name: "branch", Label: "Base Branch", Type: configuration.FieldTypeString, Required: false, Default: LaunchAgentDefaultBranch,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "sourceMode", Values: []string{"repository"}}},
		},
		{
			Name: "prUrl", Label: "Existing PR URL", Type: configuration.FieldTypeString, Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "sourceMode", Values: []string{"pr"}}},
		},
		{
			Name: "autoCreatePr", Label: "Auto Create PR", Type: configuration.FieldTypeBool, Required: false, Default: true,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "sourceMode", Values: []string{"repository"}}},
		},
		{
			Name: "useCursorBot", Label: "Act as Cursor Bot", Type: configuration.FieldTypeBool, Required: false, Default: true,
		},
	}
}

func (c *LaunchAgent) Setup(ctx core.SetupContext) error {
	spec := LaunchAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	// Basic validation ensuring the mode matches the inputs
	if spec.SourceMode == "repository" && spec.Repository == "" {
		return fmt.Errorf("repository URL is required when using repository mode")
	} else if spec.SourceMode == "pr" && spec.PrURL == "" {
		return fmt.Errorf("PR URL is required when using PR mode")
	}

	// Set up webhook so it's associated with the node and saved
	_, err := ctx.Webhook.Setup()
	return err
}

func (c *LaunchAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *LaunchAgent) Execute(ctx core.ExecutionContext) error {
	spec := LaunchAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// 1. Prepare Configuration
	if spec.Branch == "" {
		spec.Branch = LaunchAgentDefaultBranch
	}
	if spec.SourceMode == "repository" && spec.Repository == "" {
		return fmt.Errorf("repository URL is required")
	} else if spec.SourceMode == "pr" && spec.PrURL == "" {
		return fmt.Errorf("PR URL is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create cursor client: %w", err)
	}
	if client.LaunchAgentKey == "" {
		return fmt.Errorf("cloud agent API key is not configured")
	}

	// Get webhook URL and secret (webhook should already be set up in Setup)
	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to get webhook URL: %w", err)
	}

	webhookSecret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return fmt.Errorf("failed to get webhook secret: %w", err)
	}

	branchName := fmt.Sprintf("%s%s", LaunchAgentBranchPrefix, ctx.ID.String()[:8])

	// 3. Construct API Payload
	source := launchAgentSource{}
	target := launchAgentTarget{
		AutoCreatePr:          ptrFromBool(spec.AutoCreatePr),
		OpenAsCursorGithubApp: ptrFromBool(spec.UseCursorBot),
		BranchName:            branchName,
		SkipReviewerRequest:   ptrFromBool(LaunchAgentSkipReviewerRequest),
	}

	if spec.SourceMode == "pr" {
		source.PrURL = spec.PrURL
		autoBranch := false
		target.AutoBranch = &autoBranch
	} else {
		source.Repository = spec.Repository
		source.Ref = spec.Branch
	}

	payload := launchAgentRequest{
		Prompt:  launchAgentPrompt{Text: spec.Prompt},
		Source:  source,
		Target:  target,
		Webhook: launchAgentWebhook{URL: webhookURL, Secret: string(webhookSecret)},
	}
	if spec.Model != "" {
		payload.Model = spec.Model
	}

	// 4. Trigger External Job
	result, err := client.LaunchAgent(payload)
	if err != nil {
		return fmt.Errorf("failed to launch cursor agent: %w", err)
	}

	// 5. Initialize State
	metadata := LaunchAgentExecutionMetadata{
		Agent:  &AgentMetadata{ID: result.ID, Name: result.Name, Status: result.Status},
		Target: &TargetMetadata{BranchName: branchName},
		Source: &SourceMetadata{Repository: spec.Repository, Ref: spec.Branch},
	}

	// Populate additional target details if available
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

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	// Set KV for Webhook correlation
	if err := ctx.ExecutionState.SetKV("agent_id", result.ID); err != nil {
		return fmt.Errorf("failed to set agent_id in KV: %w", err)
	}

	ctx.Logger.Infof("Launched Cursor Agent %s. Waiting for completion...", result.ID)

	// 6. Start Monitoring (Fallback Polling)
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": 1, "errors": 0}, LaunchAgentInitialPollInterval)
}

func ptrFromBool(b bool) *bool { return &b }
