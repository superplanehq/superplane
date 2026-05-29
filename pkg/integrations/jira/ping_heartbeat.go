package jira

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const PingJiraHeartbeatPayloadType = "jira.heartbeat.pinged"

type PingHeartbeat struct{}

type PingHeartbeatSpec struct {
	Team      string `json:"team" mapstructure:"team"`
	Heartbeat string `json:"heartbeat" mapstructure:"heartbeat"`
}

type PingHeartbeatNodeMetadata struct {
	TeamName string `json:"teamName,omitempty"`
}

func (c *PingHeartbeat) Name() string {
	return "jira.pingHeartbeat"
}

func (c *PingHeartbeat) Label() string {
	return "Ping Heartbeat"
}

func (c *PingHeartbeat) Description() string {
	return "Send a ping to a Jira Service Management Operations heartbeat"
}

func (c *PingHeartbeat) Documentation() string {
	return `The Ping Heartbeat component reports that a monitored system is alive.

## Use Cases

- **Scheduled jobs**: Ping at the end of a cron workflow so JSM detects missed runs
- **Deploy pipelines**: Confirm a service is up after deployment
- **Agent check-ins**: Let automation prove an external process completed

## Configuration

- **Team**: Team that owns the heartbeat
- **Heartbeat**: Heartbeat name to ping (from the team's heartbeat list)

## Output

Returns the API **message** (for example "PONG - Heartbeat received").

## Notes

- Requires Jira Service Management Operations with heartbeats enabled.
- The heartbeat must already exist (use Create Heartbeat or configure it in JSM).`
}

func (c *PingHeartbeat) Icon() string {
	return "jira"
}

func (c *PingHeartbeat) Color() string {
	return "green"
}

func (c *PingHeartbeat) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PingHeartbeat) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "JSM Operations team that owns the heartbeat",
			Placeholder: "Select a team",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "team",
				},
			},
		},
		{
			Name:        "heartbeat",
			Label:       "Heartbeat",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Heartbeat to ping (listed from the selected team)",
			Placeholder: "Select a heartbeat",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "heartbeat",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "team",
							ValueFrom: &configuration.ParameterValueFrom{Field: "team"},
						},
					},
				},
			},
		},
	}
}

func (c *PingHeartbeat) Setup(ctx core.SetupContext) error {
	spec := PingHeartbeatSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	cloudID, err := resolveCloudID(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if strings.TrimSpace(spec.Team) == "" {
		return fmt.Errorf("team is required")
	}
	if strings.TrimSpace(spec.Heartbeat) == "" {
		return fmt.Errorf("heartbeat is required")
	}

	return ctx.Metadata.Set(PingHeartbeatNodeMetadata{TeamName: resolveOpsTeamName(ctx, cloudID, spec.Team)})
}

func (c *PingHeartbeat) Execute(ctx core.ExecutionContext) error {
	spec := PingHeartbeatSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	cloudID, err := resolveCloudID(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	teamID := strings.TrimSpace(spec.Team)
	name := strings.TrimSpace(spec.Heartbeat)
	if teamID == "" {
		return fmt.Errorf("team is required")
	}
	if name == "" {
		return fmt.Errorf("heartbeat is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	resp, err := client.PingHeartbeat(cloudID, teamID, name)
	if err != nil {
		return fmt.Errorf("failed to ping heartbeat: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PingJiraHeartbeatPayloadType,
		[]any{resp},
	)
}

func (c *PingHeartbeat) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PingHeartbeat) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PingHeartbeat) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *PingHeartbeat) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *PingHeartbeat) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *PingHeartbeat) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
