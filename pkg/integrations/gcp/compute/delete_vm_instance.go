package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

type DeleteVMInstance struct{}

type DeleteVMInstanceSpec struct {
	Zone     string `mapstructure:"zone"`
	Instance string `mapstructure:"instance"`
}

type VMInstanceNodeMetadata struct {
	InstanceName string `json:"instanceName" mapstructure:"instanceName"`
	Zone         string `json:"zone" mapstructure:"zone"`
}

func (d *DeleteVMInstance) Name() string {
	return "gcp.deleteVMInstance"
}

func (d *DeleteVMInstance) Label() string {
	return "Compute • Delete VM Instance"
}

func (d *DeleteVMInstance) Description() string {
	return "Permanently delete a Google Compute Engine VM instance by name and zone"
}

func (d *DeleteVMInstance) Documentation() string {
	return `The Delete VM Instance component permanently deletes a Compute Engine VM instance.

## Use Cases

- **Cleanup**: Remove temporary or test VMs after use
- **Cost optimization**: Automatically tear down unused infrastructure
- **Automated workflows**: Delete VMs as part of deployment rollback or cleanup processes
- **Environment management**: Remove ephemeral environments after testing

## Configuration

- **Zone**: The GCP zone where the instance lives (required)
- **Instance**: The VM instance name to delete (required, supports expressions)

## Output

Returns information about the deleted instance:
- **instanceName**: The name of the instance that was deleted
- **zone**: The zone the instance was in

## Important Notes

- This operation is **permanent** and cannot be undone
- All data on the instance will be lost unless boot/data disks have auto-delete disabled
- The instance will be stopped if running before deletion
- Deleting an already-deleted instance is treated as success (idempotent)`
}

func (d *DeleteVMInstance) Icon() string {
	return "trash-2"
}

func (d *DeleteVMInstance) Color() string {
	return "red"
}

func (d *DeleteVMInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteVMInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The GCP zone where the VM instance is located (e.g. us-central1-a).",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeZone,
				},
			},
		},
		{
			Name:        "instance",
			Label:       "VM Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The VM instance to delete.",
			Placeholder: "Select instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
					Parameters: []configuration.ParameterRef{
						{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
					},
				},
			},
		},
	}
}

func (d *DeleteVMInstance) Setup(ctx core.SetupContext) error {
	spec := DeleteVMInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Zone) == "" {
		return errors.New("zone is required")
	}

	if strings.TrimSpace(spec.Instance) == "" {
		return errors.New("instance is required")
	}

	return d.resolveNodeMetadata(ctx, spec)
}

func (d *DeleteVMInstance) resolveNodeMetadata(ctx core.SetupContext, spec DeleteVMInstanceSpec) error {
	zone := lastSegment(strings.TrimSpace(spec.Zone))
	instanceName := strings.TrimSpace(spec.Instance)

	// If the instance is an expression, skip the API call and store what we have
	if strings.Contains(instanceName, "{{") {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: instanceName,
			Zone:         zone,
		})
	}

	// If metadata is already set for the same instance, skip the API call
	var existing VMInstanceNodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err == nil &&
		existing.InstanceName == instanceName && existing.Zone == zone {
		return nil
	}

	// No integration available (e.g. in tests without credentials) — store what we have
	if ctx.Integration == nil {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: instanceName,
			Zone:         zone,
		})
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: instanceName,
			Zone:         zone,
		})
	}

	body, err := GetInstance(context.Background(), client, client.ProjectID(), zone, instanceName)
	if err != nil {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: instanceName,
			Zone:         zone,
		})
	}

	payload, err := InstancePayloadFromGetResponse(body, zone)
	if err != nil {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: instanceName,
			Zone:         zone,
		})
	}

	resolvedName, _ := payload["name"].(string)
	if resolvedName == "" {
		resolvedName = instanceName
	}

	return ctx.Metadata.Set(VMInstanceNodeMetadata{
		InstanceName: resolvedName,
		Zone:         zone,
	})
}

func (d *DeleteVMInstance) Execute(ctx core.ExecutionContext) error {
	spec := DeleteVMInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	zone := lastSegment(strings.TrimSpace(spec.Zone))
	instanceName := strings.TrimSpace(spec.Instance)

	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	project := client.ProjectID()
	callCtx := context.Background()

	path := fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, zone, instanceName)
	body, err := client.Delete(callCtx, path)
	if err != nil {
		var apiErr *gcpcommon.GCPAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			// Already deleted — emit success (idempotent)
			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				"gcp.compute.vmInstance.deleted",
				[]any{map[string]any{"instanceName": instanceName, "zone": zone}},
			)
		}
		return fmt.Errorf("failed to delete VM instance: %v", err)
	}

	var opResp struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &opResp); err != nil || opResp.Name == "" {
		// If we can't parse the operation, the delete may have already completed
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"gcp.compute.vmInstance.deleted",
			[]any{map[string]any{"instanceName": instanceName, "zone": zone}},
		)
	}

	if err := WaitForZoneOperation(callCtx, client, project, zone, lastSegment(opResp.Name)); err != nil {
		return fmt.Errorf("error waiting for delete operation: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.vmInstance.deleted",
		[]any{map[string]any{"instanceName": instanceName, "zone": zone}},
	)
}

func (d *DeleteVMInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteVMInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteVMInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteVMInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeleteVMInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteVMInstance) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
