package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const actionPollInterval = 5 * time.Second

type ManageDropletPower struct{}

type ManageDropletPowerSpec struct {
	Droplet   string `json:"droplet"`
	Operation string `json:"operation"`
}

func (m *ManageDropletPower) Name() string {
	return "digitalocean.manageDropletPower"
}

func (m *ManageDropletPower) Label() string {
	return "Manage Droplet Power"
}

func (m *ManageDropletPower) Description() string {
	return "Perform power operations on a DigitalOcean Droplet"
}

func (m *ManageDropletPower) Documentation() string {
	return `The Manage Droplet Power component performs power management operations on a droplet.

## Use Cases

- **Automated restarts**: Reboot droplets on a schedule or in response to alerts
- **Cost optimization**: Power off droplets during non-business hours
- **Maintenance workflows**: Shutdown droplets before updates, power on after completion
- **Recovery procedures**: Power cycle droplets experiencing issues

## Configuration

- **Droplet**: The droplet to manage (required, supports expressions)
- **Operation**: The power operation to perform (required):
  - **power_on**: Power on a powered-off droplet
  - **power_off**: Power off a running droplet (forced shutdown)
  - **shutdown**: Gracefully shutdown a running droplet
  - **reboot**: Gracefully reboot a running droplet
  - **power_cycle**: Power cycle a droplet (forced reboot)

## Output

Returns the action result including:
- **id**: Action ID
- **status**: Final action status (completed or errored)
- **type**: Type of action performed
- **started_at**: When the action started
- **completed_at**: When the action completed
- **resource_id**: Droplet ID
- **region**: Region slug

## Important Notes

- **power_off** and **power_cycle** are forced operations and may cause data loss
- **shutdown** and **reboot** are graceful and wait for the OS to complete the operation
- The component waits for the action to complete before emitting
- Actions may take several minutes depending on the droplet state`
}

func (m *ManageDropletPower) Icon() string {
	return "power"
}

func (m *ManageDropletPower) Color() string {
	return "orange"
}

func (m *ManageDropletPower) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (m *ManageDropletPower) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "droplet",
			Label:       "Droplet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The droplet to manage",
			Placeholder: "Select droplet",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "droplet",
					UseNameAsValue: false,
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
						{Label: "Power On", Value: "power_on"},
						{Label: "Power Off (Forced)", Value: "power_off"},
						{Label: "Shutdown (Graceful)", Value: "shutdown"},
						{Label: "Reboot (Graceful)", Value: "reboot"},
						{Label: "Power Cycle (Forced)", Value: "power_cycle"},
					},
				},
			},
		},
	}
}

func (m *ManageDropletPower) Setup(ctx core.SetupContext) error {
	spec := ManageDropletPowerSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Droplet == "" {
		return errors.New("droplet is required")
	}

	if spec.Operation == "" {
		return errors.New("operation is required")
	}

	validOperations := map[string]bool{
		"power_on":    true,
		"power_off":   true,
		"shutdown":    true,
		"reboot":      true,
		"power_cycle": true,
	}

	if !validOperations[spec.Operation] {
		return fmt.Errorf("invalid operation %q: must be one of power_on, power_off, shutdown, reboot, power_cycle", spec.Operation)
	}

	err = resolveDropletMetadata(ctx, spec.Droplet)
	if err != nil {
		return fmt.Errorf("error resolving droplet metadata: %v", err)
	}

	return nil
}

func (m *ManageDropletPower) Execute(ctx core.ExecutionContext) error {
	spec := ManageDropletPowerSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	dropletID, err := parseDropletID(spec.Droplet)
	if err != nil {
		return fmt.Errorf("invalid droplet ID %q: %w", spec.Droplet, err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	action, err := client.PostDropletAction(dropletID, spec.Operation)
	if err != nil {
		return fmt.Errorf("failed to initiate power action: %v", err)
	}

	err = ctx.Metadata.Set(map[string]any{
		"actionID":  action.ID,
		"dropletID": dropletID,
		"operation": spec.Operation,
	})
	if err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, actionPollInterval)
}

func (m *ManageDropletPower) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (m *ManageDropletPower) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (m *ManageDropletPower) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (m *ManageDropletPower) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata struct {
		ActionID  int    `mapstructure:"actionID"`
		DropletID int    `mapstructure:"dropletID"`
		Operation string `mapstructure:"operation"`
	}

	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	action, err := client.GetAction(metadata.ActionID)
	if err != nil {
		return fmt.Errorf("failed to get action status: %v", err)
	}

	switch action.Status {
	case "completed":
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			fmt.Sprintf("digitalocean.droplet.power.%s", metadata.Operation),
			[]any{action},
		)
	case "in-progress":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, actionPollInterval)
	case "errored":
		return fmt.Errorf("power action failed with status: errored")
	default:
		return fmt.Errorf("action reached unexpected status %q", action.Status)
	}
}

func (m *ManageDropletPower) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (m *ManageDropletPower) Cleanup(ctx core.SetupContext) error {
	return nil
}
