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
)

type UpdateVMInstanceType struct{}

type UpdateVMInstanceTypeSpec struct {
	Instance           string `mapstructure:"instance"`
	MachineType        string `mapstructure:"machineType"`
	RestartAfterUpdate *bool  `mapstructure:"restartAfterUpdate"`
}

func (u *UpdateVMInstanceType) Name() string {
	return "gcp.updateVMInstanceType"
}

func (u *UpdateVMInstanceType) Label() string {
	return "Compute • Update VM Machine Type"
}

func (u *UpdateVMInstanceType) Description() string {
	return "Change the machine type of a Google Compute Engine VM instance"
}

func (u *UpdateVMInstanceType) Documentation() string {
	return `The Update VM Machine Type component changes the machine type (size) of an existing Compute Engine VM instance.

## Use Cases

- **Vertical scaling**: Resize an instance up or down in response to load
- **Cost optimization**: Move to a smaller machine type during off-peak hours
- **Right-sizing**: Adjust machine type based on observed utilization

## Configuration

- **VM Instance**: Pick from the list of VMs in your project, or pass an expression chained from an upstream node (e.g. the ` + "`selfLink`" + ` emitted by ` + "`gcp.createVM`" + `). The selection encodes both the zone and the instance name.
- **Machine Type**: The new machine type name, e.g. ` + "`e2-medium`" + ` or ` + "`n2-standard-4`" + ` (required, supports expressions).
- **Restart after update**: Whether to start the instance again after the machine type is changed. Enabled by default. Compute Engine requires the instance to be stopped to change its machine type, so a running instance is always stopped first.

## Output

Returns the instance state after the update completes:
- **instanceId**, **name**, **zone**, **status**, **selfLink**, **machineType**, **internalIP**, **externalIP**

## Important Notes

- Changing the machine type requires the instance to be **stopped (TERMINATED)**. A running instance is stopped automatically before the change.
- If **Restart after update** is enabled, the instance is started again once the new machine type is applied.
- The new machine type must be available in the instance's zone.
- The component waits for each underlying zone operation to complete before proceeding.`
}

func (u *UpdateVMInstanceType) Icon() string {
	return "cpu"
}

func (u *UpdateVMInstanceType) Color() string {
	return "blue"
}

func (u *UpdateVMInstanceType) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateVMInstanceType) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instance",
			Label:       "VM Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The VM instance to update. Lists every VM in your project across all zones.",
			Placeholder: "Select instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
		{
			Name:        "machineType",
			Label:       "Machine Type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The new machine type to resize the instance to. Lists machine types available in the instance's zone.",
			Placeholder: "Select machine type",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstanceMachineType,
					Parameters: []configuration.ParameterRef{
						{Name: "instance", ValueFrom: &configuration.ParameterValueFrom{Field: "instance"}},
					},
				},
			},
		},
		{
			Name:        "restartAfterUpdate",
			Label:       "Restart after update",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Start the instance again after changing the machine type.",
		},
	}
}

func (u *UpdateVMInstanceType) Setup(ctx core.SetupContext) error {
	spec := UpdateVMInstanceTypeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Instance) == "" {
		return errors.New("instance is required")
	}

	if strings.TrimSpace(spec.MachineType) == "" {
		return errors.New("machineType is required")
	}

	return resolveInstanceNodeMetadata(ctx, spec.Instance)
}

func (u *UpdateVMInstanceType) Execute(ctx core.ExecutionContext) error {
	spec := UpdateVMInstanceTypeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	machineType := strings.TrimSpace(spec.MachineType)
	if machineType == "" {
		return ctx.ExecutionState.Fail("error", "machineType is required")
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
	if urlProject != "" && urlProject != project {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(
			"instance belongs to project %q but this GCP integration is bound to project %q; cross-project operations are not supported",
			urlProject, project,
		))
	}

	// setMachineType expects a zone-qualified machine type path. Accept either a
	// bare name (e.g. e2-medium) or a full path, mirroring gcp.createVM.
	if !strings.Contains(machineType, "/") {
		machineType = fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType)
	}

	callCtx := context.Background()

	body, err := GetInstance(callCtx, client, project, zone, instanceName)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read instance: %v", err))
	}

	var current struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &current); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse instance: %v", err))
	}

	// Compute Engine only allows changing the machine type while the instance is
	// stopped. Stop it first when it is not already TERMINATED.
	if current.Status != "TERMINATED" {
		if err := runInstancePowerOperation(callCtx, client, project, zone, instanceName, "stop"); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to stop instance before update: %v", err))
		}
	}

	if err := setMachineType(callCtx, client, project, zone, instanceName, machineType); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to update machine type: %v", err))
	}

	restart := spec.RestartAfterUpdate == nil || *spec.RestartAfterUpdate
	if restart {
		if err := runInstancePowerOperation(callCtx, client, project, zone, instanceName, "start"); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to start instance after update: %v", err))
		}
	}

	updated, err := GetInstance(callCtx, client, project, zone, instanceName)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read instance after update: %v", err))
	}

	payload, err := InstancePayloadFromGetResponse(updated, zone)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse updated instance: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.vmInstance.machineTypeUpdated",
		[]any{payload},
	)
}

// setMachineType issues the setMachineType POST and waits for the zone operation.
func setMachineType(ctx context.Context, client Client, project, zone, instanceName, machineType string) error {
	path := fmt.Sprintf("projects/%s/zones/%s/instances/%s/setMachineType", project, zone, instanceName)
	body, err := client.Post(ctx, path, map[string]any{"machineType": machineType})
	if err != nil {
		return err
	}

	var opResp struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &opResp); err != nil {
		return fmt.Errorf("parse setMachineType operation response: %w", err)
	}
	if opResp.Name == "" {
		return errors.New("setMachineType operation response missing operation name")
	}

	return WaitForZoneOperation(ctx, client, project, zone, lastSegment(opResp.Name))
}

func (u *UpdateVMInstanceType) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateVMInstanceType) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateVMInstanceType) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (u *UpdateVMInstanceType) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (u *UpdateVMInstanceType) Hooks() []core.Hook {
	return []core.Hook{}
}

func (u *UpdateVMInstanceType) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
