package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateLoadBalancer struct{}

type CreateLoadBalancerSpec struct {
	Name                string   `mapstructure:"name"`
	Region              string   `mapstructure:"region"`
	Protocol            string   `mapstructure:"protocol"`
	Ports               []string `mapstructure:"ports"`
	InstanceGroup       string   `mapstructure:"instanceGroup"`
	HealthCheckProtocol string   `mapstructure:"healthCheckProtocol"`
	HealthCheckPort     int      `mapstructure:"healthCheckPort"`
	IPAddress           string   `mapstructure:"ipAddress"`
}

func (c *CreateLoadBalancer) Name() string {
	return "gcp.compute.createLoadBalancer"
}

func (c *CreateLoadBalancer) Label() string {
	return "Compute • Create Load Balancer"
}

func (c *CreateLoadBalancer) Description() string {
	return "Create a regional external passthrough Network Load Balancer that forwards TCP/UDP traffic to a group of VM instances"
}

func (c *CreateLoadBalancer) Documentation() string {
	return `The Create Load Balancer component builds a regional external passthrough Network Load Balancer (Layer 4). It gives you one public IP that spreads incoming TCP/UDP traffic across the VMs in an instance group, and routes around unhealthy VMs using a health check.

Google Cloud has no single "load balancer" resource — a load balancer is a collection of resources. This component creates them for you, in order:

1. **Health check** — how the LB decides a VM is healthy
2. **Backend service** — references your instance group and the health check
3. **Forwarding rule** — the public IP + ports that send traffic to the backend service

If any step fails, the resources already created are rolled back.

## Use Cases

- **Distribute traffic**: Spread requests across several VMs running the same service
- **Resilience**: Stop sending traffic to VMs that fail their health check
- **Non-HTTP services**: Balance any TCP/UDP protocol (this is an L4 balancer; it does not terminate HTTPS or route by URL — use an Application Load Balancer for that)

## Configuration

- **Name**: Base name for the load balancer; the pieces are named ` + "`<name>-hc`" + `, ` + "`<name>-backend`" + `, ` + "`<name>-fr`" + ` (required)
- **Region**: The region to create it in (required)
- **Protocol**: ` + "`TCP`" + ` or ` + "`UDP`" + ` (default TCP)
- **Ports**: One or more ports to forward, e.g. 80, 443 (required)
- **Instance group**: The existing instance group whose VMs receive traffic (required)
- **Health check protocol / port**: ` + "`TCP`" + ` (default), ` + "`HTTP`" + `, or ` + "`HTTPS`" + `; port defaults to the first forwarded port
- **Reserved IP**: Optionally use a reserved static IP; leave blank for an ephemeral IP

## Output

Returns the assembled load balancer: **name**, **region**, **ipAddress**, **protocol**, **ports**, and the underlying **forwardingRule**, **backendService**, **healthCheck**, plus **resources** (selfLinks of everything created).

## Important Notes

- Targets an **existing** instance group; the group's VMs must be in the chosen region
- Requires the ` + "`roles/compute.loadBalancerAdmin`" + ` IAM role
- This is an L4 passthrough LB: it forwards ports and preserves the client IP; it does not terminate TLS`
}

func (c *CreateLoadBalancer) Icon() string {
	return "globe"
}

func (c *CreateLoadBalancer) Color() string {
	return "blue"
}

func (c *CreateLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Base name for the load balancer (lowercase letters, numbers and hyphens).",
			Placeholder: "e.g. web-lb",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The region to create the load balancer in.",
			Placeholder: "Select region",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeRegion},
			},
		},
		{
			Name:        "protocol",
			Label:       "Protocol",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "TCP",
			Description: "Transport protocol to forward.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "TCP", Value: "TCP"},
				{Label: "UDP", Value: "UDP"},
			}}},
		},
		{
			Name:        "ports",
			Label:       "Ports",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Ports to forward to the backend VMs (e.g. 80, 443).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Port",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
		},
		{
			Name:        "instanceGroup",
			Label:       "Instance Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The instance group whose VMs receive balanced traffic.",
			Placeholder: "Select instance group",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeInstanceGroup},
			},
		},
		{
			Name:        "healthCheckProtocol",
			Label:       "Health Check Protocol",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "TCP",
			Description: "Protocol the health check uses to probe the VMs.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "TCP", Value: "TCP"},
				{Label: "HTTP", Value: "HTTP"},
				{Label: "HTTPS", Value: "HTTPS"},
			}}},
		},
		{
			Name:        "healthCheckPort",
			Label:       "Health Check Port",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Port the health check probes. Defaults to the first forwarded port.",
		},
		{
			Name:        "ipAddress",
			Label:       "Reserved IP",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Optional reserved static IP. Leave blank for an ephemeral IP.",
			Placeholder: "Select reserved IP",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeStaticIP},
			},
		},
	}
}

func (c *CreateLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := CreateLoadBalancerSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(spec.Region) == "" {
		return errors.New("region is required")
	}
	if strings.TrimSpace(spec.InstanceGroup) == "" {
		return errors.New("instance group is required")
	}
	if _, err := validatePorts(spec.Ports); err != nil {
		return err
	}
	return nil
}

func (c *CreateLoadBalancer) Execute(ctx core.ExecutionContext) error {
	spec := CreateLoadBalancerSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return ctx.ExecutionState.Fail("error", "name is required")
	}
	region := lastSegment(strings.TrimSpace(spec.Region))
	if region == "" {
		return ctx.ExecutionState.Fail("error", "region is required")
	}
	if strings.TrimSpace(spec.InstanceGroup) == "" {
		return ctx.ExecutionState.Fail("error", "instance group is required")
	}
	ports, err := validatePorts(spec.Ports)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	protocol := strings.ToUpper(strings.TrimSpace(spec.Protocol))
	if protocol == "" {
		protocol = "TCP"
	}
	hcPort := spec.HealthCheckPort
	if hcPort == 0 {
		hcPort, _ = strconv.Atoi(ports[0])
	}

	hcName := name + "-hc"
	besName := name + "-backend"
	frName := name + "-fr"
	// The backend service name (suffix "-backend") is the longest derived name,
	// so it is the first to exceed GCP's 63-character resource name limit.
	if len(besName) > 63 {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("name %q is too long; the derived backend service name %q exceeds 63 characters", name, besName))
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}
	project := client.ProjectID()
	callCtx := context.Background()

	// Resolve a reserved static IP (selected as an integration resource, whose
	// value is the address selfLink) to its literal IP up front, so a bad
	// reference fails before any resources are created. Compute Engine expects
	// the literal address on the forwarding rule, not the selfLink. An empty
	// value means an ephemeral IP.
	ipLiteral := ""
	if ipRef := strings.TrimSpace(spec.IPAddress); ipRef != "" {
		ipProject, ipRegion, ipName, perr := parseAddressPath(ipRef)
		if perr != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("invalid reserved IP: %v", perr))
		}
		if ipProject != "" && ipProject != project {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf(
				"reserved IP belongs to project %q but this GCP integration is bound to project %q", ipProject, project))
		}
		if ipRegion != "" && ipRegion != region {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf(
				"reserved IP is in region %q but the load balancer is being created in region %q", ipRegion, region))
		}
		body, gerr := GetAddress(callCtx, client, project, ipRegion, ipName)
		if gerr != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read reserved IP %q: %v", ipName, gerr))
		}
		var addr addressGetResp
		if json.Unmarshal(body, &addr) != nil || strings.TrimSpace(addr.Address) == "" {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("reserved IP %q has no literal address", ipName))
		}
		ipLiteral = addr.Address
	}

	// Track created resources so we can roll back if a later step fails. Each
	// name is recorded the moment its insert is *accepted* by GCP (see the
	// onAccepted callback), not before: an insert that is rejected because a
	// resource of that derived name already exists belongs to another load
	// balancer and must never be rolled back. Recording on acceptance still
	// covers the case where the insert completes but the follow-up read or
	// operation poll fails, since rollback is best-effort and ignores not-found.
	created := &lbResources{client: client, project: project, region: region}
	fail := func(format string, args ...any) error {
		created.rollback(callCtx)
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(format, args...))
	}

	// 1. Health check
	hcSelfLink, err := createAndWait(callCtx, client, project, region, "healthChecks", healthCheckBody(hcName, spec.HealthCheckProtocol, hcPort), func() { created.healthCheck = hcName })
	if err != nil {
		return fail("failed to create health check: %v", err)
	}

	// 2. Backend service
	besBody := map[string]any{
		"name":                besName,
		"loadBalancingScheme": loadBalancingSchemeExternal,
		"protocol":            protocol,
		"healthChecks":        []string{hcSelfLink},
		"backends":            []any{map[string]any{"group": strings.TrimSpace(spec.InstanceGroup)}},
	}
	besSelfLink, err := createAndWait(callCtx, client, project, region, "backendServices", besBody, func() { created.backendService = besName })
	if err != nil {
		return fail("failed to create backend service: %v", err)
	}

	// 3. Forwarding rule
	frBody := map[string]any{
		"name":                frName,
		"loadBalancingScheme": loadBalancingSchemeExternal,
		"IPProtocol":          protocol,
		"ports":               ports,
		"backendService":      besSelfLink,
	}
	if ipLiteral != "" {
		frBody["IPAddress"] = ipLiteral
	}
	frSelfLink, err := createAndWait(callCtx, client, project, region, "forwardingRules", frBody, func() { created.forwardingRule = frName })
	if err != nil {
		return fail("failed to create forwarding rule: %v", err)
	}

	// Determine the load balancer's IP. A reserved IP is already known; for an
	// ephemeral IP, read the forwarding rule back to learn the assigned address.
	// Treat a failed read-back as fatal so we never report success with an empty
	// IP — roll back to a clean state instead.
	assignedIP := ipLiteral
	if assignedIP == "" {
		body, err := client.Get(callCtx, regionalPath(project, region, "forwardingRules", frName))
		if err != nil {
			return fail("forwarding rule %q was created but could not be read back for its IP address: %v", frName, err)
		}
		var fr forwardingRuleGetResp
		if err := json.Unmarshal(body, &fr); err != nil {
			return fail("forwarding rule %q was created but its response could not be parsed: %v", frName, err)
		}
		assignedIP = strings.TrimSpace(fr.IPAddress)
	}
	if assignedIP == "" {
		return fail("forwarding rule %q was created but no IP address was assigned", frName)
	}

	// Emit the forwarding rule as a Delete-consumable reference (its canonical,
	// project-qualified selfLink) rather than a bare name, so the documented
	// create -> delete cleanup flow can chain on data.forwardingRule. Fall back
	// to a relative path if the selfLink is somehow unavailable.
	frRef := strings.TrimSpace(frSelfLink)
	if frRef == "" {
		frRef = fmt.Sprintf("regions/%s/forwardingRules/%s", region, frName)
	}

	payload := map[string]any{
		"name":           name,
		"region":         region,
		"ipAddress":      assignedIP,
		"protocol":       protocol,
		"ports":          ports,
		"forwardingRule": frRef,
		"backendService": besName,
		"healthCheck":    hcName,
		"instanceGroup":  lastSegment(spec.InstanceGroup),
		"resources":      []string{hcSelfLink, besSelfLink, frSelfLink},
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.loadBalancer.created",
		[]any{payload},
	)
}

// lbResources tracks the load balancer pieces created so far so a partial
// failure can be unwound (delete in reverse creation order, best-effort).
type lbResources struct {
	client                                      Client
	project, region                             string
	healthCheck, backendService, forwardingRule string
}

func (r *lbResources) rollback(ctx context.Context) {
	if r.forwardingRule != "" {
		deleteAndWait(ctx, r.client, r.project, r.region, "forwardingRules", r.forwardingRule)
	}
	if r.backendService != "" {
		deleteAndWait(ctx, r.client, r.project, r.region, "backendServices", r.backendService)
	}
	if r.healthCheck != "" {
		deleteAndWait(ctx, r.client, r.project, r.region, "healthChecks", r.healthCheck)
	}
}

// createAndWait POSTs a regional resource, waits for the operation, and returns
// the new resource's selfLink. onAccepted (if non-nil) is invoked once the
// insert has been accepted by GCP — i.e. the POST succeeded and returned an
// operation — so the caller can mark the resource for rollback. It is NOT called
// when the POST itself is rejected (e.g. the resource already exists), so a
// caller never rolls back a resource it did not create.
func createAndWait(ctx context.Context, client Client, project, region, kind string, body map[string]any, onAccepted func()) (string, error) {
	respBody, err := client.Post(ctx, fmt.Sprintf("projects/%s/regions/%s/%s", project, region, kind), body)
	if err != nil {
		return "", err
	}
	var op struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &op); err != nil {
		return "", fmt.Errorf("parse create operation response: %w", err)
	}
	if op.Name == "" {
		return "", fmt.Errorf("create operation response missing operation name")
	}
	if onAccepted != nil {
		onAccepted()
	}
	if err := WaitForRegionOperation(ctx, client, project, region, lastSegment(op.Name)); err != nil {
		return "", err
	}
	body2, err := client.Get(ctx, regionalPath(project, region, kind, body["name"].(string)))
	if err != nil {
		return "", err
	}
	var got struct {
		SelfLink string `json:"selfLink"`
	}
	if err := json.Unmarshal(body2, &got); err != nil {
		return "", fmt.Errorf("parse %s response: %w", kind, err)
	}
	return got.SelfLink, nil
}

// deleteAndWait DELETEs a regional resource and waits for the operation,
// ignoring errors (used for best-effort rollback / teardown of shared pieces).
func deleteAndWait(ctx context.Context, client Client, project, region, kind, name string) error {
	respBody, err := client.Delete(ctx, regionalPath(project, region, kind, name))
	if err != nil {
		return err
	}
	var op struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &op); err != nil {
		return err
	}
	if op.Name == "" {
		return nil
	}
	return WaitForRegionOperation(ctx, client, project, region, lastSegment(op.Name))
}

func regionalPath(project, region, kind, name string) string {
	return fmt.Sprintf("projects/%s/regions/%s/%s/%s", project, region, kind, name)
}

func validatePorts(ports []string) ([]string, error) {
	out := make([]string, 0, len(ports))
	for _, p := range ports {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("invalid port %q: must be a number between 1 and 65535", p)
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, errors.New("at least one port is required")
	}
	return out, nil
}

func (c *CreateLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateLoadBalancer) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateLoadBalancer) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
