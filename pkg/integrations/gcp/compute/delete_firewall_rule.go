package compute

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteFirewall struct{}

type DeleteFirewallSpec struct {
	Firewall string `mapstructure:"firewall"`
}

func (d *DeleteFirewall) Name() string {
	return "gcp.compute.deleteFirewallRule"
}

func (d *DeleteFirewall) Label() string {
	return "Compute • Delete Firewall Rule"
}

func (d *DeleteFirewall) Description() string {
	return "Permanently delete a VPC firewall rule"
}

func (d *DeleteFirewall) Documentation() string {
	return `The Delete Firewall Rule component permanently deletes a VPC firewall rule.

## Use Cases

- **Cleanup**: Remove a firewall rule created by **Create Firewall Rule**
- **Tighten security**: Remove an overly permissive rule as part of a workflow
- **Lifecycle automation**: Remove temporary access rules once they are no longer needed

## Configuration

- **Firewall rule**: The firewall rule to delete. Pick from the list of rules in your project, or pass an expression chained from an upstream node (e.g. the ` + "`selfLink`" + ` emitted by ` + "`gcp.compute.createFirewallRule`" + `) (required)

## Output

Returns the name of the deleted firewall rule.

## Important Notes

- This operation is **permanent** and cannot be undone.
- If the firewall rule is not found, the action fails so that misconfigured or stale expressions do not silently mask incomplete cleanup.
- Requires the ` + "`roles/compute.securityAdmin`" + ` IAM role (or ` + "`roles/compute.admin`" + `).`
}

func (d *DeleteFirewall) Icon() string {
	return "trash-2"
}

func (d *DeleteFirewall) Color() string {
	return "red"
}

func (d *DeleteFirewall) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteFirewall) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "firewall",
			Label:       "Firewall rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The firewall rule to delete. Lists every firewall rule in your project.",
			Placeholder: "Select firewall rule",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeFirewall},
			},
		},
	}
}

func (d *DeleteFirewall) Setup(ctx core.SetupContext) error {
	spec := DeleteFirewallSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Firewall) == "" {
		return errors.New("firewall rule is required")
	}
	return resolveFirewallNodeMetadata(ctx, spec.Firewall)
}

func (d *DeleteFirewall) Execute(ctx core.ExecutionContext) error {
	spec := DeleteFirewallSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	urlProject, name, err := parseFirewallPath(spec.Firewall)
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
			"firewall rule belongs to project %q but this GCP integration is bound to project %q; cross-project deletes are not supported",
			urlProject, project,
		))
	}

	callCtx := context.Background()
	path := fmt.Sprintf("projects/%s/global/firewalls/%s", project, name)
	body, err := client.Delete(callCtx, path)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete firewall rule: %v", err))
	}

	opName, err := operationNameFromResponse(body, "delete firewall rule")
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := WaitForGlobalOperation(callCtx, client, project, opName); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("error waiting for delete firewall rule operation: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.firewallRule.deleted",
		[]any{map[string]any{"name": name}},
	)
}

func (d *DeleteFirewall) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteFirewall) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteFirewall) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteFirewall) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeleteFirewall) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteFirewall) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
