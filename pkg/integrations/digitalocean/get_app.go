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

type GetApp struct{}

type GetAppSpec struct {
	App string `json:"app" mapstructure:"app"`
}

func (g *GetApp) Name() string {
	return "digitalocean.getApp"
}

func (g *GetApp) Label() string {
	return "Get App"
}

func (g *GetApp) Description() string {
	return "Fetch details of a DigitalOcean App Platform application by ID"
}

func (g *GetApp) Documentation() string {
	return `The Get App component retrieves detailed information about a specific DigitalOcean App Platform application.

## Use Cases

- **Status checks**: Verify app state and deployment status before performing operations
- **Information retrieval**: Get current app configuration, URLs, and deployment details
- **Pre-flight validation**: Check app exists before operations like update or delete
- **Monitoring**: Track app configuration, active deployments, and ingress URLs
- **Integration workflows**: Retrieve app details for use in downstream workflow steps

## Configuration

- **App ID**: The unique identifier of the app to retrieve (required)

## Output

Returns the app object including:
- **id**: The unique app ID
- **name**: The app name
- **default_ingress**: The default ingress URL
- **live_url**: The live URL for the app
- **region**: The region where the app is deployed
- **active_deployment**: Information about the active deployment
- **in_progress_deployment**: Information about any in-progress deployment
- **spec**: Complete app specification including services, workers, jobs, static sites, databases, and configuration

## Notes

- The app ID can be obtained from the output of the Create App component or from the DigitalOcean dashboard
- The component returns the current state of the app, including all deployed components
- Use this component to verify deployment status before performing updates or other operations`
}

func (g *GetApp) Icon() string {
	return "info"
}

func (g *GetApp) Color() string {
	return "gray"
}

func (g *GetApp) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetApp) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "app",
			Label:       "App",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The ID of the app to retrieve",
			Placeholder: "Select app",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "app",
					UseNameAsValue: false,
				},
			},
		},
	}
}

func (g *GetApp) Setup(ctx core.SetupContext) error {
	spec := GetAppSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.App == "" {
		return errors.New("app ID is required")
	}

	err = resolveAppMetadata(ctx, spec.App)
	if err != nil {
		return fmt.Errorf("error resolving app metadata: %v", err)
	}

	return nil
}

func (g *GetApp) Execute(ctx core.ExecutionContext) error {
	spec := GetAppSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	app, err := client.GetApp(spec.App)
	if err != nil {
		return fmt.Errorf("failed to get app: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.app.fetched",
		[]any{app},
	)
}

func (g *GetApp) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetApp) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetApp) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetApp) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetApp) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetApp) Cleanup(ctx core.SetupContext) error {
	return nil
}
