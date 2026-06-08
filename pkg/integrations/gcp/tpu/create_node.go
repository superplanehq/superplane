package tpu

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateNode struct{}

type CreateNodeSpec struct {
	Name              string       `mapstructure:"name"`
	Location          string       `mapstructure:"location"`
	AcceleratorType   string       `mapstructure:"acceleratorType"`
	RuntimeVersion    string       `mapstructure:"runtimeVersion"`
	Description       string       `mapstructure:"description"`
	Network           string       `mapstructure:"network"`
	Subnetwork        string       `mapstructure:"subnetwork"`
	EnableExternalIps bool         `mapstructure:"enableExternalIps"`
	Preemptible       bool         `mapstructure:"preemptible"`
	Labels            []LabelEntry `mapstructure:"labels"`
}

func (c *CreateNode) Name() string {
	return "gcp.tpu.createNode"
}

func (c *CreateNode) Label() string {
	return "Compute • Create TPU Node"
}

func (c *CreateNode) Description() string {
	return "Create a Cloud TPU node (TPU VM) in a Google Cloud project"
}

func (c *CreateNode) Documentation() string {
	return `The Create Node component provisions a new Cloud TPU node (TPU VM) and waits for it to become ready before emitting.

## Use Cases

- **On-demand accelerators**: Spin up a TPU for a training or inference job as part of a workflow.
- **Reproducible environments**: Pin the accelerator type and runtime version so every run uses the same hardware and software.

## Configuration

- **Node name**: Name for the new TPU node. Start with a letter; use lowercase letters, numbers, and hyphens.
- **Location**: The zone to create the TPU in (e.g. ` + "`us-central1-b`" + `).
- **Accelerator type**: The TPU hardware type (e.g. ` + "`v2-8`" + `, ` + "`v3-8`" + `, ` + "`v5litepod-4`" + `).
- **Runtime version**: The TPU software/runtime version (e.g. ` + "`tpu-vm-tf-2.16.1`" + `).
- **Description**: Optional human-readable description.
- **Network / Subnetwork**: Optional VPC network and subnetwork. Defaults to the project's ` + "`default`" + ` network.
- **Assign external IP**: Whether to give the TPU an external IP address.
- **Preemptible**: Create the TPU as preemptible (cheaper, can be reclaimed at any time).
- **Labels**: Optional key-value labels (billing, environment, team).

## Required IAM roles

The service account must have ` + "`roles/tpu.admin`" + ` on the project.

## Output

Emits the created node: name, resourceName, location, acceleratorType, runtimeVersion, state, health, labels, ipAddresses, createTime.

## Important Notes

- The component waits for the create operation to complete before emitting, so the node is ` + "`READY`" + ` in the output.
- The Cloud TPU API must be enabled in the project.`
}

func (c *CreateNode) Icon() string {
	return "cpu"
}

func (c *CreateNode) Color() string {
	return "blue"
}

func (c *CreateNode) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateNode) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Node name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name for the new TPU node. Start with a letter; use only a-z, 0-9, and hyphens.",
			Placeholder: "e.g. my-tpu",
		},
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The zone to create the TPU node in.",
			Placeholder: "Select location",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTPULocation,
				},
			},
		},
		{
			Name:        "acceleratorType",
			Label:       "Accelerator type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The TPU hardware type (e.g. v2-8).",
			Placeholder: "Select accelerator type",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTPUAcceleratorType,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
			},
		},
		{
			Name:        "runtimeVersion",
			Label:       "Runtime version",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The TPU software/runtime version (e.g. tpu-vm-tf-2.16.1).",
			Placeholder: "Select runtime version",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTPURuntimeVersion,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
			},
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Optional node description",
		},
		{
			Name:        "network",
			Label:       "Network",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional VPC network name or full URL. Defaults to the project's default network.",
			Placeholder: "e.g. default",
		},
		{
			Name:        "subnetwork",
			Label:       "Subnetwork",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional subnetwork name or full URL.",
			Placeholder: "e.g. default",
		},
		{
			Name:        "enableExternalIps",
			Label:       "Assign external IP",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Give the TPU node an external IP address.",
		},
		{
			Name:        "preemptible",
			Label:       "Preemptible",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Create the TPU as preemptible (cheaper, can be reclaimed at any time).",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Key-value labels for the TPU node (billing, environment, team).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Label:       "Key",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label key (e.g. env, team, cost-center).",
								Placeholder: "e.g. env",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Label value.",
								Placeholder: "e.g. production",
							},
						},
					},
				},
			},
		},
	}
}

func (c *CreateNode) Setup(ctx core.SetupContext) error {
	spec := CreateNodeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if err := validateCreateNodeSpec(spec); err != nil {
		return err
	}
	return ctx.Metadata.Set(TPUNodeMetadata{NodeName: strings.TrimSpace(spec.Name)})
}

func validateCreateNodeSpec(spec CreateNodeSpec) error {
	if strings.TrimSpace(spec.Name) == "" {
		return errors.New("node name is required")
	}
	if strings.TrimSpace(spec.Location) == "" {
		return errors.New("location is required")
	}
	if strings.TrimSpace(spec.AcceleratorType) == "" {
		return errors.New("accelerator type is required")
	}
	if strings.TrimSpace(spec.RuntimeVersion) == "" {
		return errors.New("runtime version is required")
	}
	return nil
}

func (c *CreateNode) Execute(ctx core.ExecutionContext) error {
	spec := CreateNodeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	if err := validateCreateNodeSpec(spec); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	project := client.ProjectID()
	location := strings.TrimSpace(spec.Location)
	name := strings.TrimSpace(spec.Name)
	node := buildNodeFromSpec(spec)

	callCtx := context.Background()
	body, err := createNode(callCtx, client, project, location, name, node)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create TPU node: %v", err))
	}

	opName, err := operationNameFromBody(body, "create node")
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if _, err := waitForOperation(callCtx, client, opName); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create TPU node: %v", err))
	}

	nodeBody, err := getNode(callCtx, client, project, location, name)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read TPU node after create: %v", err))
	}
	payload, err := nodePayloadFromResponse(nodeBody)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gcp.tpu.node.created", []any{payload})
}

func buildNodeFromSpec(spec CreateNodeSpec) *Node {
	node := &Node{
		AcceleratorType: strings.TrimSpace(spec.AcceleratorType),
		RuntimeVersion:  strings.TrimSpace(spec.RuntimeVersion),
		Description:     strings.TrimSpace(spec.Description),
		Labels:          labelsFromEntries(spec.Labels),
	}

	network := strings.TrimSpace(spec.Network)
	subnetwork := strings.TrimSpace(spec.Subnetwork)
	if network != "" || subnetwork != "" || spec.EnableExternalIps {
		node.NetworkConfig = &NetworkConfig{
			Network:           network,
			Subnetwork:        subnetwork,
			EnableExternalIps: spec.EnableExternalIps,
		}
	}

	if spec.Preemptible {
		node.SchedulingConfig = &SchedulingConfig{Preemptible: true}
	}

	return node
}

func (c *CreateNode) Cancel(_ core.ExecutionContext) error { return nil }

func (c *CreateNode) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateNode) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateNode) Cleanup(_ core.SetupContext) error { return nil }

func (c *CreateNode) Hooks() []core.Hook { return []core.Hook{} }

func (c *CreateNode) HandleHook(_ core.ActionHookContext) error { return nil }
