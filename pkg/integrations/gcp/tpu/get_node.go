package tpu

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetNode struct{}

type NodeRefSpec struct {
	Node string `mapstructure:"node"`
}

func (c *GetNode) Name() string {
	return "gcp.tpu.getNode"
}

func (c *GetNode) Label() string {
	return "Compute • Get TPU Node"
}

func (c *GetNode) Description() string {
	return "Read the details and status of a Cloud TPU node"
}

func (c *GetNode) Documentation() string {
	return `The Get Node component reads the current details and status of a Cloud TPU node.

## Use Cases

- **Status checks**: Inspect a TPU's ` + "`state`" + ` and ` + "`health`" + ` before dispatching work to it.
- **Chaining**: Feed a TPU's IP addresses or resource name into downstream components.

## Configuration

- **TPU node**: The TPU node to read, selected from all nodes in the project.

## Required IAM roles

The service account must have ` + "`roles/tpu.viewer`" + ` (or ` + "`roles/tpu.admin`" + `) on the project.

## Output

Emits the node: name, resourceName, location, acceleratorType, runtimeVersion, state, health, labels, ipAddresses, createTime.`
}

func (c *GetNode) Icon() string {
	return "cpu"
}

func (c *GetNode) Color() string {
	return "blue"
}

func (c *GetNode) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetNode) Configuration() []configuration.Field {
	return nodeSelectorField()
}

// nodeSelectorField is the shared TPU node selector used by the Get and Delete
// components. It lists every TPU node in the project (across all locations); the
// picker value is the node's full resource name, which carries its location.
func nodeSelectorField() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "node",
			Label:       "TPU node",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The TPU node to operate on.",
			Placeholder: "Select TPU node",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTPUNode,
				},
			},
		},
	}
}

func (c *GetNode) Setup(ctx core.SetupContext) error {
	spec := NodeRefSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	_, nodeID, err := resolveNodeSelection(spec.Node, "")
	if err != nil {
		return err
	}
	return ctx.Metadata.Set(TPUNodeMetadata{NodeName: nodeID})
}

func (c *GetNode) Execute(ctx core.ExecutionContext) error {
	spec := NodeRefSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	project := client.ProjectID()
	location, nodeID, err := resolveNodeSelection(spec.Node, project)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	body, err := getNode(context.Background(), client, project, location, nodeID)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read TPU node: %v", err))
	}
	payload, err := nodePayloadFromResponse(body)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gcp.tpu.node.read", []any{payload})
}

func (c *GetNode) Cancel(_ core.ExecutionContext) error { return nil }

func (c *GetNode) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetNode) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetNode) Cleanup(_ core.SetupContext) error { return nil }

func (c *GetNode) Hooks() []core.Hook { return []core.Hook{} }

func (c *GetNode) HandleHook(_ core.ActionHookContext) error { return nil }
