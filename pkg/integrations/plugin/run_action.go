package plugin

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	SuccessChannel     = "success"
	FailureChannel     = "failure"
	SuccessPayloadType = "plugin.action.success"
	FailurePayloadType = "plugin.action.failed"
)

type RunAction struct{}

type RunActionConfiguration struct {
	ActionName string `json:"actionName" mapstructure:"actionName"`
}

func (r *RunAction) Name() string {
	return "plugin.runAction"
}

func (r *RunAction) Label() string {
	return "Run Plugin Action"
}

func (r *RunAction) Description() string {
	manifest := getCachedManifest()
	if manifest == nil {
		return "Execute an action on a connected plugin server"
	}
	return fmt.Sprintf("Execute an action on %s", manifest.Label)
}

func (r *RunAction) Documentation() string {
	return `Run a remote action on a plugin server connected via the Plugin integration.

## Use Cases

- Execute custom business logic hosted on your own server
- Integrate with internal services via the Plugin SDK
- Run any action defined in the plugin server's manifest

## Configuration

- **Action**: Select which action to run from the plugin server's manifest
- Additional fields appear dynamically based on the selected action

## Output Channels

- **Success**: Emitted when the plugin action succeeds, contains the action's response data
- **Failure**: Emitted when the action fails or the plugin server returns an error`
}

func (r *RunAction) Icon() string {
	return "puzzle"
}

func (r *RunAction) Color() string {
	return "gray"
}

func (r *RunAction) OutputChannels(cfg any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: SuccessChannel, Label: "Success"},
		{Name: FailureChannel, Label: "Failure"},
	}
}

func (r *RunAction) Configuration() []configuration.Field {
	fields := []configuration.Field{
		{
			Name:     "actionName",
			Label:    "Action",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "action",
				},
			},
			Description: "The plugin action to execute",
		},
	}

	manifest := getCachedManifest()
	if manifest != nil {
		for _, action := range manifest.Actions {
			fields = append(fields, manifestFieldsToConfig(action.Fields, action.Name)...)
		}
	}

	return fields
}

func manifestFieldsToConfig(mFields []FieldManifest, actionName string) []configuration.Field {
	fields := make([]configuration.Field, 0, len(mFields))
	for _, f := range mFields {
		field := configuration.Field{
			Name:        fmt.Sprintf("param_%s_%s", actionName, f.Name),
			Label:       f.Label,
			Description: f.Description,
			Required:    f.Required,
			Default:     f.Default,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "actionName", Values: []string{actionName}},
			},
		}

		switch f.Type {
		case "string":
			field.Type = configuration.FieldTypeString
		case "text":
			field.Type = configuration.FieldTypeText
		case "number":
			field.Type = configuration.FieldTypeNumber
		case "bool":
			field.Type = configuration.FieldTypeBool
		case "select":
			field.Type = configuration.FieldTypeSelect
			opts := make([]configuration.FieldOption, 0, len(f.Options))
			for _, o := range f.Options {
				opts = append(opts, configuration.FieldOption{Label: o.Label, Value: o.Value})
			}
			field.TypeOptions = &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: opts},
			}
		case "object":
			field.Type = configuration.FieldTypeObject
		default:
			field.Type = configuration.FieldTypeString
		}

		fields = append(fields, field)
	}
	return fields
}

func (r *RunAction) Setup(ctx core.SetupContext) error {
	var config RunActionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ActionName == "" {
		return fmt.Errorf("actionName is required")
	}

	client, err := NewClientWithHTTP(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to create plugin client: %w", err)
	}

	manifest, err := client.FetchManifest()
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	found := false
	for _, action := range manifest.Actions {
		if action.Name == config.ActionName {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("action %q not found in plugin manifest", config.ActionName)
	}

	return nil
}

func (r *RunAction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *RunAction) Execute(ctx core.ExecutionContext) error {
	var config RunActionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return r.emitFailure(ctx, fmt.Sprintf("failed to decode configuration: %v", err))
	}

	if config.ActionName == "" {
		return r.emitFailure(ctx, "actionName is required")
	}

	client, err := NewClientWithHTTP(ctx.Integration, ctx.HTTP)
	if err != nil {
		return r.emitFailure(ctx, fmt.Sprintf("failed to create plugin client: %v", err))
	}

	params := extractActionParams(ctx.Configuration, config.ActionName)

	result, err := client.ExecuteAction(config.ActionName, params, ctx.Data)
	if err != nil {
		return r.emitFailure(ctx, fmt.Sprintf("failed to execute action: %v", err))
	}

	if !result.Success {
		return r.emitFailure(ctx, result.Error)
	}

	payload := map[string]any{
		"action": config.ActionName,
		"result": result.Data,
	}

	return ctx.ExecutionState.Emit(SuccessChannel, SuccessPayloadType, []any{payload})
}

func extractActionParams(rawConfig any, actionName string) map[string]any {
	configMap, ok := rawConfig.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	params := map[string]any{}
	prefix := fmt.Sprintf("param_%s_", actionName)
	for key, value := range configMap {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			paramName := key[len(prefix):]
			params[paramName] = value
		}
	}

	return params
}

func (r *RunAction) emitFailure(ctx core.ExecutionContext, errMsg string) error {
	payload := map[string]any{"error": errMsg}
	if err := ctx.ExecutionState.Emit(FailureChannel, FailurePayloadType, []any{payload}); err != nil {
		return err
	}
	return ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, errMsg)
}

func (r *RunAction) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (r *RunAction) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *RunAction) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (r *RunAction) Hooks() []core.Hook {
	return []core.Hook{}
}

func (r *RunAction) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
