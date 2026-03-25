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

const reservedIPPollInterval = 5 * time.Second

type AssignReservedIP struct{}

type AssignReservedIPSpec struct {
	ReservedIP string `json:"reservedIP" mapstructure:"reservedIP"`
	Action     string `json:"action" mapstructure:"action"`
	Droplet    string `json:"droplet" mapstructure:"droplet"`
}

func (a *AssignReservedIP) Name() string {
	return "digitalocean.assignReservedIP"
}

func (a *AssignReservedIP) Label() string {
	return "Assign Reserved IP"
}

func (a *AssignReservedIP) Description() string {
	return "Assign or unassign a DigitalOcean Reserved IP to a Droplet"
}

func (a *AssignReservedIP) Documentation() string {
	return `The Assign Reserved IP component assigns or unassigns a DigitalOcean Reserved IP to a droplet.

## Use Cases

- **Blue/green deployments**: Reassign a reserved IP to the new deployment with zero downtime
- **Failover**: Quickly reassign a reserved IP from a failed droplet to a healthy replacement
- **Maintenance**: Temporarily unassign a reserved IP while a droplet is being serviced

## Configuration

- **Reserved IP**: The reserved IP address to manage (required)
- **Action**: The operation to perform: assign or unassign (required)
- **Droplet ID**: The target droplet for the assignment (required when action is assign)

## Output

Returns the action result including:
- **id**: Action ID
- **status**: Final action status (completed)
- **type**: Type of action performed (assign or unassign)
- **started_at**: When the action started
- **completed_at**: When the action completed
- **resource_id**: Reserved IP resource identifier

## Important Notes

- The component polls until the action completes
- For **assign**, the reserved IP will be unassigned from any current droplet first
- For **unassign**, the **Droplet ID** field is ignored`
}

func (a *AssignReservedIP) Icon() string {
	return "globe"
}

func (a *AssignReservedIP) Color() string {
	return "green"
}

func (a *AssignReservedIP) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (a *AssignReservedIP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "reservedIP",
			Label:       "Reserved IP",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The reserved IP address to manage",
			Placeholder: "Select reserved IP",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "reserved_ip",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "action",
			Label:       "Action",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The operation to perform on the reserved IP",
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
			Name:        "droplet",
			Label:       "Droplet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The droplet to assign the reserved IP to (required for assign action)",
			Placeholder: "Select droplet",
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "action", Values: []string{"assign"}},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "action", Values: []string{"assign"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "droplet",
					UseNameAsValue: false,
				},
			},
		},
	}
}

func (a *AssignReservedIP) Setup(ctx core.SetupContext) error {
	spec := AssignReservedIPSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.ReservedIP == "" {
		return errors.New("reservedIP is required")
	}

	if spec.Action == "" {
		return errors.New("action is required")
	}

	if spec.Action != "assign" && spec.Action != "unassign" {
		return fmt.Errorf("invalid action %q: must be assign or unassign", spec.Action)
	}

	if spec.Action == "assign" && spec.Droplet == "" {
		return errors.New("droplet is required when action is assign")
	}

	return nil
}

func (a *AssignReservedIP) Execute(ctx core.ExecutionContext) error {
	spec := AssignReservedIPSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	var dropletIDPtr *int
	if spec.Action == "assign" {
		id, err := parseDropletID(spec.Droplet)
		if err != nil {
			return fmt.Errorf("invalid droplet ID %q: %w", spec.Droplet, err)
		}
		dropletIDPtr = &id
	}

	action, err := client.PostReservedIPAction(spec.ReservedIP, spec.Action, dropletIDPtr)
	if err != nil {
		return fmt.Errorf("failed to initiate reserved IP action: %v", err)
	}

	if err := ctx.Metadata.Set(map[string]any{
		"actionID":   action.ID,
		"reservedIP": spec.ReservedIP,
		"action":     spec.Action,
	}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, reservedIPPollInterval)
}

func (a *AssignReservedIP) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (a *AssignReservedIP) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *AssignReservedIP) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (a *AssignReservedIP) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata struct {
		ActionID   int    `mapstructure:"actionID"`
		ReservedIP string `mapstructure:"reservedIP"`
		Action     string `mapstructure:"action"`
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
			fmt.Sprintf("digitalocean.reservedip.%s", metadata.Action),
			[]any{action},
		)
	case "in-progress":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, reservedIPPollInterval)
	case "errored":
		return fmt.Errorf("reserved IP action failed with status: errored")
	default:
		return fmt.Errorf("action reached unexpected status %q", action.Status)
	}
}

func (a *AssignReservedIP) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (a *AssignReservedIP) Cleanup(ctx core.SetupContext) error {
	return nil
}
