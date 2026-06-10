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

type DeleteStaticIP struct{}

type DeleteStaticIPSpec struct {
	Address string `mapstructure:"address"`
}

func (d *DeleteStaticIP) Name() string {
	return "gcp.compute.deleteStaticIP"
}

func (d *DeleteStaticIP) Label() string {
	return "Compute • Delete Static IP"
}

func (d *DeleteStaticIP) Description() string {
	return "Release a regional external static IP address from a Google Cloud project"
}

func (d *DeleteStaticIP) Documentation() string {
	return `The Delete Static IP component releases (deletes) a regional external static IP reservation.

## Use Cases

- **Cost optimization**: Release reserved IPs that are no longer needed (idle reserved IPs are billed)
- **Cleanup**: Tear down addresses as part of environment teardown

## Configuration

- **Static IP**: Pick from the reserved external IPs across all regions, or pass an expression chained from an upstream node (e.g. the ` + "`selfLink`" + ` emitted by ` + "`gcp.compute.createStaticIP`" + `). The selection encodes both the region and the address name.

## Output

Returns the released address:
- **name**: The name of the address that was released
- **region**: The region it was in

## Important Notes

- A static IP that is still **attached** to a VM cannot be deleted — detach it first with **Manage Static IP**
- If the address is not found at the resolved region/name, the action fails so that misconfigured or stale expressions do not silently mask incomplete cleanup`
}

func (d *DeleteStaticIP) Icon() string {
	return "trash-2"
}

func (d *DeleteStaticIP) Color() string {
	return "red"
}

func (d *DeleteStaticIP) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteStaticIP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "address",
			Label:       "Static IP",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The reserved external IP to release. Lists external static IPs across all regions.",
			Placeholder: "Select static IP",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeStaticIP,
				},
			},
		},
	}
}

func (d *DeleteStaticIP) Setup(ctx core.SetupContext) error {
	spec := DeleteStaticIPSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	address := strings.TrimSpace(spec.Address)
	if address == "" {
		return errors.New("address is required")
	}

	// Expressions are resolved at execution time; only validate the shape of a
	// literal selection here.
	if strings.Contains(address, "{{") {
		return nil
	}

	if _, _, _, err := parseAddressPath(address); err != nil {
		return err
	}

	return nil
}

func (d *DeleteStaticIP) Execute(ctx core.ExecutionContext) error {
	spec := DeleteStaticIPSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	urlProject, region, name, err := parseAddressPath(spec.Address)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	project := client.ProjectID()
	// A selfLink carrying an explicit project must match the integration's bound
	// project; silently rewriting it could release an address in the wrong place.
	if urlProject != "" && urlProject != project {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(
			"static IP belongs to project %q but this GCP integration is bound to project %q; cross-project deletes are not supported",
			urlProject, project,
		))
	}

	callCtx := context.Background()
	path := fmt.Sprintf("projects/%s/regions/%s/addresses/%s", project, region, name)
	respBody, err := client.Delete(callCtx, path)
	if err != nil {
		// Surface the underlying API error (including 404s). A 404 may mean the
		// address is already gone, but it can also be a stale expression — fail
		// loudly so the workflow author handles it explicitly.
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete static IP: %v", err))
	}

	var opResp struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &opResp); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse delete operation response: %v", err))
	}
	if opResp.Name == "" {
		return ctx.ExecutionState.Fail("error", "delete operation response missing operation name; cannot confirm deletion")
	}

	if err := WaitForRegionOperation(callCtx, client, project, region, lastSegment(opResp.Name)); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("error waiting for delete operation: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.staticIP.deleted",
		[]any{map[string]any{"name": name, "region": region}},
	)
}

func (d *DeleteStaticIP) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteStaticIP) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteStaticIP) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteStaticIP) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeleteStaticIP) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteStaticIP) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
