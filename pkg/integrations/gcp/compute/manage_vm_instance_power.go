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

type ManageVMInstancePower struct{}

type ManageVMInstancePowerSpec struct {
	Instance  string `mapstructure:"instance"`
	Operation string `mapstructure:"operation"`
}

// powerOperationEndpoints maps the user-facing operation to the Compute Engine
// instance sub-resource that performs it. Each call returns a zone operation
// that we wait on before reading back the instance state.
var powerOperationEndpoints = map[string]string{
	"power_on":  "start",
	"power_off": "stop",
	"reset":     "reset",
	"suspend":   "suspend",
	"resume":    "resume",
}

func (m *ManageVMInstancePower) Name() string {
	return "gcp.manageVMInstancePower"
}

func (m *ManageVMInstancePower) Label() string {
	return "Compute • Manage VM Power"
}

func (m *ManageVMInstancePower) Description() string {
	return "Perform power operations on a Google Compute Engine VM instance"
}

func (m *ManageVMInstancePower) Documentation() string {
	return `The Manage VM Power component performs power management operations on a Compute Engine VM instance.

## Use Cases

- **Automated restarts**: Reset instances on a schedule or in response to alerts
- **Cost optimization**: Stop instances during non-business hours
- **Maintenance workflows**: Stop instances before updates, start them after completion
- **Recovery procedures**: Reset instances experiencing issues

## Configuration

- **VM Instance**: Pick from the list of VMs in your project, or pass an expression chained from an upstream node (e.g. the ` + "`selfLink`" + ` emitted by ` + "`gcp.createVM`" + `). The selection encodes both the zone and the instance name.
- **Operation**: The power operation to perform (required):
  - **start**: Start a stopped (TERMINATED) instance
  - **stop**: Stop a running instance
  - **reset**: Hard reset a running instance (does not perform a clean shutdown)
  - **suspend**: Suspend a running instance, preserving memory state
  - **resume**: Resume a suspended instance

## Output

Returns the instance state after the operation completes:
- **instanceId**, **name**, **zone**, **status**, **selfLink**, **machineType**, **internalIP**, **externalIP**
- **operation**: The power operation that was performed

## Important Notes

- **reset** is a forced operation and does not perform a clean OS shutdown
- The component waits for the underlying zone operation to complete before emitting
- Operations may take several minutes depending on the instance state`
}

func (m *ManageVMInstancePower) Icon() string {
	return "power"
}

func (m *ManageVMInstancePower) Color() string {
	return "orange"
}

func (m *ManageVMInstancePower) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (m *ManageVMInstancePower) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instance",
			Label:       "VM Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The VM instance to manage. Lists every VM in your project across all zones.",
			Placeholder: "Select instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
		{
			Name:        "operation",
			Label:       "Operation",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The power operation to perform",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Start", Value: "power_on"},
						{Label: "Stop", Value: "power_off"},
						{Label: "Reset (Forced)", Value: "reset"},
						{Label: "Suspend", Value: "suspend"},
						{Label: "Resume", Value: "resume"},
					},
				},
			},
		},
	}
}

func (m *ManageVMInstancePower) Setup(ctx core.SetupContext) error {
	spec := ManageVMInstancePowerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Instance) == "" {
		return errors.New("instance is required")
	}

	if spec.Operation == "" {
		return errors.New("operation is required")
	}

	if _, ok := powerOperationEndpoints[spec.Operation]; !ok {
		return fmt.Errorf("invalid operation %q: must be one of power_on, power_off, reset, suspend, resume", spec.Operation)
	}

	return resolveInstanceNodeMetadata(ctx, spec.Instance)
}

func (m *ManageVMInstancePower) Execute(ctx core.ExecutionContext) error {
	spec := ManageVMInstancePowerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	endpoint, ok := powerOperationEndpoints[spec.Operation]
	if !ok {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("invalid operation %q", spec.Operation))
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

	callCtx := context.Background()
	if err := runInstancePowerOperation(callCtx, client, project, zone, instanceName, endpoint); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to %s VM instance: %v", endpoint, err))
	}

	body, err := GetInstance(callCtx, client, project, zone, instanceName)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read instance after %s: %v", endpoint, err))
	}

	payload, err := InstancePayloadFromGetResponse(body, zone)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse instance: %v", err))
	}
	payload["operation"] = spec.Operation

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		fmt.Sprintf("gcp.compute.vmInstance.power.%s", spec.Operation),
		[]any{payload},
	)
}

// runInstancePowerOperation issues the power sub-resource POST and waits for the
// resulting zone operation to complete.
func runInstancePowerOperation(ctx context.Context, client Client, project, zone, instanceName, endpoint string) error {
	path := fmt.Sprintf("projects/%s/zones/%s/instances/%s/%s", project, zone, instanceName, endpoint)
	body, err := client.Post(ctx, path, nil)
	if err != nil {
		return err
	}

	var opResp struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &opResp); err != nil {
		return fmt.Errorf("parse power operation response: %w", err)
	}
	if opResp.Name == "" {
		return errors.New("power operation response missing operation name")
	}

	return WaitForZoneOperation(ctx, client, project, zone, lastSegment(opResp.Name))
}

func (m *ManageVMInstancePower) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (m *ManageVMInstancePower) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (m *ManageVMInstancePower) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (m *ManageVMInstancePower) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (m *ManageVMInstancePower) Hooks() []core.Hook {
	return []core.Hook{}
}

func (m *ManageVMInstancePower) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
