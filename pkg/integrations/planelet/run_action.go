package planelet

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	SuccessChannel     = "success"
	FailureChannel     = "failure"
	SuccessPayloadType = "planelet.action.success"
	FailurePayloadType = "planelet.action.failed"
)

type RunAction struct{}

type RunActionConfiguration struct {
	ActionID string `json:"actionId" mapstructure:"actionId"`
}

func (r *RunAction) Name() string {
	return "planelet.runAction"
}

func (r *RunAction) Label() string {
	return "Run Planelet Action"
}

func (r *RunAction) Description() string {
	manifest := getCachedManifest()
	if manifest == nil {
		return "Execute an action on a connected Planelet server"
	}
	return fmt.Sprintf("Execute an action on %s", manifest.Label)
}

func (r *RunAction) Documentation() string {
	return `Run a remote action on a Planelet server connected via the Planelets integration.

## Use Cases

- Execute custom business logic hosted on your own server
- Integrate with internal services via the Planelet SDK
- Run any action defined in the Planelet server's manifest

## Configuration

- **Action**: Select which action to run from the Planelet server's manifest
- Additional parameters appear dynamically based on the selected action

## Output Channels

- **Success**: Emitted when the Planelet action succeeds, contains the action's response data
- **Failure**: Emitted when the action fails or the Planelet server returns an error`
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
			Name:     "actionId",
			Label:    "Action",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "action",
				},
			},
			Description: "The Planelet action to execute",
		},
	}

	manifest := getCachedManifest()
	if manifest != nil {
		for _, action := range manifest.Actions {
			fields = append(fields, manifestParametersToConfig(action.Parameters, "actionId", action.ID)...)
		}
	}

	return fields
}

func manifestParametersToConfig(parameters []ParameterManifest, ownerField string, ownerID string) []configuration.Field {
	fields := make([]configuration.Field, 0, len(parameters))
	for _, p := range parameters {
		field := configuration.Field{
			Name:        parameterFieldName(ownerID, p.ID),
			Label:       p.Label,
			Description: p.Description,
			Required:    p.Required,
			Default:     p.Default,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: ownerField, Values: []string{ownerID}},
			},
		}

		switch p.Type {
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
			opts := make([]configuration.FieldOption, 0, len(p.Options))
			for _, o := range p.Options {
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

func parameterFieldName(ownerID string, parameterID string) string {
	return fmt.Sprintf("param_%s_%s", ownerID, parameterID)
}

func extractPlaneletParams(rawConfig any, ownerID string) map[string]any {
	configMap, ok := rawConfig.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	params := map[string]any{}
	prefix := fmt.Sprintf("param_%s_", ownerID)
	for key, value := range configMap {
		if strings.HasPrefix(key, prefix) && len(key) > len(prefix) {
			paramID := strings.TrimPrefix(key, prefix)
			params[paramID] = value
		}
	}

	return params
}

func (r *RunAction) Setup(ctx core.SetupContext) error {
	var config RunActionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ActionID == "" {
		return fmt.Errorf("actionId is required")
	}

	client, err := NewClientWithHTTP(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to create Planelet client: %w", err)
	}

	manifest, err := client.FetchManifest()
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	found := false
	for _, action := range manifest.Actions {
		if action.ID == config.ActionID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("action %q not found in Planelet manifest", config.ActionID)
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

	if config.ActionID == "" {
		return r.emitFailure(ctx, "actionId is required")
	}

	client, err := NewClientWithHTTP(ctx.Integration, ctx.HTTP)
	if err != nil {
		return r.emitFailure(ctx, fmt.Sprintf("failed to create Planelet client: %v", err))
	}

	params := extractPlaneletParams(ctx.Configuration, config.ActionID)

	result, err := client.ExecuteAction(config.ActionID, params, ctx.Data)
	if err != nil {
		return r.emitFailure(ctx, fmt.Sprintf("failed to execute action: %v", err))
	}

	if !result.Success {
		return r.emitFailure(ctx, result.Error)
	}

	payload := map[string]any{
		"action": config.ActionID,
		"result": result.Data,
	}

	return ctx.ExecutionState.Emit(SuccessChannel, SuccessPayloadType, []any{payload})
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
