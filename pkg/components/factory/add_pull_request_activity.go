package factory

import (
	"errors"
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const AddPullRequestActivityComponentName = "addPullRequestActivity"
const addPullRequestActivityEventType = "pullRequest.activityAdded"

func init() {
	registry.RegisterAction(AddPullRequestActivityComponentName, &AddPullRequestActivity{})
}

type AddPullRequestActivity struct{}

type AddPullRequestActivityConfiguration struct {
	PullRequestID string `json:"pullRequestId" mapstructure:"pullRequestId"`
	Revision      string `json:"revision" mapstructure:"revision"`
	Description   string `json:"description" mapstructure:"description"`
	Access        string `json:"access" mapstructure:"access"`
}

func (c *AddPullRequestActivity) Name() string {
	return AddPullRequestActivityComponentName
}

func (c *AddPullRequestActivity) Label() string {
	return "Add Pull Request Activity"
}

func (c *AddPullRequestActivity) Description() string {
	return "Create a factory pull request activity for the current run"
}

func (c *AddPullRequestActivity) Documentation() string {
	return `The Add Pull Request Activity component creates one activity for the current canvas run.

Set ` + "`pullRequestId`" + ` from Find Pull Request output. Use ` + "`revision`" + ` to bind the activity to one pull request head SHA. Check handlers use a revision. Discussion handlers omit it so the activity stays pull-request-scoped.

Use ` + "`access`" + ` to request ` + "`concurrent`" + ` or ` + "`exclusive`" + ` access. Concurrent activities can observe a pull request without changing it. Exclusive access is required before a run changes the pull request. If exclusive access is not available, the component waits and retries.

Use ` + "`description`" + ` to record why this run is linked. The field accepts event data, for example ` + "`{{ root().data.comment.body }}`" + `.

If another active activity already owns the same handler and revision, this node finishes without output. The existing activity keeps running. A newer pull request head does not stop an in-flight activity.

This component can only be used in factory-owned apps.`
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
		"type":      addPullRequestActivityEventType,
		"data": map[string]any{
			"description": "Waiting for checks on d1209da",
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
			"currentHead": "d1209da000000000000000000000000000000000",
		},
	}
}

func (c *AddPullRequestActivity) OutputChannels(configuration any) []core.OutputChannel {
	return pullRequestActivityChannels()
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
			Name:        "revision",
			Label:       "Revision",
			Description: "Full pull request head SHA. Omit this field for a pull-request-scoped activity.",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
		{
			Name:        "description",
			Label:       "Description",
			Description: "Write why this run is linked to the pull request. You can use event data such as {{ root().data.comment.body }}.",
			Type:        configuration.FieldTypeText,
			Required:    false,
		},
		{
			Name:        "access",
			Label:       "Access",
			Description: "Concurrent activities observe the pull request. Exclusive access is required before a run changes it.",
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

func (c *AddPullRequestActivity) Execute(ctx core.ExecutionContext) error {
	config := AddPullRequestActivityConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	result, err := ctx.Factory.AddPullRequestActivity(core.AddPullRequestActivityParams{
		PullRequestID: config.PullRequestID,
		Description:   config.Description,
		Revision:      config.Revision,
		Access:        config.Access,
	})
	return finishAddedPullRequestActivity(ctx.ExecutionState, ctx.Requests, ctx.Runs, config.Access, result, err)
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
	return []core.Hook{acquireAccessHook}
}

func (c *AddPullRequestActivity) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != acquireAccessHookName {
		return nil
	}
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	result, err := ctx.Factory.AddPullRequestActivity(core.AddPullRequestActivityParams{
		PullRequestID: hookPullRequestID(ctx.Configuration),
		Description:   hookString(ctx.Configuration, "description"),
		Revision:      hookString(ctx.Configuration, "revision"),
		Access:        hookString(ctx.Configuration, "access"),
	})
	return finishAddedPullRequestActivity(ctx.ExecutionState, ctx.Requests, ctx.Runs, hookString(ctx.Configuration, "access"), result, err)
}

func finishAddedPullRequestActivity(
	state core.ExecutionStateContext,
	requests core.RequestContext,
	runs core.RunExecutionContext,
	requestedAccess string,
	result *core.PullRequestActivityResult,
	err error,
) error {
	if errors.Is(err, core.ErrPullRequestActivityAlreadyActive) {
		return state.Pass()
	}
	if err != nil {
		return err
	}
	return finishPullRequestActivity(state, requests, runs, addPullRequestActivityEventType, result, requestedAccess)
}

func hookPullRequestID(configuration any) string {
	return hookString(configuration, "pullRequestId")
}

func hookString(configuration any, name string) string {
	values, ok := configuration.(map[string]any)
	if !ok {
		return ""
	}
	value, _ := values[name].(string)
	return value
}
