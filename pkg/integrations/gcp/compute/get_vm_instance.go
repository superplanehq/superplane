package compute

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetVMInstance struct{}

type GetVMInstanceSpec struct {
	Instance string `mapstructure:"instance"`
}

func (g *GetVMInstance) Name() string {
	return "gcp.getVMInstance"
}

func (g *GetVMInstance) Label() string {
	return "Compute • Get VM Instance"
}

func (g *GetVMInstance) Description() string {
	return "Fetch the current state of a Google Compute Engine VM instance"
}

func (g *GetVMInstance) Documentation() string {
	return `The Get VM Instance component reads the current state of a Compute Engine VM
instance and emits its details on the default output channel.

## Use Cases

- **Status checks**: Verify a VM is in the expected state (e.g. ` + "`RUNNING`" + `) before
  proceeding with downstream work.
- **Detail lookup**: Fetch IPs, machine type, or selfLink for use in later workflow steps.
- **Health gates**: Pair with a condition to branch a workflow based on instance status.

## Configuration

- **VM Instance**: Pick from the list of VMs in your project, or pass an expression chained
  from an upstream node (e.g. ` + "`selfLink`" + ` from ` + "`gcp.createVM`" + `). The
  selection encodes both the zone and the instance name.

## Output

The emitted payload contains the full instance summary:

- **instanceId**, **selfLink**, **status**, **zone**, **name**, **machineType**
- **internalIP**, **externalIP** (when present)

## Important Notes

- If the instance is not found at the resolved zone/name, the action fails so that
  misconfigured or stale expressions do not silently mask a missing resource.
- The integration's bound project is authoritative; a chained ` + "`selfLink`" + ` pointing
  at a different project is rejected rather than silently rewritten.`
}

func (g *GetVMInstance) Icon() string {
	return "search"
}

func (g *GetVMInstance) Color() string {
	return "blue"
}

func (g *GetVMInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetVMInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instance",
			Label:       "VM Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The VM instance to fetch. Lists every VM in your project across all zones.",
			Placeholder: "Select instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
	}
}

func (g *GetVMInstance) Setup(ctx core.SetupContext) error {
	spec := GetVMInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	instanceValue := strings.TrimSpace(spec.Instance)
	if instanceValue == "" {
		return fmt.Errorf("instance is required")
	}

	// Reuse the shared instance-metadata resolver (see instance_helpers.go),
	// which the delete/power/update/metrics components also use.
	return resolveInstanceNodeMetadata(ctx, instanceValue)
}

func (g *GetVMInstance) Execute(ctx core.ExecutionContext) error {
	spec := GetVMInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	urlProject, zone, instanceName, err := parseInstancePath(spec.Instance)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	project := client.ProjectID()
	// If the value carried an explicit project (selfLink form), it must match
	// the integration's bound project. Silently rewriting could fetch the
	// wrong VM in a different project that happens to share the same name.
	if urlProject != "" && urlProject != project {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(
			"instance belongs to project %q but this GCP integration is bound to project %q; cross-project reads are not supported",
			urlProject, project,
		))
	}

	body, err := GetInstance(context.Background(), client, project, zone, instanceName)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get VM instance: %v", err))
	}

	payload, err := InstancePayloadFromGetResponse(body, zone)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse instance response: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.vmInstance.fetched",
		[]any{payload},
	)
}

func (g *GetVMInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetVMInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetVMInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetVMInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetVMInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetVMInstance) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
