package digitalocean

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteApp struct{}

type DeleteAppSpec struct {
	App string `json:"app" mapstructure:"app"`
}

func (d *DeleteApp) Name() string {
	return "digitalocean.deleteApp"
}

func (d *DeleteApp) Label() string {
	return "Delete App"
}

func (d *DeleteApp) Description() string {
	return "Delete a DigitalOcean App Platform application"
}

func (d *DeleteApp) Documentation() string {
	return `The Delete App component removes a DigitalOcean App Platform application.

## Use Cases

- **Cleanup**: Remove applications that are no longer needed
- **Environment teardown**: Delete temporary or test app instances
- **Resource management**: Free up resources by deleting unused apps

## Configuration

- **App**: The app to delete (required)

## Output

Returns confirmation of the deleted app including:
- **appId**: The ID of the deleted app

## Notes

- This operation is idempotent - deleting an already deleted app will succeed
- All deployments and associated resources will be removed
- This action cannot be undone`
}

func (d *DeleteApp) Icon() string {
	return "trash-2"
}

func (d *DeleteApp) Color() string {
	return "red"
}

func (d *DeleteApp) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteApp) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "app",
			Label:       "App",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select an app",
			Description: "The app to delete",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "app",
				},
			},
		},
	}
}

func (d *DeleteApp) Setup(ctx core.SetupContext) error {
	spec := DeleteAppSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.App == "" {
		return errors.New("app is required")
	}

	err = resolveAppMetadata(ctx, spec.App)
	if err != nil {
		return fmt.Errorf("error resolving app metadata: %v", err)
	}

	return nil
}

func (d *DeleteApp) Execute(ctx core.ExecutionContext) error {
	spec := DeleteAppSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.DeleteApp(spec.App)
	if err != nil {
		if doErr, ok := err.(*DOAPIError); ok && doErr.StatusCode == http.StatusNotFound {
			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				"digitalocean.app.deleted",
				[]any{map[string]any{"appId": spec.App}},
			)
		}
		return fmt.Errorf("failed to delete app: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.app.deleted",
		[]any{map[string]any{"appId": spec.App}},
	)
}

func (d *DeleteApp) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteApp) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteApp) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteApp) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined")
}

func (d *DeleteApp) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (d *DeleteApp) Cleanup(ctx core.SetupContext) error {
	return nil
}
