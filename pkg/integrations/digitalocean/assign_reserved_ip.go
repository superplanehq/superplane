package digitalocean

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const reservedIPPollInterval = 10 * time.Second

type AssignReservedIP struct{}

type AssignReservedIPSpec struct {
	ReservedIP string `json:"reservedIp" mapstructure:"reservedIp"`
	Action     string `json:"action" mapstructure:"action"`
	DropletID  string `json:"dropletId" mapstructure:"dropletId"`
}

func (c *AssignReservedIP) Name() string {
	return "digitalocean.assignReservedIP"
}

func (c *AssignReservedIP) Label() string {
	return "Assign Reserved IP"
}

func (c *AssignReservedIP) Description() string {
	return "Assign or unassign a DigitalOcean Reserved IP"
}

func (c *AssignReservedIP) Documentation() string {
	return `The Assign Reserved IP component assigns or unassigns a Reserved IP to/from a droplet and waits for completion.

## How It Works

1. Sends an assign or unassign action for the specified Reserved IP
2. Polls the action until it completes
3. Emits on the default output when the action finishes. If the action fails, the execution errors.

## Configuration

- **Reserved IP**: The Reserved IP address (required, supports expressions)
- **Action**: The action to perform — assign or unassign (required)
- **Droplet ID**: The droplet ID to assign the IP to (required for assign, ignored for unassign; supports expressions)

## Output

Returns the action details:
- **actionId**: The DigitalOcean action ID
- **reservedIp**: The Reserved IP address
- **action**: The action performed (assign or unassign)
- **status**: The final status of the action`
}

func (c *AssignReservedIP) Icon() string {
	return "server"
}

func (c *AssignReservedIP) Color() string {
	return "gray"
}

func (c *AssignReservedIP) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AssignReservedIP) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"actionId":   12345,
			"reservedIp": "45.55.96.47",
			"action":     "assign",
			"dropletId":  98765432,
			"status":     "completed",
		},
	}
}

func (c *AssignReservedIP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "reservedIp",
			Label:       "Reserved IP",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The Reserved IP address",
		},
		{
			Name:        "action",
			Label:       "Action",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The action to perform",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Assign", Value: "assign"},
						{Label: "Unassign", Value: "unassign"},
					},
				},
			},
		},
		{
			Name:        "dropletId",
			Label:       "Droplet ID",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Togglable:   true,
			Description: "The droplet ID to assign the IP to (required for assign)",
		},
	}
}

func (c *AssignReservedIP) Setup(ctx core.SetupContext) error {
	spec := AssignReservedIPSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.ReservedIP == "" {
		return fmt.Errorf("reservedIp is required")
	}

	if spec.Action == "" {
		return fmt.Errorf("action is required")
	}

	if spec.Action != "assign" && spec.Action != "unassign" {
		return fmt.Errorf("action must be one of: assign, unassign")
	}

	return nil
}

func (c *AssignReservedIP) Execute(ctx core.ExecutionContext) error {
	spec := AssignReservedIPSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	reservedIP := readStringFromAny(spec.ReservedIP)
	if reservedIP == "" {
		return fmt.Errorf("reservedIp is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	var action *DOAction

	switch spec.Action {
	case "assign":
		dropletID, err := resolveIntID(ctx.Configuration, "dropletId")
		if err != nil {
			return fmt.Errorf("dropletId is required for assign action")
		}

		if err := ctx.Metadata.Set(map[string]any{
			"reservedIp": reservedIP,
			"action":     "assign",
			"dropletId":  dropletID,
		}); err != nil {
			return fmt.Errorf("failed to store metadata: %v", err)
		}

		action, err = client.AssignReservedIP(reservedIP, dropletID)
		if err != nil {
			return fmt.Errorf("failed to assign reserved IP: %v", err)
		}

	case "unassign":
		if err := ctx.Metadata.Set(map[string]any{
			"reservedIp": reservedIP,
			"action":     "unassign",
		}); err != nil {
			return fmt.Errorf("failed to store metadata: %v", err)
		}

		action, err = client.UnassignReservedIP(reservedIP)
		if err != nil {
			return fmt.Errorf("failed to unassign reserved IP: %v", err)
		}

	default:
		return fmt.Errorf("invalid action %q: must be assign or unassign", spec.Action)
	}

	metadata := map[string]any{
		"reservedIp": reservedIP,
		"action":     spec.Action,
		"actionId":   action.ID,
	}

	if spec.Action == "assign" {
		dropletID, _ := resolveIntID(ctx.Configuration, "dropletId")
		metadata["dropletId"] = dropletID
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, reservedIPPollInterval)
}

func (c *AssignReservedIP) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AssignReservedIP) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AssignReservedIP) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *AssignReservedIP) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata struct {
		ReservedIP string `mapstructure:"reservedIp"`
		Action     string `mapstructure:"action"`
		ActionID   int    `mapstructure:"actionId"`
		DropletID  int    `mapstructure:"dropletId"`
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
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, reservedIPPollInterval)
	}

	switch action.Status {
	case "completed":
		payload := map[string]any{
			"actionId":   metadata.ActionID,
			"reservedIp": metadata.ReservedIP,
			"action":     metadata.Action,
			"status":     "completed",
		}
		if metadata.DropletID != 0 {
			payload["dropletId"] = metadata.DropletID
		}
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.reserved_ip.action_completed",
			[]any{payload},
		)
	case "in-progress":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, reservedIPPollInterval)
	case "errored":
		return fmt.Errorf("reserved IP %s action %q failed", metadata.ReservedIP, metadata.Action)
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, reservedIPPollInterval)
	}
}

func (c *AssignReservedIP) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *AssignReservedIP) Cleanup(ctx core.SetupContext) error {
	return nil
}
