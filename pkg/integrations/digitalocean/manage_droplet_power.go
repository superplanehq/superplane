package digitalocean

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const actionPollInterval = 10 * time.Second

type ManageDropletPower struct{}

type ManageDropletPowerSpec struct {
	DropletID string `json:"dropletId" mapstructure:"dropletId"`
	Action    string `json:"action" mapstructure:"action"`
}

var validPowerActions = map[string]bool{
	"power_on":    true,
	"shutdown":    true,
	"reboot":      true,
	"power_cycle": true,
	"power_off":   true,
}

func (c *ManageDropletPower) Name() string {
	return "digitalocean.manageDropletPower"
}

func (c *ManageDropletPower) Label() string {
	return "Manage Droplet Power"
}

func (c *ManageDropletPower) Description() string {
	return "Run power operations on a DigitalOcean Droplet"
}

func (c *ManageDropletPower) Documentation() string {
	return `The Manage Droplet Power component runs power operations on a DigitalOcean Droplet and waits for completion.

## How It Works

1. Sends a power action (power_on, shutdown, reboot, power_cycle, or power_off) to the specified droplet
2. Polls the action until it completes
3. Emits on the default output when the action finishes. If the action fails, the execution errors.

## Configuration

- **Droplet ID**: The ID of the droplet (required, supports expressions)
- **Action**: The power action to perform (required). One of: power_on, shutdown, reboot, power_cycle, power_off

## Output

Returns the completed action details:
- **actionId**: The DigitalOcean action ID
- **dropletId**: The droplet ID
- **action**: The action type performed
- **status**: The final status of the action`
}

func (c *ManageDropletPower) Icon() string {
	return "server"
}

func (c *ManageDropletPower) Color() string {
	return "gray"
}

func (c *ManageDropletPower) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ManageDropletPower) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"actionId":  12345,
			"dropletId": 98765432,
			"action":    "power_on",
			"status":    "completed",
		},
	}
}

func (c *ManageDropletPower) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dropletId",
			Label:       "Droplet ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The ID of the droplet",
		},
		{
			Name:        "action",
			Label:       "Power Action",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The power action to perform",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Power On", Value: "power_on"},
						{Label: "Shutdown", Value: "shutdown"},
						{Label: "Reboot", Value: "reboot"},
						{Label: "Power Cycle", Value: "power_cycle"},
						{Label: "Power Off", Value: "power_off"},
					},
				},
			},
		},
	}
}

func (c *ManageDropletPower) Setup(ctx core.SetupContext) error {
	spec := ManageDropletPowerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.DropletID == "" {
		return fmt.Errorf("dropletId is required")
	}

	if spec.Action == "" {
		return fmt.Errorf("action is required")
	}

	if !validPowerActions[spec.Action] {
		return fmt.Errorf("invalid action %q: must be one of power_on, shutdown, reboot, power_cycle, power_off", spec.Action)
	}

	return nil
}

func (c *ManageDropletPower) Execute(ctx core.ExecutionContext) error {
	spec := ManageDropletPowerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	dropletID, err := resolveIntID(ctx.Configuration, "dropletId")
	if err != nil {
		return err
	}

	if !validPowerActions[spec.Action] {
		return fmt.Errorf("invalid action %q", spec.Action)
	}

	if err := ctx.Metadata.Set(map[string]any{"dropletId": dropletID, "action": spec.Action}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	action, err := client.PerformDropletAction(dropletID, spec.Action)
	if err != nil {
		return fmt.Errorf("failed to perform action %q: %v", spec.Action, err)
	}

	if err := ctx.Metadata.Set(map[string]any{
		"dropletId": dropletID,
		"action":    spec.Action,
		"actionId":  action.ID,
	}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, actionPollInterval)
}

func (c *ManageDropletPower) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ManageDropletPower) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ManageDropletPower) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *ManageDropletPower) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata struct {
		DropletID int    `mapstructure:"dropletId"`
		Action    string `mapstructure:"action"`
		ActionID  int    `mapstructure:"actionId"`
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
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, actionPollInterval)
	}

	switch action.Status {
	case "completed":
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.droplet.power_action_completed",
			[]any{map[string]any{
				"actionId":  metadata.ActionID,
				"dropletId": metadata.DropletID,
				"action":    metadata.Action,
				"status":    "completed",
			}},
		)
	case "in-progress":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, actionPollInterval)
	case "errored":
		return fmt.Errorf("power action %q failed for droplet %d", metadata.Action, metadata.DropletID)
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, actionPollInterval)
	}
}

func (c *ManageDropletPower) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *ManageDropletPower) Cleanup(ctx core.SetupContext) error {
	return nil
}
