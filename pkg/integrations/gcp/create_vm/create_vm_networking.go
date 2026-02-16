package createvm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	compute "google.golang.org/api/compute/v1"
)

const (
	ResourceTypeNetwork    = "network"
	ResourceTypeSubnetwork = "subnetwork"
	ResourceTypeAddress    = "address"
	ResourceTypeFirewall   = "firewall"
)

const (
	NICTypeGVNIC     = "GVNIC"
	NICTypeVirtioNet = "VIRTIO_NET"
)

const (
	StackTypeIPv4Only  = "IPV4_ONLY"
	StackTypeDualStack = "IPV4_IPV6"
)

const (
	ExternalIPNone      = "none"
	ExternalIPEphemeral = "ephemeral"
	ExternalIPStatic    = "static"
)

const (
	InternalIPEphemeral = "ephemeral"
	InternalIPStatic    = "static"
)

const AddressTypeExternal = "EXTERNAL"

type Network struct {
	Name     string `json:"name"`
	SelfLink string `json:"selfLink"`
}

type Subnetwork struct {
	Name     string `json:"name"`
	Region   string `json:"region"`
	SelfLink string `json:"selfLink"`
}

type Address struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
	Region      string `json:"region"`
	SelfLink    string `json:"selfLink"`
	Status      string `json:"status"`
	AddressType string `json:"addressType"`
}

type Firewall struct {
	Name     string `json:"name"`
	SelfLink string `json:"selfLink"`
	Network  string `json:"network,omitempty"`
}

type networksListResp struct {
	Items         []*networkItem `json:"items"`
	NextPageToken string         `json:"nextPageToken"`
}

type networkItem struct {
	Name     string `json:"name"`
	SelfLink string `json:"selfLink"`
}

type subnetworksListResp struct {
	Items         []*subnetworkItem `json:"items"`
	NextPageToken string            `json:"nextPageToken"`
}

type subnetworkItem struct {
	Name     string `json:"name"`
	Region   string `json:"region"`
	SelfLink string `json:"selfLink"`
}

type addressesListResp struct {
	Items         []*addressItem `json:"items"`
	NextPageToken string         `json:"nextPageToken"`
}

type addressItem struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
	Region      string `json:"region"`
	SelfLink    string `json:"selfLink"`
	Status      string `json:"status"`
	AddressType string `json:"addressType"`
}

type firewallsListResp struct {
	Items         []*firewallItem `json:"items"`
	NextPageToken string          `json:"nextPageToken"`
}

type firewallItem struct {
	Name     string `json:"name"`
	SelfLink string `json:"selfLink"`
	Network  string `json:"network"`
}

func ensureProject(project string, c Client) string {
	if project == "" {
		return c.ProjectID()
	}
	return project
}

func ensureNonEmptyRegion(region, errMsg string) (string, error) {
	r := strings.TrimSpace(region)
	if r == "" {
		return "", fmt.Errorf("%s", errMsg)
	}
	return r, nil
}

func defaultRegion(itemRegion, fallback string) string {
	if strings.TrimSpace(itemRegion) != "" {
		return itemRegion
	}
	return fallback
}

func CreateVMNetworkingConfigFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "network",
			Label:       "VPC network",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "VPC network for the VM. Leave empty to use the default network.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeNetwork,
				},
			},
		},
		{
			Name:        "subnetwork",
			Label:       "Subnet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Subnetwork in the selected region. Leave empty to use the default subnet in the network.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeSubnetwork,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
					},
				},
			},
		},
		{
			Name:        "nicType",
			Label:       "NIC type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Virtual NIC type. GVNIC is recommended for newer images and higher throughput.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "VIRTIO_NET (default)", Value: NICTypeVirtioNet},
						{Label: "GVNIC", Value: NICTypeGVNIC},
					},
				},
			},
		},
		{
			Name:        "internalIPType",
			Label:       "Internal IP",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Use an ephemeral internal IP (assigned by GCP) or a reserved static internal IP.",
			Default:     InternalIPEphemeral,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Ephemeral", Value: InternalIPEphemeral},
						{Label: "Static (reserved)", Value: InternalIPStatic},
					},
				},
			},
		},
		{
			Name:        "internalIPAddress",
			Label:       "Reserved internal IP",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Reserved internal IP address or its full URL. Used when Internal IP is Static.",
			Placeholder: "e.g. 10.0.0.5 or full address URL",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "internalIPType", Values: []string{InternalIPStatic}},
			},
		},
		{
			Name:        "externalIPType",
			Label:       "External IP",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "No external IP, ephemeral (temporary), or a reserved static external IP.",
			Default:     ExternalIPEphemeral,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None", Value: ExternalIPNone},
						{Label: "Ephemeral", Value: ExternalIPEphemeral},
						{Label: "Static (reserved)", Value: ExternalIPStatic},
					},
				},
			},
		},
		{
			Name:        "externalIPAddress",
			Label:       "Reserved external IP",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select a reserved external IP address in the same region as the VM.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeAddress,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "externalIPType", Values: []string{ExternalIPStatic}},
			},
		},
		{
			Name:        "networkTags",
			Label:       "Network tags",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Comma-separated tags for firewall rules and identification (e.g. allow-ssh).",
			Placeholder: "e.g. http-server, allow-ssh",
		},
		{
			Name:        "stackType",
			Label:       "IP stack type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "IPv4 only or dual stack (IPv4 and IPv6). Dual stack requires a dual-stack subnet.",
			Default:     StackTypeIPv4Only,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "IPv4 only", Value: StackTypeIPv4Only},
						{Label: "IPv4 and IPv6 (dual stack)", Value: StackTypeDualStack},
					},
				},
			},
		},
		{
			Name:        "firewallRules",
			Label:       "Firewall rules",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Select firewall rules that apply to this instance.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Firewall rule",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "id",
								Label:       "Firewall rule",
								Type:        configuration.FieldTypeIntegrationResource,
								Required:    false,
								Description: "Select a firewall rule.",
								TypeOptions: &configuration.TypeOptions{
									Resource: &configuration.ResourceTypeOptions{
										Type: ResourceTypeFirewall,
										Parameters: []configuration.ParameterRef{
											{Name: "project", ValueFrom: &configuration.ParameterValueFrom{Field: "project"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func ListNetworks(ctx context.Context, c Client, project string) ([]Network, error) {
	project = ensureProject(project, c)
	path := fmt.Sprintf("projects/%s/global/networks", project)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp networksListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse networks list: %w", err)
	}
	out := make([]Network, 0, len(resp.Items))
	for _, n := range resp.Items {
		if n == nil {
			continue
		}
		out = append(out, Network{Name: n.Name, SelfLink: n.SelfLink})
	}
	return out, nil
}

func ListSubnetworks(ctx context.Context, c Client, project, region string) ([]Subnetwork, error) {
	project = ensureProject(project, c)
	region, err := ensureNonEmptyRegion(region, "region is required for listing subnetworks")
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("projects/%s/regions/%s/subnetworks", project, region)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp subnetworksListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse subnetworks list: %w", err)
	}
	out := make([]Subnetwork, 0, len(resp.Items))
	for _, s := range resp.Items {
		if s == nil {
			continue
		}
		out = append(out, Subnetwork{
			Name:     s.Name,
			Region:   defaultRegion(s.Region, region),
			SelfLink: s.SelfLink,
		})
	}
	return out, nil
}

func ListAddresses(ctx context.Context, c Client, project, region string) ([]Address, error) {
	project = ensureProject(project, c)
	region, err := ensureNonEmptyRegion(region, "region is required for listing addresses")
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("projects/%s/regions/%s/addresses", project, region)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp addressesListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse addresses list: %w", err)
	}
	out := make([]Address, 0, len(resp.Items))
	for _, a := range resp.Items {
		if a == nil {
			continue
		}
		out = append(out, Address{
			Name:        a.Name,
			Address:     a.Address,
			Region:      defaultRegion(a.Region, region),
			SelfLink:    a.SelfLink,
			Status:      a.Status,
			AddressType: a.AddressType,
		})
	}
	return out, nil
}

func ListFirewalls(ctx context.Context, c Client, project string) ([]Firewall, error) {
	project = ensureProject(project, c)
	path := fmt.Sprintf("projects/%s/global/firewalls", project)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp firewallsListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse firewalls list: %w", err)
	}
	out := make([]Firewall, 0, len(resp.Items))
	for _, f := range resp.Items {
		if f == nil {
			continue
		}
		out = append(out, Firewall{Name: f.Name, SelfLink: f.SelfLink, Network: f.Network})
	}
	return out, nil
}

func ListNetworkResources(ctx context.Context, c Client, project string) ([]core.IntegrationResource, error) {
	list, err := ListNetworks(ctx, c, project)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, n := range list {
		out = append(out, core.IntegrationResource{Type: ResourceTypeNetwork, Name: n.Name, ID: n.SelfLink})
	}
	return out, nil
}

func ListSubnetworkResources(ctx context.Context, c Client, project, region string) ([]core.IntegrationResource, error) {
	list, err := ListSubnetworks(ctx, c, project, region)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, s := range list {
		label := s.Name
		if s.Region != "" {
			label = fmt.Sprintf("%s (%s)", s.Name, s.Region)
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeSubnetwork, Name: label, ID: s.SelfLink})
	}
	return out, nil
}

func ListAddressResources(ctx context.Context, c Client, project, region string) ([]core.IntegrationResource, error) {
	list, err := ListAddresses(ctx, c, project, region)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, a := range list {
		if a.AddressType != AddressTypeExternal {
			continue
		}
		label := a.Name
		if a.Address != "" {
			label = fmt.Sprintf("%s (%s)", a.Name, a.Address)
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeAddress, Name: label, ID: a.SelfLink})
	}
	return out, nil
}

func ListFirewallResources(ctx context.Context, c Client, project string) ([]core.IntegrationResource, error) {
	list, err := ListFirewalls(ctx, c, project)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, f := range list {
		label := f.Name
		if f.Network != "" {
			label = fmt.Sprintf("%s (%s)", f.Name, lastSegment(f.Network))
		}
		id := f.SelfLink
		if id == "" {
			id = f.Name
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeFirewall, Name: label, ID: id})
	}
	return out, nil
}

type NetworkingConfig struct {
	Network           string              `mapstructure:"network"`
	Subnetwork        string              `mapstructure:"subnetwork"`
	NicType           string              `mapstructure:"nicType"`
	InternalIPType    string              `mapstructure:"internalIPType"`
	InternalIPAddress string              `mapstructure:"internalIPAddress"`
	ExternalIPType    string              `mapstructure:"externalIPType"`
	ExternalIPAddress string              `mapstructure:"externalIPAddress"`
	NetworkTags       string              `mapstructure:"networkTags"`
	StackType         string              `mapstructure:"stackType"`
	FirewallRules     []FirewallRuleEntry `mapstructure:"firewallRules"`
}

type FirewallRuleEntry struct {
	ID string `mapstructure:"id"`
}

func ParseNetworkTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func BuildNetworkInterfaces(project, region string, config NetworkingConfig) []*compute.NetworkInterface {
	network := strings.TrimSpace(config.Network)
	subnetwork := strings.TrimSpace(config.Subnetwork)
	if network == "" && subnetwork == "" {
		network = "default"
	}
	ni := &compute.NetworkInterface{
		Network:    resolveNetworkURL(project, network),
		Subnetwork: resolveSubnetworkURL(project, region, subnetwork),
	}
	if config.NicType != "" {
		ni.NicType = config.NicType
	}
	if config.StackType != "" {
		ni.StackType = config.StackType
	}
	if config.InternalIPType == InternalIPStatic && strings.TrimSpace(config.InternalIPAddress) != "" {
		ni.NetworkIP = strings.TrimSpace(config.InternalIPAddress)
	}
	externalType := strings.TrimSpace(config.ExternalIPType)
	if externalType == "" {
		externalType = ExternalIPEphemeral
	}
	if externalType != ExternalIPNone {
		ac := &compute.AccessConfig{Type: "ONE_TO_ONE_NAT"}
		if externalType == ExternalIPStatic && strings.TrimSpace(config.ExternalIPAddress) != "" {
			ac.NatIP = strings.TrimSpace(config.ExternalIPAddress)
		}
		ni.AccessConfigs = []*compute.AccessConfig{ac}
	}
	return []*compute.NetworkInterface{ni}
}

func resolveNetworkURL(project, network string) string {
	if strings.Contains(network, "/") {
		return network
	}
	if project == "" || network == "" {
		return network
	}
	return fmt.Sprintf("projects/%s/global/networks/%s", project, network)
}

func resolveSubnetworkURL(project, region, subnetwork string) string {
	if strings.TrimSpace(subnetwork) == "" {
		return ""
	}
	if strings.Contains(subnetwork, "/") {
		return subnetwork
	}
	if project == "" || region == "" {
		return subnetwork
	}
	return fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", project, region, subnetwork)
}
