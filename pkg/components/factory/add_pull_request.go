package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const AddPullRequestComponentName = "addPullRequest"

func init() {
	registry.RegisterAction(AddPullRequestComponentName, &AddPullRequest{})
}

type AddPullRequest struct{}

type AddPullRequestConfiguration struct {
	OrderID    string `json:"orderId" mapstructure:"orderId"`
	Provider   string `json:"provider" mapstructure:"provider"`
	ExternalID string `json:"externalId" mapstructure:"externalId"`
	Repository string `json:"repository" mapstructure:"repository"`
	Number     string `json:"number" mapstructure:"number"`
	URL        string `json:"url" mapstructure:"url"`
	Title      string `json:"title" mapstructure:"title"`
	State      any    `json:"state,omitempty" mapstructure:"state,omitempty"`
	Merged     any    `json:"merged,omitempty" mapstructure:"merged,omitempty"`
	Draft      any    `json:"draft,omitempty" mapstructure:"draft,omitempty"`
	MergedAt   string `json:"mergedAt" mapstructure:"mergedAt"`
	ClosedAt   string `json:"closedAt" mapstructure:"closedAt"`
}

func (c *AddPullRequest) Name() string {
	return AddPullRequestComponentName
}

func (c *AddPullRequest) Label() string {
	return "Add Pull Request"
}

func (c *AddPullRequest) Description() string {
	return "Attach a pull request to a task"
}

func (c *AddPullRequest) Documentation() string {
	return `The Add Pull Request component records a pull request on a task.

Required fields are ` + "`repository`" + `, ` + "`number`" + `, and ` + "`url`" + `. ` + "`provider`" + ` defaults to ` + "`github`" + `. ` + "`state`" + ` defaults to ` + "`open`" + `.

` + "`state`" + `, ` + "`merged`" + `, and ` + "`draft`" + ` accept expressions, so you can pass a GitHub webhook payload through as-is. A GitHub-shaped ` + "`state: \"closed\"`" + ` + ` + "`merged: true`" + ` becomes SuperPlane state ` + "`merged`" + `.

Set ` + "`mergedAt`" + ` or ` + "`closedAt`" + ` (RFC3339) when the pull request is already merged or closed so Velocity uses the real day.

` + "`orderId`" + ` defaults to ` + "`{{ order().id }}`" + `. In a flow that is not dispatched from a factory line, replace it with the task id from a previous step. This component can only be used in factory-owned apps.`
}

func (c *AddPullRequest) Icon() string {
	return "factory"
}

func (c *AddPullRequest) Color() string {
	return "blue"
}

func (c *AddPullRequest) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.pullRequestAdded",
		"data": map[string]any{
			"pullRequest": map[string]any{
				"id":          "2b3cf24d-c0e2-4d42-bbe7-4c30ff2cb2a4",
				"workOrderId": "9ac921bc-68e4-46dc-a78b-60b955f3bbf2",
				"provider":    "github",
				"repository":  "acme/app",
				"number":      42,
				"url":         "https://github.com/acme/app/pull/42",
				"title":       "Fix retry handling",
				"state":       "open",
			},
		},
	}
}

func (c *AddPullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddPullRequest) Configuration() []configuration.Field {
	fields := []configuration.Field{
		{
			Name:        "orderId",
			Label:       "Task ID",
			Description: "Task to target. Defaults to the task driving the current run.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "{{ order().id }}",
		},
		{
			Name:        "provider",
			Label:       "Provider",
			Description: "Source control provider. Defaults to github.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "github",
		},
		{
			Name:        "externalId",
			Label:       "External ID",
			Description: "Provider pull request id, when the event includes one.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Description: "Repository in owner/name format.",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "number",
			Label:       "Number",
			Description: "Pull request number.",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "url",
			Label:       "URL",
			Description: "Absolute http(s) pull request URL.",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "title",
			Label:       "Title",
			Description: "Pull request title.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
	}

	fields = append(fields, prArtifactLifecycleFields(prArtifactLifecycleFieldOptions{
		StateDefault:   "open",
		StateTogglable: false,
	})...)

	return append(fields,
		configuration.Field{
			Name:        "mergedAt",
			Label:       "Merged At",
			Description: "Optional RFC3339 merge timestamp. Unset falls back to now when state is merged.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		configuration.Field{
			Name:        "closedAt",
			Label:       "Closed At",
			Description: "Optional RFC3339 close timestamp. Unset falls back to now when state is closed.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
	)
}

func (c *AddPullRequest) Execute(ctx core.ExecutionContext) error {
	config := AddPullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	number, err := parseOptionalInt64(config.Number)
	if err != nil {
		return err
	}
	mergedAt, err := parseOptionalRFC3339(config.MergedAt)
	if err != nil {
		return err
	}
	closedAt, err := parseOptionalRFC3339(config.ClosedAt)
	if err != nil {
		return err
	}

	state := resolvePrArtifactState(config.State, config.Merged, config.Draft)
	if err := validatePrArtifactState(state); err != nil {
		return err
	}

	pullRequest, err := ctx.Factory.AddPullRequest(core.AddPullRequestParams{
		OrderID:    config.OrderID,
		Provider:   config.Provider,
		ExternalID: config.ExternalID,
		Repository: config.Repository,
		Number:     number,
		URL:        config.URL,
		Title:      config.Title,
		State:      state,
		MergedAt:   mergedAt,
		ClosedAt:   closedAt,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.pullRequestAdded",
		[]any{map[string]any{
			"pullRequest": pullRequest,
		}},
	)
}

func (c *AddPullRequest) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *AddPullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddPullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddPullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddPullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddPullRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
