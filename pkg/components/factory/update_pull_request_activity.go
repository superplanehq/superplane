package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const UpdatePullRequestActivityComponentName = "updatePullRequestActivity"
const updatePullRequestActivityEventType = "pullRequest.activityUpdated"

func init() {
	registry.RegisterAction(UpdatePullRequestActivityComponentName, &UpdatePullRequestActivity{})
}

type UpdatePullRequestActivity struct{}

type UpdatePullRequestActivityConfiguration struct {
	Description string `json:"description" mapstructure:"description"`
	Access      string `json:"access" mapstructure:"access"`
}

func (c *UpdatePullRequestActivity) Name() string {
	return UpdatePullRequestActivityComponentName
}

func (c *UpdatePullRequestActivity) Label() string {
	return "Update Pull Request Activity"
}

func (c *UpdatePullRequestActivity) Description() string {
	return "Update the current factory pull request activity"
}

func (c *UpdatePullRequestActivity) Documentation() string {
	return `The Update Pull Request Activity component updates the activity that belongs to the current canvas run.

Use ` + "`description`" + ` to replace the displayed activity text. Use ` + "`access`" + ` to keep the current access or request exclusive access. If exclusive access is not available, the component waits and retries.

The component emits ` + "`limitReached`" + ` when a check handler reaches its attempt limit. You can still update the description after the attempt limit. A newer pull request head does not stop this activity.

This component can only be used in factory-owned apps.`
}

func (c *UpdatePullRequestActivity) Icon() string {
	return "factory"
}

func (c *UpdatePullRequestActivity) Color() string {
	return "blue"
}

func (c *UpdatePullRequestActivity) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      updatePullRequestActivityEventType,
		"data": map[string]any{
			"description":  "Fixing failed checks on d1209da",
			"attempt":      1,
			"attemptLimit": 3,
			"pullRequest": map[string]any{
				"id":     "2b3cf24d-c0e2-4d42-bbe7-4c30ff2cb2a4",
				"number": 42,
			},
			"currentHead": "d1209da000000000000000000000000000000000",
		},
	}
}

func (c *UpdatePullRequestActivity) OutputChannels(configuration any) []core.OutputChannel {
	return pullRequestActivityChannels()
}

func (c *UpdatePullRequestActivity) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "description",
			Label:       "Description",
			Description: "Replace the current activity description.",
			Type:        configuration.FieldTypeText,
			Required:    false,
		},
		{
			Name:        "access",
			Label:       "Access",
			Description: "Keep the current access or request exclusive access before the run changes the pull request.",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Concurrent", Value: core.PullRequestActivityAccessConcurrent},
						{Label: "Exclusive", Value: core.PullRequestActivityAccessExclusive},
					},
				},
			},
		},
	}
}

func (c *UpdatePullRequestActivity) Execute(ctx core.ExecutionContext) error {
	return c.apply(ctx.Configuration, ctx.Factory, ctx.ExecutionState, ctx.Requests)
}

func (c *UpdatePullRequestActivity) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdatePullRequestActivity) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdatePullRequestActivity) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdatePullRequestActivity) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdatePullRequestActivity) Hooks() []core.Hook {
	return []core.Hook{acquireAccessHook}
}

func (c *UpdatePullRequestActivity) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != acquireAccessHookName {
		return nil
	}
	if ctx.ExecutionState.IsFinished() {
		return nil
	}
	return c.apply(ctx.Configuration, ctx.Factory, ctx.ExecutionState, ctx.Requests)
}

func (c *UpdatePullRequestActivity) apply(
	configuration any,
	factory core.FactoryContext,
	state core.ExecutionStateContext,
	requests core.RequestContext,
) error {
	config := UpdatePullRequestActivityConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return err
	}

	params := core.UpdatePullRequestActivityParams{Access: config.Access}
	if config.Description != "" {
		params.Description = &config.Description
	}

	result, err := factory.UpdatePullRequestActivity(params)
	if err != nil {
		return err
	}

	return finishPullRequestActivity(state, requests, updatePullRequestActivityEventType, result)
}
