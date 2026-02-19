package flyio

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetAppPayloadType = "flyio.app"

type GetApp struct{}

type GetAppSpec struct {
	App string `json:"app" mapstructure:"app"`
}

func (c *GetApp) Name() string {
	return "flyio.getApp"
}

func (c *GetApp) Label() string {
	return "Get App"
}

func (c *GetApp) Description() string {
	return "Retrieve details for a Fly.io App"
}

func (c *GetApp) Documentation() string {
	return `The Get App component fetches current details for a Fly.io application.

## Use Cases

- **Inspect app status**: Check the current status of an app before acting on it
- **Workflow context**: Use app fields (status, machine count) to drive branching decisions in downstream steps

## Configuration

- **App**: The Fly.io application to retrieve details for

## Output

Emits a ` + "`flyio.app`" + ` payload containing the app's ` + "`name`" + `, ` + "`status`" + `, ` + "`machineCount`" + `, ` + "`volumeCount`" + `, and ` + "`network`" + `.`
}

func (c *GetApp) Icon() string {
	return "server"
}

func (c *GetApp) Color() string {
	return "purple"
}

func (c *GetApp) ExampleOutput() map[string]any {
	return map[string]any{
		"name":         "my-fly-app",
		"status":       "deployed",
		"machineCount": 2,
		"volumeCount":  1,
		"network":      "default",
	}
}

func (c *GetApp) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetApp) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "app",
			Label:    "App",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "app",
					UseNameAsValue: true,
				},
			},
			Description: "Fly.io application to retrieve details for",
		},
	}
}

func decodeGetAppSpec(configuration any) (GetAppSpec, error) {
	spec := GetAppSpec{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return GetAppSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.App = strings.TrimSpace(spec.App)
	if spec.App == "" {
		return GetAppSpec{}, fmt.Errorf("app is required")
	}

	return spec, nil
}

func (c *GetApp) Setup(ctx core.SetupContext) error {
	_, err := decodeGetAppSpec(ctx.Configuration)
	return err
}

func (c *GetApp) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetApp) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetAppSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	app, err := client.GetApp(spec.App)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	output := map[string]any{
		"name":         app.Name,
		"id":           app.ID,
		"status":       app.Status,
		"machineCount": app.MachineCount,
		"volumeCount":  app.VolumeCount,
		"network":      app.Network,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetAppPayloadType, []any{output})
}

func (c *GetApp) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *GetApp) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *GetApp) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetApp) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetApp) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
