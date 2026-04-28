package oci

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CreateFunctionPayloadType = "oci.functionCreated"

type CreateFunction struct{}

type CreateFunctionSpec struct {
	CompartmentID    string `json:"compartmentId" mapstructure:"compartmentId"`
	ApplicationID    string `json:"applicationId" mapstructure:"applicationId"`
	DisplayName      string `json:"displayName" mapstructure:"displayName"`
	Image            string `json:"image" mapstructure:"image"`
	MemoryInMBs      int64  `json:"memoryInMBs" mapstructure:"memoryInMBs"`
	TimeoutInSeconds *int   `json:"timeoutInSeconds" mapstructure:"timeoutInSeconds"`
}

func (c *CreateFunction) Name() string {
	return "oci.createFunction"
}

func (c *CreateFunction) Label() string {
	return "Create Function"
}

func (c *CreateFunction) Description() string {
	return "Deploy a new function within an OCI Functions application"
}

func (c *CreateFunction) Documentation() string {
	return `The Create Function component deploys a new serverless function within an existing OCI Functions application.

## Use Cases

- **Continuous deployment**: Automatically deploy new function versions as part of a CI/CD pipeline
- **Serverless provisioning**: Create functions on-demand as part of environment setup workflows

## Configuration

- **Compartment**: The compartment containing the application
- **Application**: The application to deploy the function into
- **Display Name**: A human-readable name for the function
- **Image**: The container image URI (e.g. ` + "`phx.ocir.io/namespace/repo/image:tag`" + `)
- **Memory (MB)**: Memory allocated to the function (minimum 128 MB)
- **Timeout (seconds)**: Optional maximum execution time in seconds

## Output

Emits the created function details including:
- ` + "`functionId`" + ` — function OCID
- ` + "`displayName`" + ` — function name
- ` + "`applicationId`" + ` — parent application OCID
- ` + "`image`" + ` — the container image
- ` + "`invokeEndpoint`" + ` — the HTTPS endpoint used to invoke the function
- ` + "`lifecycleState`" + ` — current state
- ` + "`timeCreated`" + ` — ISO-8601 creation timestamp
`
}

func (c *CreateFunction) Icon() string {
	return "oci"
}

func (c *CreateFunction) Color() string {
	return "red"
}

func (c *CreateFunction) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateFunction) ExampleOutput() map[string]any {
	return exampleOutputCreateFunction()
}

func (c *CreateFunction) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartmentId",
			Label:       "Compartment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The compartment containing the application",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCompartment,
				},
			},
		},
		{
			Name:        "applicationId",
			Label:       "Application",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The application to deploy the function into",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeFunctionApplication,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartmentId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A human-readable name for the function",
			Placeholder: "my-function",
		},
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Container image URI (e.g. phx.ocir.io/namespace/repo/image:tag)",
			Placeholder: "phx.ocir.io/mynamespace/myrepo/myfunction:0.0.1",
		},
		{
			Name:        "memoryInMBs",
			Label:       "Memory (MB)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     "128",
			Description: "Memory allocated to the function in megabytes (minimum 128)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { v := 128; return &v }(),
				},
			},
		},
		{
			Name:        "timeoutInSeconds",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Maximum execution time in seconds. Defaults to 30 if not set.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { v := 1; return &v }(),
				},
			},
		},
	}
}

func (c *CreateFunction) Setup(ctx core.SetupContext) error {
	spec := CreateFunctionSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.ApplicationID) == "" {
		return errors.New("applicationId is required")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		return errors.New("displayName is required")
	}
	if strings.TrimSpace(spec.Image) == "" {
		return errors.New("image is required")
	}
	if spec.MemoryInMBs < 128 {
		return errors.New("memoryInMBs must be at least 128")
	}

	return nil
}

func (c *CreateFunction) Execute(ctx core.ExecutionContext) error {
	spec := CreateFunctionSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	fn, err := client.CreateFunction(spec.ApplicationID, spec.DisplayName, spec.Image, spec.MemoryInMBs, spec.TimeoutInSeconds)
	if err != nil {
		return fmt.Errorf("failed to create function: %w", err)
	}

	payload := map[string]any{
		"functionId":     fn.ID,
		"displayName":    fn.DisplayName,
		"applicationId":  fn.ApplicationID,
		"image":          fn.Image,
		"memoryInMBs":    fn.MemoryInMBs,
		"invokeEndpoint": fn.InvokeEndpoint,
		"lifecycleState": fn.LifecycleState,
		"timeCreated":    fn.TimeCreated,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateFunctionPayloadType, []any{payload})
}

func (c *CreateFunction) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateFunction) HandleHook(ctx core.ActionHookContext) error {
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *CreateFunction) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *CreateFunction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateFunction) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateFunction) Cleanup(_ core.SetupContext) error {
	return nil
}
