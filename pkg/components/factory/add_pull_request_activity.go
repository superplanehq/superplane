package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const AddPullRequestActivityComponentName = "addPullRequestActivity"

func init() {
	registry.RegisterAction(AddPullRequestActivityComponentName, &AddPullRequestActivity{})
}

type AddPullRequestActivity struct{}

type AddPullRequestActivityConfiguration struct {
	PullRequestID string `json:"pullRequestId" mapstructure:"pullRequestId"`
	Description   string `json:"description" mapstructure:"description"`
}

func (c *AddPullRequestActivity) Name() string {
	return AddPullRequestActivityComponentName
}

func (c *AddPullRequestActivity) Label() string {
	return "Add Pull Request Activity"
}

func (c *AddPullRequestActivity) Description() string {
	return "Link the current run to a factory pull request"
}

func (c *AddPullRequestActivity) Documentation() string {
	return `The Add Pull Request Activity component links the current canvas run to a factory pull request. An existing relation is treated as success.

Set ` + "`pullRequestId`" + ` from Find Pull Request output. Use ` + "`description`" + ` to record why this run is linked. The field accepts event data, for example ` + "`{{ root().data.comment.body }}`" + `. The output contains the same pull request and work order references as Find Pull Request.

The component rejects a pull request from another factory. It also rejects a run that is already linked to a different pull request. This component can only be used in factory-owned apps.`
}

func (c *AddPullRequestActivity) Icon() string {
	return "factory"
}

func (c *AddPullRequestActivity) Color() string {
	return "blue"
}

func (c *AddPullRequestActivity) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "pullRequest.activityAdded",
		"data": map[string]any{
			"description": "Please add tests for the retry path.",
			"pullRequest": map[string]any{
				"id":     "2b3cf24d-c0e2-4d42-bbe7-4c30ff2cb2a4",
				"number": 42,
				"url":    "https://github.com/acme/app/pull/42",
			},
			"workOrder": map[string]any{
				"id":     "9ac921bc-68e4-46dc-a78b-60b955f3bbf2",
				"number": 123,
				"key":    "SP-123",
			},
		},
	}
}

func (c *AddPullRequestActivity) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddPullRequestActivity) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "pullRequestId",
			Label:       "Pull Request ID",
			Description: "Factory pull request to link. Use the id from Find Pull Request.",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "description",
			Label:       "Description",
			Description: "Write why this run is linked to the pull request. You can use event data such as {{ root().data.comment.body }}.",
			Type:        configuration.FieldTypeText,
			Required:    false,
		},
	}
}

func (c *AddPullRequestActivity) Execute(ctx core.ExecutionContext) error {
	config := AddPullRequestActivityConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	match, err := ctx.Factory.AddPullRequestActivity(core.AddPullRequestActivityParams{
		PullRequestID: config.PullRequestID,
		Description:   config.Description,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pullRequest.activityAdded",
		[]any{map[string]any{
			"description": config.Description,
			"pullRequest": match.PullRequest,
			"workOrder":   match.WorkOrder,
		}},
	)
}

func (c *AddPullRequestActivity) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *AddPullRequestActivity) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddPullRequestActivity) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddPullRequestActivity) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddPullRequestActivity) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddPullRequestActivity) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
