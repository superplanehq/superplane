package cursor

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const LaunchAgentPayloadType = "cursor.agent.launched"

type LaunchAgent struct{}

type LaunchAgentSpec struct {
	Prompt                string `json:"prompt"`
	Repository            string `json:"repository"`
	Ref                   string `json:"ref"`
	AutoCreatePr          bool   `json:"autoCreatePr"`
	OpenAsCursorGithubApp bool   `json:"openAsCursorGithubApp"`
	SkipReviewerRequest   bool   `json:"skipReviewerRequest"`
	BranchName            string `json:"branchName"`
	WebhookURL            string `json:"webhookUrl"`
	WebhookSecret         string `json:"webhookSecret"`
}

type LaunchAgentPayload struct {
	Agent *LaunchAgentResponse `json:"agent"`
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
			Name:        "openAsCursorGithubApp",
			Label:       "Open PR as Cursor GitHub App",
			Type:        configuration.FieldTypeBool,
			Description: "Only applies when auto-create PR is enabled",
		},
		{
			Name:        "skipReviewerRequest",
			Label:       "Skip reviewer request",
			Type:        configuration.FieldTypeBool,
			Description: "Skip adding the user as a reviewer when using the Cursor GitHub App",
		},
		{
			Name:        "branchName",
			Label:       "Branch name",
			Type:        configuration.FieldTypeString,
			Placeholder: "feature/add-readme",
		},
		{
			Name:        "webhookUrl",
			Label:       "Webhook URL",
			Type:        configuration.FieldTypeString,
			Description: "Optional URL to receive agent status change notifications",
		},
		{
			Name:        "webhookSecret",
			Label:       "Webhook secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Optional secret for webhook payload verification",
		},
	}
}

func (l *LaunchAgent) Setup(ctx core.SetupContext) error {
	spec := LaunchAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	return l.validateSpec(spec)
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

	request := LaunchAgentRequest{
		Prompt: LaunchAgentPrompt{Text: spec.Prompt},
		Source: LaunchAgentSource{
			Repository: spec.Repository,
			Ref:        spec.Ref,
		},
		Target:  buildLaunchAgentTarget(spec),
		Webhook: buildLaunchAgentWebhook(spec),
	}

	response, err := client.LaunchAgent(request)
	if err != nil {
		return err
	}

	payload := LaunchAgentPayload{Agent: response}
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		LaunchAgentPayloadType,
		[]any{payload},
	)
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

	if spec.WebhookURL != "" && len(spec.WebhookSecret) > 0 && len(spec.WebhookSecret) < 32 {
		return fmt.Errorf("webhookSecret must be at least 32 characters")
	}

	if spec.WebhookURL == "" && spec.WebhookSecret != "" {
		return fmt.Errorf("webhookUrl is required when webhookSecret is set")
	}

	return nil
}

func buildLaunchAgentTarget(spec LaunchAgentSpec) *LaunchAgentTarget {
	if !spec.AutoCreatePr && !spec.OpenAsCursorGithubApp && !spec.SkipReviewerRequest && spec.BranchName == "" {
		return nil
	}

	return &LaunchAgentTarget{
		AutoCreatePr:          spec.AutoCreatePr,
		OpenAsCursorGithubApp: spec.OpenAsCursorGithubApp,
		SkipReviewerRequest:   spec.SkipReviewerRequest,
		BranchName:            spec.BranchName,
	}
}

func buildLaunchAgentWebhook(spec LaunchAgentSpec) *LaunchAgentWebhook {
	if spec.WebhookURL == "" {
		return nil
	}

	return &LaunchAgentWebhook{
		URL:    spec.WebhookURL,
		Secret: spec.WebhookSecret,
	}
}
