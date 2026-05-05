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

// TagItem represents a single key-value free-form tag.
type TagItem struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

func tagItemsToMap(items []TagItem) map[string]string {
	if len(items) == 0 {
		return nil
	}
	m := make(map[string]string, len(items))
	for _, t := range items {
		if strings.TrimSpace(t.Key) != "" {
			m[t.Key] = t.Value
		}
	}
	return m
}

type CreateFunctionNodeMetadata struct {
	ApplicationID   string `json:"applicationId" mapstructure:"applicationId"`
	ApplicationName string `json:"applicationName" mapstructure:"applicationName"`
}

type CreateFunctionSpec struct {
	Compartment                    string    `json:"compartment" mapstructure:"compartment"`
	Application                    string    `json:"application" mapstructure:"application"`
	DisplayName                    string    `json:"displayName" mapstructure:"displayName"`
	ImageRepository                string    `json:"imageRepository" mapstructure:"imageRepository"`
	Image                          string    `json:"image" mapstructure:"image"`
	MemoryInMBs                    int64     `json:"memoryInMBs" mapstructure:"memoryInMBs"`
	TimeoutInSeconds               *int      `json:"timeoutInSeconds" mapstructure:"timeoutInSeconds"`
	TraceEnabled                   bool      `json:"traceEnabled" mapstructure:"traceEnabled"`
	SourceTriggerType              string    `json:"sourceTriggerType" mapstructure:"sourceTriggerType"`
	ProvisionedConcurrencyStrategy string    `json:"provisionedConcurrencyStrategy" mapstructure:"provisionedConcurrencyStrategy"`
	ProvisionedConcurrencyCount    int       `json:"provisionedConcurrencyCount" mapstructure:"provisionedConcurrencyCount"`
	FreeformTags                   []TagItem `json:"freeformTags" mapstructure:"freeformTags"`
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
- ` + "`memoryInMBs`" + ` — memory allocated in megabytes
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
			Name:        "compartment",
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
			Name:        "application",
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
								Field: "compartment",
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
			Name:        "imageRepository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The OCIR container repository containing the function image",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeContainerRepository,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartment",
							},
						},
					},
				},
			},
		},
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The container image to deploy (full URI is used as the function image)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeContainerImage,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartment",
							},
						},
						{
							Name: "repositoryId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "imageRepository",
							},
						},
					},
				},
			},
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
		{
			Name:        "traceEnabled",
			Label:       "Enable Tracing",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Enable OCI Application Performance Monitoring tracing for this function",
		},
		{
			Name:        "sourceTriggerType",
			Label:       "Invocation Mode",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Configure how the function receives invocations. OCI_STREAMING enables detached async invocation via OCI Streaming.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None (synchronous)", Value: "NONE"},
						{Label: "OCI Streaming (detached / async)", Value: "OCI_STREAMING"},
					},
				},
			},
		},
		{
			Name:        "provisionedConcurrencyStrategy",
			Label:       "Provisioned Concurrency",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Pre-warm instances to reduce cold-start latency. Use CONSTANT to keep a fixed number of warm instances.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None", Value: "NONE"},
						{Label: "Constant", Value: "CONSTANT"},
					},
				},
			},
		},
		{
			Name:        "provisionedConcurrencyCount",
			Label:       "Warm Instance Count",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "1",
			Description: "Number of pre-warmed function instances to keep running",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "provisionedConcurrencyStrategy", Values: []string{"CONSTANT"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { v := 1; return &v }(),
				},
			},
		},
		{
			Name:        "freeformTags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Free-form tags (key-value pairs) to apply to the function",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "key", Label: "Key", Type: configuration.FieldTypeString, Required: true},
							{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true},
						},
					},
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

	if strings.TrimSpace(spec.Application) == "" {
		return errors.New("application is required")
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

	return resolveCreateFunctionMetadata(ctx, spec.Application)
}

func resolveCreateFunctionMetadata(ctx core.SetupContext, applicationID string) error {
	if strings.Contains(applicationID, "{{") {
		return ctx.Metadata.Set(CreateFunctionNodeMetadata{ApplicationName: applicationID})
	}

	var existing CreateFunctionNodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err == nil &&
		existing.ApplicationID == applicationID && existing.ApplicationName != "" {
		return nil
	}

	return ctx.Metadata.Set(CreateFunctionNodeMetadata{
		ApplicationID:   applicationID,
		ApplicationName: resolveApplicationName(ctx, applicationID),
	})
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

	fn, err := client.CreateFunction(CreateFunctionInput{
		ApplicationID:                  spec.Application,
		DisplayName:                    spec.DisplayName,
		Image:                          spec.Image,
		MemoryInMBs:                    spec.MemoryInMBs,
		TimeoutInSeconds:               spec.TimeoutInSeconds,
		TraceEnabled:                   spec.TraceEnabled,
		SourceTriggerType:              spec.SourceTriggerType,
		ProvisionedConcurrencyStrategy: spec.ProvisionedConcurrencyStrategy,
		ProvisionedConcurrencyCount:    spec.ProvisionedConcurrencyCount,
		FreeformTags:                   tagItemsToMap(spec.FreeformTags),
	})
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
