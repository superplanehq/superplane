package fluxcd

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ReconcileSource struct{}

type ReconcileSourceSpec struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

var reconcileKindOptions = []configuration.FieldOption{
	{Label: "Kustomization", Value: "Kustomization"},
	{Label: "HelmRelease", Value: "HelmRelease"},
	{Label: "GitRepository", Value: "GitRepository"},
	{Label: "HelmRepository", Value: "HelmRepository"},
	{Label: "OCIRepository", Value: "OCIRepository"},
	{Label: "Bucket", Value: "Bucket"},
}

func (c *ReconcileSource) Name() string {
	return "fluxcd.reconcileSource"
}

func (c *ReconcileSource) Label() string {
	return "Reconcile Source"
}

func (c *ReconcileSource) Description() string {
	return "Force reconciliation of a Flux resource"
}

func (c *ReconcileSource) Documentation() string {
	return `The Reconcile Source component triggers reconciliation of a Flux CD resource by patching the ` + "`reconcile.fluxcd.io/requestedAt`" + ` annotation.

## Use Cases

- **Manual re-sync**: Force a Flux resource to reconcile on demand
- **Approval-based deploys**: Trigger reconciliation after manual approval
- **Cross-cluster sync**: Reconcile resources in response to events from other systems

## Configuration

- **Kind**: The type of Flux resource to reconcile (Kustomization, HelmRelease, GitRepository, etc.)
- **Namespace**: The Kubernetes namespace of the resource
- **Name**: The name of the Flux resource

## Outputs

The component emits the updated Kubernetes resource object, including:
- ` + "`kind`" + `: The resource kind
- ` + "`namespace`" + `: The resource namespace
- ` + "`name`" + `: The resource name
- ` + "`annotations`" + `: Updated annotations including the reconciliation timestamp
`
}

func (c *ReconcileSource) Icon() string {
	return "git-branch"
}

func (c *ReconcileSource) Color() string {
	return "gray"
}

func (c *ReconcileSource) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ReconcileSource) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "kind",
			Label:    "Resource Kind",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "Kustomization",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: reconcileKindOptions,
				},
			},
			Description: "The type of Flux resource to reconcile",
		},
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Kubernetes namespace of the resource (defaults to the integration's default namespace)",
			Placeholder: "flux-system",
		},
		{
			Name:        "name",
			Label:       "Resource Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the Flux resource to reconcile",
			Placeholder: "my-app",
		},
	}
}

func (c *ReconcileSource) Setup(ctx core.SetupContext) error {
	spec := ReconcileSourceSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Kind == "" {
		return fmt.Errorf("kind is required")
	}

	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}

	if _, ok := fluxResourceAPIs[spec.Kind]; !ok {
		return fmt.Errorf("unsupported resource kind: %s", spec.Kind)
	}

	return nil
}

func (c *ReconcileSource) Execute(ctx core.ExecutionContext) error {
	spec := ReconcileSourceSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	result, err := client.ReconcileResource(spec.Kind, spec.Namespace, spec.Name)
	if err != nil {
		return fmt.Errorf("failed to reconcile resource: %v", err)
	}

	output := reconcileResultToOutput(result, spec)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"fluxcd.reconciliation",
		[]any{output},
	)
}

func (c *ReconcileSource) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ReconcileSource) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ReconcileSource) Actions() []core.Action {
	return []core.Action{}
}

func (c *ReconcileSource) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ReconcileSource) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *ReconcileSource) Cleanup(ctx core.SetupContext) error {
	return nil
}

func reconcileResultToOutput(result map[string]any, spec ReconcileSourceSpec) map[string]any {
	output := map[string]any{
		"kind":      spec.Kind,
		"namespace": spec.Namespace,
		"name":      spec.Name,
	}

	if metadata, ok := result["metadata"].(map[string]any); ok {
		if annotations, ok := metadata["annotations"].(map[string]any); ok {
			output["annotations"] = annotations
		}
		if resourceVersion, ok := metadata["resourceVersion"].(string); ok {
			output["resourceVersion"] = resourceVersion
		}
	}

	if status, ok := result["status"].(map[string]any); ok {
		if lastAppliedRevision, ok := status["lastAppliedRevision"].(string); ok {
			output["lastAppliedRevision"] = lastAppliedRevision
		}
		if lastAttemptedRevision, ok := status["lastAttemptedRevision"].(string); ok {
			output["lastAttemptedRevision"] = lastAttemptedRevision
		}
	}

	return output
}
