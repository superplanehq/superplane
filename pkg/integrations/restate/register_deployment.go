package restate

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type RegisterDeployment struct{}

type RegisterDeploymentSpec struct {
	URI       string `json:"uri"`
	Force     bool   `json:"force"`
	DryRun    bool   `json:"dryRun"`
	UseHTTP11 bool   `json:"useHttp11"`
}

func (c *RegisterDeployment) Name() string {
	return "restate.registerDeployment"
}

func (c *RegisterDeployment) Label() string {
	return "Register Deployment"
}

func (c *RegisterDeployment) Description() string {
	return "Register a new service deployment with the Restate server"
}

func (c *RegisterDeployment) Icon() string {
	return "repeat"
}

func (c *RegisterDeployment) Color() string {
	return "gray"
}

func (c *RegisterDeployment) Documentation() string {
	return `The Register Deployment component registers a new service deployment endpoint with the Restate server.

## Use Cases

- **CI/CD pipelines**: Automatically register new service versions after deployment
- **Service discovery**: Register services with Restate as part of infrastructure provisioning
- **Rolling updates**: Force-register updated service endpoints during rolling deployments

## Options

- **Force**: Override an existing deployment with the same URI. Required when updating a deployment.
- **Dry Run**: Validate the deployment without actually registering it.
- **Use HTTP 1.1**: Use HTTP 1.1 instead of HTTP 2 for service communication.

## Outputs

The component emits the full deployment response from Restate, including:
- ` + "`id`" + `: The deployment ID
- ` + "`uri`" + `: The registered URI
- ` + "`services`" + `: Array of discovered services and their handlers
- ` + "`protocol_type`" + `: The protocol type used
`
}

func (c *RegisterDeployment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RegisterDeployment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "uri",
			Label:       "Service URI",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "URI of the service deployment endpoint (e.g. http://my-service:9080)",
			Placeholder: "http://my-service:9080",
		},
		{
			Name:        "force",
			Label:       "Force",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Override an existing deployment with the same URI",
		},
		{
			Name:        "dryRun",
			Label:       "Dry Run",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Validate the deployment without registering it",
		},
		{
			Name:        "useHttp11",
			Label:       "Use HTTP 1.1",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Use HTTP 1.1 instead of HTTP 2 for service communication",
		},
	}
}

func (c *RegisterDeployment) Setup(ctx core.SetupContext) error {
	spec := RegisterDeploymentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.URI == "" {
		return errors.New("uri is required")
	}

	return nil
}

func (c *RegisterDeployment) Execute(ctx core.ExecutionContext) error {
	spec := RegisterDeploymentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	req := RegisterDeploymentRequest{
		URI:       spec.URI,
		Force:     spec.Force,
		DryRun:    spec.DryRun,
		UseHTTP11: spec.UseHTTP11,
	}

	result, err := client.RegisterDeployment(req)
	if err != nil {
		return fmt.Errorf("failed to register deployment: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.deployment.registered",
		[]any{result},
	)
}

func (c *RegisterDeployment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RegisterDeployment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RegisterDeployment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RegisterDeployment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RegisterDeployment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RegisterDeployment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
