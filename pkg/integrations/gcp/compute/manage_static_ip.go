package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	staticIPActionAttach    = "attach"
	staticIPActionDetach    = "detach"
	defaultNetworkInterface = "nic0"
	defaultAccessConfigName = "External NAT"
	accessConfigOneToOneNAT = "ONE_TO_ONE_NAT"
)

type ManageStaticIP struct{}

type ManageStaticIPSpec struct {
	Action           string `mapstructure:"action"`
	Instance         string `mapstructure:"instance"`
	Address          string `mapstructure:"address"`
	NetworkInterface string `mapstructure:"networkInterface"`
}

// staticIPNetworkInterface is the subset of an instance's network interface we
// need to attach or detach an external access config (the static IP).
type staticIPAccessConfig struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	NatIP string `json:"natIP"`
}

type staticIPNetworkInterface struct {
	Name          string                 `json:"name"`
	NetworkIP     string                 `json:"networkIP"`
	AccessConfigs []staticIPAccessConfig `json:"accessConfigs"`
}

type staticIPInstanceResp struct {
	NetworkInterfaces []staticIPNetworkInterface `json:"networkInterfaces"`
}

func (m *ManageStaticIP) Name() string {
	return "gcp.compute.manageStaticIP"
}

func (m *ManageStaticIP) Label() string {
	return "Compute • Manage Static IP"
}

func (m *ManageStaticIP) Description() string {
	return "Attach or detach a static IP address to/from a Google Compute Engine VM instance"
}

func (m *ManageStaticIP) Documentation() string {
	return `The Manage Static IP component attaches a reserved external static IP to a VM instance, or detaches the instance's current external IP.

Attaching works by managing the network interface's external access config: any existing external IP on the interface is removed first, then the static IP is assigned. Detaching removes the external access config entirely (the instance keeps its internal IP but loses external connectivity unless another address is attached).

## Use Cases

- **Blue/green deployments**: Move a stable public IP from the old VM to the new one with zero DNS changes
- **Failover**: Reassign a reserved IP from a failed VM to a healthy replacement
- **Maintenance**: Temporarily detach a public IP while a VM is serviced

## Configuration

- **Action**: ` + "`attach`" + ` or ` + "`detach`" + ` (required)
- **VM Instance**: The target VM. The selection encodes both the zone and the instance name.
- **Static IP** *(attach only)*: The reserved external IP to attach. Only IPs in the selected VM's region are listed, since a regional IP can attach only to a VM in the same region.
- **Network Interface**: The interface to modify (default ` + "`nic0`" + `)

## Output

Returns the instance state after the operation:
- **instanceId**, **name**, **zone**, **status**, **selfLink**, **machineType**, **internalIP**, **externalIP**
- **action**: The action performed (attach or detach)

## Important Notes

- A regional static IP can only be attached to a VM in the **same region**
- Attaching is idempotent: if the static IP is already the instance's external IP, the component succeeds without changes
- Detaching is idempotent: if the interface already has no external IP, the component succeeds without changes
- The component waits for each underlying zone operation to complete before emitting`
}

func (m *ManageStaticIP) Icon() string {
	return "globe"
}

func (m *ManageStaticIP) Color() string {
	return "orange"
}

func (m *ManageStaticIP) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (m *ManageStaticIP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "action",
			Label:       "Action",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Whether to attach a static IP or detach the current external IP.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Attach", Value: staticIPActionAttach},
						{Label: "Detach", Value: staticIPActionDetach},
					},
				},
			},
		},
		{
			Name:        "instance",
			Label:       "VM Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The VM instance to attach the static IP to, or detach it from.",
			Placeholder: "Select instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
		{
			Name:        "address",
			Label:       "Static IP",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The reserved external IP to attach. Lists reserved IPs in the selected VM's region (a regional IP can only attach to a VM in the same region).",
			Placeholder: "Select static IP",
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "action", Values: []string{staticIPActionAttach}},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "action", Values: []string{staticIPActionAttach}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeStaticIP,
					Parameters: []configuration.ParameterRef{
						{Name: "instance", ValueFrom: &configuration.ParameterValueFrom{Field: "instance"}},
					},
				},
			},
		},
		{
			Name:        "networkInterface",
			Label:       "Network Interface",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The network interface to modify. Defaults to nic0.",
			Default:     defaultNetworkInterface,
			Placeholder: defaultNetworkInterface,
		},
	}
}

func (m *ManageStaticIP) Setup(ctx core.SetupContext) error {
	spec := ManageStaticIPSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Action == "" {
		return errors.New("action is required")
	}
	if spec.Action != staticIPActionAttach && spec.Action != staticIPActionDetach {
		return fmt.Errorf("invalid action %q: must be %s or %s", spec.Action, staticIPActionAttach, staticIPActionDetach)
	}

	if strings.TrimSpace(spec.Instance) == "" {
		return errors.New("instance is required")
	}

	if spec.Action == staticIPActionAttach && strings.TrimSpace(spec.Address) == "" {
		return errors.New("address is required when action is attach")
	}

	return resolveInstanceNodeMetadata(ctx, spec.Instance)
}

func (m *ManageStaticIP) Execute(ctx core.ExecutionContext) error {
	spec := ManageStaticIPSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	if spec.Action != staticIPActionAttach && spec.Action != staticIPActionDetach {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("invalid action %q", spec.Action))
	}

	urlProject, zone, instanceName, err := parseInstancePath(spec.Instance)
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
			"instance belongs to project %q but this GCP integration is bound to project %q; cross-project operations are not supported",
			urlProject, project,
		))
	}

	networkInterface := strings.TrimSpace(spec.NetworkInterface)
	if networkInterface == "" {
		networkInterface = defaultNetworkInterface
	}

	callCtx := context.Background()
	nics, err := getInstanceNetworkInterfaces(callCtx, client, project, zone, instanceName)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read instance: %v", err))
	}

	nic, ok := findNetworkInterface(nics, networkInterface)
	if !ok {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("network interface %q not found on instance %q", networkInterface, instanceName))
	}

	if spec.Action == staticIPActionAttach {
		if err := m.attach(callCtx, ctx, client, project, zone, instanceName, networkInterface, nic, spec.Address); err != nil {
			return err
		}
	} else {
		if err := m.detach(callCtx, ctx, client, project, zone, instanceName, networkInterface, nic); err != nil {
			return err
		}
	}

	// attach/detach report validation/API failures via ExecutionState.Fail, which
	// finishes the execution but returns a nil error; bail out before emitting a
	// success payload on top of a failed state.
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	body, err := GetInstance(callCtx, client, project, zone, instanceName)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read instance after %s: %v", spec.Action, err))
	}

	payload, err := InstancePayloadFromGetResponse(body, zone)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse instance: %v", err))
	}
	payload["action"] = spec.Action

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		fmt.Sprintf("gcp.compute.staticIP.%sed", spec.Action),
		[]any{payload},
	)
}

// attach assigns a reserved external IP to the instance's network interface,
// replacing any existing external IP. It is a no-op when the static IP is
// already the interface's external address.
func (m *ManageStaticIP) attach(callCtx context.Context, ctx core.ExecutionContext, client Client, project, zone, instanceName, networkInterface string, nic staticIPNetworkInterface, address string) error {
	addrProject, addrRegion, addrName, err := parseAddressPath(address)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if addrProject != "" && addrProject != project {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(
			"static IP belongs to project %q but this GCP integration is bound to project %q; cross-project operations are not supported",
			addrProject, project,
		))
	}

	instanceRegion := deriveRegionFromZone(zone)
	if instanceRegion != "" && addrRegion != instanceRegion {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(
			"static IP is in region %q but the instance is in region %q; a regional static IP can only attach to a VM in the same region",
			addrRegion, instanceRegion,
		))
	}

	addrBody, err := GetAddress(callCtx, client, project, addrRegion, addrName)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read static IP: %v", err))
	}
	var addr addressGetResp
	if err := json.Unmarshal(addrBody, &addr); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse static IP response: %v", err))
	}
	natIP := strings.TrimSpace(addr.Address)
	if natIP == "" {
		return ctx.ExecutionState.Fail("error", "static IP has no address value; cannot attach")
	}

	// Already attached -> nothing to do.
	if len(nic.AccessConfigs) > 0 && nic.AccessConfigs[0].NatIP == natIP {
		return nil
	}

	accessConfigName := defaultAccessConfigName
	if len(nic.AccessConfigs) > 0 {
		if nic.AccessConfigs[0].Name != "" {
			accessConfigName = nic.AccessConfigs[0].Name
		}
		// An interface can hold a single external access config, so remove the
		// existing one before assigning the static IP.
		if err := deleteInstanceAccessConfig(callCtx, client, project, zone, instanceName, networkInterface, accessConfigName); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to remove existing external IP: %v", err))
		}
	}

	if err := addInstanceAccessConfig(callCtx, client, project, zone, instanceName, networkInterface, accessConfigName, natIP); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to attach static IP: %v", err))
	}
	return nil
}

// detach removes the external access config (and its IP) from the instance's
// network interface. It is a no-op when the interface has no external IP.
func (m *ManageStaticIP) detach(callCtx context.Context, ctx core.ExecutionContext, client Client, project, zone, instanceName, networkInterface string, nic staticIPNetworkInterface) error {
	if len(nic.AccessConfigs) == 0 {
		return nil
	}
	accessConfigName := nic.AccessConfigs[0].Name
	if accessConfigName == "" {
		accessConfigName = defaultAccessConfigName
	}
	if err := deleteInstanceAccessConfig(callCtx, client, project, zone, instanceName, networkInterface, accessConfigName); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to detach external IP: %v", err))
	}
	return nil
}

func getInstanceNetworkInterfaces(ctx context.Context, client Client, project, zone, instance string) ([]staticIPNetworkInterface, error) {
	body, err := GetInstance(ctx, client, project, zone, instance)
	if err != nil {
		return nil, err
	}
	var resp staticIPInstanceResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse instance response: %w", err)
	}
	return resp.NetworkInterfaces, nil
}

func findNetworkInterface(nics []staticIPNetworkInterface, name string) (staticIPNetworkInterface, bool) {
	for _, ni := range nics {
		if ni.Name == name {
			return ni, true
		}
	}
	return staticIPNetworkInterface{}, false
}

func addInstanceAccessConfig(ctx context.Context, client Client, project, zone, instance, networkInterface, accessConfigName, natIP string) error {
	query := url.Values{"networkInterface": {networkInterface}}
	path := fmt.Sprintf("projects/%s/zones/%s/instances/%s/addAccessConfig?%s", project, zone, instance, query.Encode())
	body := map[string]any{
		"type":  accessConfigOneToOneNAT,
		"name":  accessConfigName,
		"natIP": natIP,
	}
	return runInstanceAccessConfigOperation(ctx, client, project, zone, path, body)
}

func deleteInstanceAccessConfig(ctx context.Context, client Client, project, zone, instance, networkInterface, accessConfigName string) error {
	query := url.Values{"networkInterface": {networkInterface}, "accessConfig": {accessConfigName}}
	path := fmt.Sprintf("projects/%s/zones/%s/instances/%s/deleteAccessConfig?%s", project, zone, instance, query.Encode())
	return runInstanceAccessConfigOperation(ctx, client, project, zone, path, nil)
}

func runInstanceAccessConfigOperation(ctx context.Context, client Client, project, zone, path string, body any) error {
	respBody, err := client.Post(ctx, path, body)
	if err != nil {
		return err
	}
	var opResp struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &opResp); err != nil {
		return fmt.Errorf("parse operation response: %w", err)
	}
	if opResp.Name == "" {
		return errors.New("operation response missing operation name")
	}
	return WaitForZoneOperation(ctx, client, project, zone, lastSegment(opResp.Name))
}

func (m *ManageStaticIP) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (m *ManageStaticIP) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (m *ManageStaticIP) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (m *ManageStaticIP) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (m *ManageStaticIP) Hooks() []core.Hook {
	return []core.Hook{}
}

func (m *ManageStaticIP) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
