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
	return "Permanently delete a Google Compute Engine VM instance"
}

func (d *DeleteVMInstance) Documentation() string {
	return `The Delete VM Instance component permanently deletes a Compute Engine VM instance.

## Use Cases

- **Cleanup**: Remove temporary or test VMs after use
- **Cost optimization**: Automatically tear down unused infrastructure
- **Automated workflows**: Delete VMs as part of deployment rollback or cleanup processes
- **Environment management**: Remove ephemeral environments after testing

## Configuration

- **VM Instance**: Pick from the list of VMs in your project, or pass an expression chained from an upstream node (e.g. the ` + "`selfLink`" + ` emitted by ` + "`gcp.createVM`" + `). The selection encodes both the zone and the instance name.

## Output

Returns information about the deleted instance:
- **instanceName**: The name of the instance that was deleted
- **zone**: The zone the instance was in

## Important Notes

- This operation is **permanent** and cannot be undone
- All data on the instance will be lost unless boot/data disks have auto-delete disabled
- The instance will be stopped if running before deletion
- If the instance is not found at the resolved zone/name, the action fails so that misconfigured or stale expressions do not silently mask incomplete cleanup`
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
			Name:        "instance",
			Label:       "VM Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The VM instance to delete. Lists every VM in your project across all zones.",
			Placeholder: "Select instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
	}
}

// parseInstancePath extracts (project, zone, name) from a value of the form
// `zones/<zone>/instances/<name>` (relative path) or a full GCE selfLink URL
// containing `projects/<project>/zones/<zone>/instances/<name>`. The project
// segment is optional — relative paths from the dropdown have no project, but
// chained selfLinks do, and the caller must verify it matches the integration's
// bound project before issuing the delete.
func parseInstancePath(value string) (project, zone, name string, err error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return "", "", "", errors.New("instance is required")
	}
	if idx := strings.Index(s, "projects/"); idx >= 0 {
		rest := s[idx+len("projects/"):]
		if slash := strings.Index(rest, "/"); slash > 0 {
			project = rest[:slash]
		}
	}
	idx := strings.Index(s, "zones/")
	if idx < 0 {
		return "", "", "", fmt.Errorf("instance %q must be a path like zones/<zone>/instances/<name> or a GCE selfLink URL", value)
	}
	rest := s[idx+len("zones/"):]
	slash := strings.Index(rest, "/")
	if slash <= 0 {
		return "", "", "", fmt.Errorf("instance %q is missing a zone segment", value)
	}
	zone = rest[:slash]
	after := rest[slash+1:]
	const prefix = "instances/"
	if !strings.HasPrefix(after, prefix) {
		return "", "", "", fmt.Errorf("instance %q is missing an instances/ segment", value)
	}
	name = after[len(prefix):]
	if q := strings.IndexAny(name, "/?#"); q >= 0 {
		name = name[:q]
	}
	if zone == "" || name == "" {
		return "", "", "", fmt.Errorf("instance %q is missing a zone or name", value)
	}
	return project, zone, name, nil
}

func (d *DeleteVMInstance) Setup(ctx core.SetupContext) error {
	spec := DeleteVMInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	instanceValue := strings.TrimSpace(spec.Instance)
	if instanceValue == "" {
		return errors.New("instance is required")
	}

	return d.resolveNodeMetadata(ctx, instanceValue)
}

func (d *DeleteVMInstance) resolveNodeMetadata(ctx core.SetupContext, instanceValue string) error {
	// Expressions are resolved at execution time. Store the raw value so the UI
	// can still display something meaningful in the collapsed node.
	if strings.Contains(instanceValue, "{{") {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: instanceValue,
		})
	}

	_, zone, name, err := parseInstancePath(instanceValue)
	if err != nil {
		return err
	}

	// If metadata is already set for the same instance, skip the API call
	var existing VMInstanceNodeMetadata
	if decErr := mapstructure.Decode(ctx.Metadata.Get(), &existing); decErr == nil &&
		existing.InstanceName == name && existing.Zone == zone {
		return nil
	}

	// No integration available (e.g. in tests without credentials) — store what we have
	if ctx.Integration == nil {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: name,
			Zone:         zone,
		})
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: name,
			Zone:         zone,
		})
	}

	body, err := GetInstance(context.Background(), client, client.ProjectID(), zone, name)
	if err != nil {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: name,
			Zone:         zone,
		})
	}

	payload, err := InstancePayloadFromGetResponse(body, zone)
	if err != nil {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: name,
			Zone:         zone,
		})
	}

	resolvedName, _ := payload["name"].(string)
	if resolvedName == "" {
		resolvedName = name
	}

	return ctx.Metadata.Set(VMInstanceNodeMetadata{
		InstanceName: resolvedName,
		Zone:         zone,
	})
}

func (d *DeleteVMInstance) Execute(ctx core.ExecutionContext) error {
	spec := DeleteVMInstanceSpec{}
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
	// the integration's bound project. Silently rewriting to the integration
	// project could delete a same-named VM in the wrong place.
	if urlProject != "" && urlProject != project {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(
			"instance belongs to project %q but this GCP integration is bound to project %q; cross-project deletes are not supported",
			urlProject, project,
		))
	}

	callCtx := context.Background()
	path := fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, zone, instanceName)
	body, err := client.Delete(callCtx, path)
	if err != nil {
		// Surface the underlying API error (including 404s). A 404 may indicate
		// the instance is genuinely already gone, but it can also be caused by a
		// stale expression or a renamed resource — in which case the VM may
		// still exist. Failing loudly lets the workflow author decide how to
		// handle it explicitly rather than silently claiming success.
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete VM instance: %v", err))
	}

	var opResp struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &opResp); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse delete operation response: %v", err))
	}
	if opResp.Name == "" {
		return ctx.ExecutionState.Fail("error", "delete operation response missing operation name; cannot confirm deletion")
	}

	if err := WaitForZoneOperation(callCtx, client, project, zone, lastSegment(opResp.Name)); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("error waiting for delete operation: %v", err))
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
