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

type DeleteLoadBalancer struct{}

type DeleteLoadBalancerSpec struct {
	LoadBalancer string `mapstructure:"loadBalancer"`
}

func (d *DeleteLoadBalancer) Name() string {
	return "gcp.compute.deleteLoadBalancer"
}

func (d *DeleteLoadBalancer) Label() string {
	return "Compute • Delete Load Balancer"
}

func (d *DeleteLoadBalancer) Description() string {
	return "Delete a regional external passthrough Network Load Balancer and its backend service and health check"
}

func (d *DeleteLoadBalancer) Documentation() string {
	return `The Delete Load Balancer component tears down a regional external passthrough Network Load Balancer. You select its **forwarding rule** (the load balancer's entry point), and the component follows the references and deletes the pieces in reverse order:

1. **Forwarding rule** (the public IP + ports)
2. **Backend service** it pointed at
3. **Health check** the backend service used

## Use Cases

- **Cleanup**: Remove a load balancer created by **Create Load Balancer**
- **Teardown**: Decommission a service's front end as part of a workflow

## Configuration

- **Load Balancer**: The regional external forwarding rule that anchors the load balancer (required)

## Output

Returns what was removed: **forwardingRule**, **backendService**, **healthCheck**, **region**.

## Important Notes

- The instance group the backend service referenced is **not** deleted — only the load balancer's own resources
- The health check is removed on a best-effort basis; if it is shared by another load balancer the API keeps it and a note is returned
- Requires the ` + "`roles/compute.loadBalancerAdmin`" + ` IAM role`
}

func (d *DeleteLoadBalancer) Icon() string {
	return "trash-2"
}

func (d *DeleteLoadBalancer) Color() string {
	return "red"
}

func (d *DeleteLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "loadBalancer",
			Label:       "Load Balancer",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The load balancer to delete (its regional external forwarding rule).",
			Placeholder: "Select load balancer",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeForwardingRule},
			},
		},
	}
}

func (d *DeleteLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := DeleteLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.LoadBalancer) == "" {
		return errors.New("load balancer is required")
	}
	// Expressions are resolved at execution time.
	if strings.Contains(spec.LoadBalancer, "{{") {
		return nil
	}
	_, _, _, err := parseRegionalResource(spec.LoadBalancer, "forwardingRules")
	return err
}

func (d *DeleteLoadBalancer) Execute(ctx core.ExecutionContext) error {
	spec := DeleteLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	urlProject, region, frName, err := parseRegionalResource(spec.LoadBalancer, "forwardingRules")
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
			"load balancer belongs to project %q but this GCP integration is bound to project %q; cross-project deletes are not supported",
			urlProject, project,
		))
	}
	callCtx := context.Background()

	// Resolve the backend service (and its health check) before deleting anything.
	var besName, hcName, hcRegion string
	if body, err := client.Get(callCtx, regionalPath(project, region, "forwardingRules", frName)); err == nil {
		var fr forwardingRuleGetResp
		if json.Unmarshal(body, &fr) == nil && fr.BackendService != "" {
			if _, besRegion, name, perr := parseRegionalResource(fr.BackendService, "backendServices"); perr == nil {
				besName = name
				if bbody, berr := client.Get(callCtx, regionalPath(project, besRegion, "backendServices", besName)); berr == nil {
					var bes backendServiceGetResp
					if json.Unmarshal(bbody, &bes) == nil && len(bes.HealthChecks) > 0 {
						if _, hr, hn, herr := parseRegionalResource(bes.HealthChecks[0], "healthChecks"); herr == nil {
							hcName, hcRegion = hn, hr
						}
					}
				}
			}
		}
	}

	// 1. Forwarding rule (required).
	if err := deleteAndWait(callCtx, client, project, region, "forwardingRules", frName); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete forwarding rule: %v", err))
	}

	// 2. Backend service (required once the forwarding rule is gone).
	if besName != "" {
		if err := deleteAndWait(callCtx, client, project, region, "backendServices", besName); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete backend service %q: %v", besName, err))
		}
	}

	// 3. Health check (best-effort — may be shared with another load balancer).
	var note string
	if hcName != "" {
		if err := deleteAndWait(callCtx, client, project, hcRegion, "healthChecks", hcName); err != nil {
			note = fmt.Sprintf("health check %q was not deleted (it may be in use by another load balancer): %v", hcName, err)
		}
	}

	payload := map[string]any{
		"forwardingRule": frName,
		"backendService": besName,
		"healthCheck":    hcName,
		"region":         region,
	}
	if note != "" {
		payload["note"] = note
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.loadBalancer.deleted",
		[]any{payload},
	)
}

func (d *DeleteLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeleteLoadBalancer) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteLoadBalancer) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
