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

const DeleteJiraHeartbeatPayloadType = "jira.heartbeat.deleted"

type DeleteHeartbeat struct{}

type DeleteHeartbeatSpec struct {
	Team      string `json:"team" mapstructure:"team"`
	Heartbeat string `json:"heartbeat" mapstructure:"heartbeat"`
}

type DeleteHeartbeatNodeMetadata struct {
	TeamName string `json:"teamName,omitempty"`
}

func (c *DeleteHeartbeat) Name() string {
	return "jira.deleteHeartbeat"
}

func (c *DeleteHeartbeat) Label() string {
	return "Delete Heartbeat"
}

func (c *DeleteHeartbeat) Description() string {
	return "Delete a Jira Service Management Operations heartbeat"
}

func (c *DeleteHeartbeat) Documentation() string {
	return `The Delete Heartbeat component removes a heartbeat from Jira Service Management Operations.

## Use Cases

- **Decommissioning**: Remove heartbeats when systems are retired
- **Test cleanup**: Delete heartbeats created during automation tests
- **Lifecycle management**: Tear down monitoring when workflows are disabled

## Configuration

- **Team**: Team that owns the heartbeat
- **Heartbeat**: Name of the heartbeat to delete

## Output

Confirms deletion with **deleted** set to true and the **name** of the removed heartbeat.

## Notes

- Requires Jira Service Management Operations with heartbeats enabled.
- Deletion is permanent; recreate the heartbeat if needed.`
}

func (c *DeleteHeartbeat) Icon() string {
	return "jira"
}

func (c *DeleteHeartbeat) Color() string {
	return "red"
}

func (c *DeleteHeartbeat) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteHeartbeat) Configuration() []configuration.Field {
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
			Description: "Heartbeat to delete (listed from the selected team)",
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

func (c *DeleteHeartbeat) Setup(ctx core.SetupContext) error {
	spec := DeleteHeartbeatSpec{}
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

	return ctx.Metadata.Set(DeleteHeartbeatNodeMetadata{TeamName: resolveOpsTeamName(ctx, cloudID, spec.Team)})
}

func (c *DeleteHeartbeat) Execute(ctx core.ExecutionContext) error {
	spec := DeleteHeartbeatSpec{}
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

	if err := client.DeleteHeartbeat(cloudID, teamID, name); err != nil {
		return fmt.Errorf("failed to delete heartbeat: %w", err)
	}

	payload := map[string]any{
		"deleted": true,
		"name":    name,
	}
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeleteJiraHeartbeatPayloadType,
		[]any{payload},
	)
}

func (c *DeleteHeartbeat) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteHeartbeat) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteHeartbeat) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteHeartbeat) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteHeartbeat) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteHeartbeat) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func resolveOpsTeamName(ctx core.SetupContext, cloudID, teamID string) string {
	teamID = strings.TrimSpace(teamID)
	if teamID == "" || ctx.HTTP == nil || cloudID == "" {
		return ""
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ""
	}
	teams, err := client.ListOpsTeams(cloudID)
	if err != nil {
		return ""
	}
	for _, t := range teams {
		if t.TeamID == teamID {
			return strings.TrimSpace(t.TeamName)
		}
	}
	return ""
}
