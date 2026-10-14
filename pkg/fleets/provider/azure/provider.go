package azureprovider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/superplanehq/superplane/pkg/fleets/provider"
	"golang.org/x/crypto/ssh"
)

const (
	TagKeyFleetManagerID = "superplane_fleet_manager_id"
	TagKeyFleetID        = "superplane_fleet_id"
	TagKeyRunnerID       = "superplane_runner_id"
	TagKeyRunnerVersion  = "superplane_runner_version"
	TagKeyArchitecture   = "superplane_runner_arch"
	tagKeyName           = "Name"
	tagKeyLegacyManaged  = "superplane_managed_runner"

	azureResourceTagLimit    = 50
	reservedResourceTagCount = 6
	maxCustomResourceTags    = azureResourceTagLimit - reservedResourceTagCount
	runnerAdminUsername      = "superplane"
	vmProvisioningDeleting   = "Deleting"
)

type Config struct {
	FleetManagerID         string
	SubscriptionID         string
	ResourceGroup          string
	Location               string
	ImageID                string
	VMSize                 string
	Architecture           string
	SubnetID               string
	NetworkSecurityGroupID string
	IdentityID             string
	Zones                  []string
	DiskSizeGB             int32
	EphemeralOSDisk        bool
	ResourceTags           map[string]string
}

type Provider struct {
	client ComputeAPI
	config Config
	log    *slog.Logger
}

func New(client ComputeAPI, config Config, log *slog.Logger) (*Provider, error) {
	config.FleetManagerID = strings.TrimSpace(config.FleetManagerID)
	config.SubscriptionID = strings.TrimSpace(config.SubscriptionID)
	config.ResourceGroup = strings.TrimSpace(config.ResourceGroup)
	config.Location = strings.TrimSpace(config.Location)
	config.ImageID = strings.TrimSpace(config.ImageID)
	config.VMSize = strings.TrimSpace(config.VMSize)
	config.Architecture = strings.ToLower(strings.TrimSpace(config.Architecture))
	config.SubnetID = strings.TrimSpace(config.SubnetID)
	config.NetworkSecurityGroupID = strings.TrimSpace(config.NetworkSecurityGroupID)
	config.IdentityID = strings.TrimSpace(config.IdentityID)
	config.Zones = nonEmpty(config.Zones)
	switch {
	case config.FleetManagerID == "":
		return nil, fmt.Errorf("Fleet Manager ID is required")
	case config.SubscriptionID == "":
		return nil, fmt.Errorf("Azure subscription ID is required")
	case config.ResourceGroup == "":
		return nil, fmt.Errorf("Azure resource group is required")
	case config.Location == "":
		return nil, fmt.Errorf("Azure location is required")
	case config.ImageID == "":
		return nil, fmt.Errorf("Azure image ID is required")
	case config.VMSize == "":
		return nil, fmt.Errorf("Azure VM size is required")
	case config.Architecture != "amd64" && config.Architecture != "arm64":
		return nil, fmt.Errorf("Azure architecture must be amd64 or arm64")
	case config.SubnetID == "":
		return nil, fmt.Errorf("Azure subnet ID is required")
	case config.NetworkSecurityGroupID == "":
		return nil, fmt.Errorf("Azure network security group ID is required")
	case config.IdentityID == "":
		return nil, fmt.Errorf("Azure identity ID is required")
	case len(config.Zones) == 0:
		return nil, fmt.Errorf("at least one Azure availability zone is required")
	case client == nil:
		return nil, fmt.Errorf("Azure compute client is required")
	}
	if config.DiskSizeGB <= 0 {
		config.DiskSizeGB = 30
	}
	if len(config.ResourceTags) > maxCustomResourceTags {
		return nil, fmt.Errorf(
			"Azure resourceTags must contain at most %d entries; Fleet Manager applies %d reserved tags",
			maxCustomResourceTags,
			reservedResourceTagCount,
		)
	}
	for key := range config.ResourceTags {
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("Azure resource tag key must not be empty")
		}
		if isReservedTag(key) {
			return nil, fmt.Errorf("reserved Azure resource tag %q cannot be overridden", key)
		}
	}
	if log == nil {
		log = slog.Default()
	}
	return &Provider{client: client, config: config, log: log}, nil
}

func (p *Provider) Name() string {
	return "azure"
}

func (p *Provider) List(ctx context.Context, fleetID string) ([]provider.Resource, error) {
	vms, err := p.client.ListVMs(ctx, p.config.ResourceGroup)
	if err != nil {
		return nil, fmt.Errorf("list Azure runner VMs: %w", err)
	}
	var resources []provider.Resource
	vmNames := map[string]struct{}{}
	for _, vm := range vms {
		if tagValue(vm.Tags, TagKeyFleetManagerID) != p.config.FleetManagerID {
			continue
		}
		if tagValue(vm.Tags, TagKeyFleetID) != fleetID {
			continue
		}
		runnerID := tagValue(vm.Tags, TagKeyRunnerID)
		if runnerID == "" || vm.Name == nil {
			continue
		}
		if provisioningState(vm) == vmProvisioningDeleting {
			continue
		}
		resource := provider.Resource{
			ID:       *vm.Name,
			RunnerID: runnerID,
			FleetID:  tagValue(vm.Tags, TagKeyFleetID),
			State:    provisioningState(vm),
		}
		if created := createdAt(vm); !created.IsZero() {
			resource.CreatedAt = created
		}
		vmNames[resource.ID] = struct{}{}
		resources = append(resources, resource)
	}

	nics, err := p.client.ListNICs(ctx, p.config.ResourceGroup)
	if err != nil {
		return nil, fmt.Errorf("list Azure runner NICs: %w", err)
	}
	for _, nic := range nics {
		resource, ok := orphanNICResource(nic, p.config.FleetManagerID, fleetID, vmNames)
		if !ok {
			continue
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func orphanNICResource(
	nic armnetwork.Interface,
	fleetManagerID, fleetID string,
	vmNames map[string]struct{},
) (provider.Resource, bool) {
	if tagValue(nic.Tags, TagKeyFleetManagerID) != fleetManagerID {
		return provider.Resource{}, false
	}
	if tagValue(nic.Tags, TagKeyFleetID) != fleetID {
		return provider.Resource{}, false
	}
	runnerID := tagValue(nic.Tags, TagKeyRunnerID)
	if runnerID == "" || nic.Name == nil {
		return provider.Resource{}, false
	}
	vmName, ok := vmNameFromNIC(*nic.Name)
	if !ok {
		return provider.Resource{}, false
	}
	if _, exists := vmNames[vmName]; exists {
		return provider.Resource{}, false
	}
	return provider.Resource{
		ID:       vmName,
		RunnerID: runnerID,
		FleetID:  fleetID,
	}, true
}

func vmNameFromNIC(name string) (string, bool) {
	const suffix = "-nic"
	if !strings.HasSuffix(name, suffix) || len(name) <= len(suffix) {
		return "", false
	}
	return strings.TrimSuffix(name, suffix), true
}

func (p *Provider) BuildBootstrap(request provider.RunnerBootstrap) ([]byte, error) {
	return buildUserData(request)
}

func (p *Provider) Create(
	ctx context.Context,
	request provider.CreateRequest,
) (provider.Resource, error) {
	if strings.TrimSpace(request.RunnerID) == "" {
		return provider.Resource{}, fmt.Errorf("runner ID is required")
	}
	if strings.TrimSpace(request.FleetID) == "" {
		return provider.Resource{}, fmt.Errorf("fleet ID is required")
	}
	if len(request.Bootstrap) == 0 {
		return provider.Resource{}, fmt.Errorf("runner bootstrap data is required")
	}

	name := virtualMachineName(request.RunnerID)
	tags := p.resourceTags(request)
	nic, err := p.client.CreateNIC(ctx, p.config.ResourceGroup, nicName(name), p.nicParameters(tags))
	if err != nil {
		return provider.Resource{}, fmt.Errorf("create Azure runner NIC: %w", err)
	}
	if nic.ID == nil || strings.TrimSpace(*nic.ID) == "" {
		return provider.Resource{}, fmt.Errorf("Azure NIC create returned no resource ID")
	}

	sshKey, err := throwawaySSHPublicKey()
	if err != nil {
		_ = p.deleteNIC(ctx, name)
		return provider.Resource{}, fmt.Errorf("create Azure runner SSH key: %w", err)
	}

	var lastErr error
	for _, zone := range p.config.Zones {
		vm, err := p.client.CreateVM(
			ctx,
			p.config.ResourceGroup,
			name,
			p.vmParameters(request, name, *nic.ID, zone, sshKey, tags),
		)
		if err == nil {
			resource := provider.Resource{
				ID:        name,
				RunnerID:  request.RunnerID,
				FleetID:   request.FleetID,
				State:     provisioningState(vm),
				CreatedAt: time.Now().UTC(),
			}
			if created := createdAt(vm); !created.IsZero() {
				resource.CreatedAt = created
			}
			p.log.Info(
				"created Azure runner",
				slog.String("runner_id", request.RunnerID),
				slog.String("fleet_id", request.FleetID),
				slog.String("vm_name", resource.ID),
				slog.String("zone", zone),
			)
			return resource, nil
		}
		lastErr = err
		if !isInsufficientCapacity(err) {
			return provider.Resource{}, p.cleanupNIC(
				ctx,
				name,
				fmt.Errorf("create Azure runner: %w", err),
			)
		}
		p.log.Warn(
			"Azure zone has insufficient capacity",
			slog.String("zone", zone),
			slog.String("runner_id", request.RunnerID),
		)
	}
	return provider.Resource{}, p.cleanupNIC(ctx, name, fmt.Errorf(
		"create Azure runner in configured zones: %w",
		lastErr,
	))
}

func (p *Provider) Delete(ctx context.Context, resource provider.Resource) error {
	name := strings.TrimSpace(resource.ID)
	if name == "" {
		return fmt.Errorf("Azure VM name is required")
	}
	err := p.client.DeleteVM(ctx, p.config.ResourceGroup, name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("delete Azure runner %s: %w", name, err)
	}
	if err := p.deleteNIC(ctx, name); err != nil {
		return err
	}
	p.log.Info(
		"deleted Azure runner",
		slog.String("runner_id", resource.RunnerID),
		slog.String("vm_name", name),
	)
	return nil
}

func (p *Provider) cleanupNIC(ctx context.Context, vmName string, cause error) error {
	if err := p.deleteNIC(ctx, vmName); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

func (p *Provider) deleteNIC(ctx context.Context, vmName string) error {
	err := p.client.DeleteNIC(ctx, p.config.ResourceGroup, nicName(vmName))
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("delete Azure runner NIC %s: %w", nicName(vmName), err)
	}
	return nil
}

func (p *Provider) nicParameters(tags map[string]*string) armnetwork.Interface {
	return armnetwork.Interface{
		Location: to.Ptr(p.config.Location),
		Tags:     tags,
		Properties: &armnetwork.InterfacePropertiesFormat{
			NetworkSecurityGroup: &armnetwork.SecurityGroup{
				ID: to.Ptr(p.config.NetworkSecurityGroupID),
			},
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{{
				Name: to.Ptr("ipconfig"),
				Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
					PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
					Subnet: &armnetwork.Subnet{
						ID: to.Ptr(p.config.SubnetID),
					},
				},
			}},
		},
	}
}

func (p *Provider) vmParameters(
	request provider.CreateRequest,
	name, nicID, zone, sshKey string,
	tags map[string]*string,
) armcompute.VirtualMachine {
	osDisk := &armcompute.OSDisk{
		CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesFromImage),
		DeleteOption: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
		DiskSizeGB:   to.Ptr(p.config.DiskSizeGB),
		ManagedDisk: &armcompute.ManagedDiskParameters{
			StorageAccountType: to.Ptr(armcompute.StorageAccountTypesPremiumLRS),
		},
	}
	if p.config.EphemeralOSDisk {
		osDisk.DiffDiskSettings = &armcompute.DiffDiskSettings{
			Option:    to.Ptr(armcompute.DiffDiskOptionsLocal),
			Placement: to.Ptr(armcompute.DiffDiskPlacementCacheDisk),
		}
	}
	return armcompute.VirtualMachine{
		Location: to.Ptr(p.config.Location),
		Zones:    []*string{to.Ptr(zone)},
		Tags:     tags,
		Identity: &armcompute.VirtualMachineIdentity{
			Type: to.Ptr(armcompute.ResourceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armcompute.UserAssignedIdentitiesValue{
				p.config.IdentityID: {},
			},
		},
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: to.Ptr(armcompute.VirtualMachineSizeTypes(p.config.VMSize)),
			},
			StorageProfile: &armcompute.StorageProfile{
				ImageReference: &armcompute.ImageReference{
					ID: to.Ptr(p.config.ImageID),
				},
				OSDisk: osDisk,
			},
			OSProfile: &armcompute.OSProfile{
				ComputerName:  to.Ptr(name),
				AdminUsername: to.Ptr(runnerAdminUsername),
				CustomData:    to.Ptr(base64.StdEncoding.EncodeToString(request.Bootstrap)),
				LinuxConfiguration: &armcompute.LinuxConfiguration{
					DisablePasswordAuthentication: to.Ptr(true),
					SSH: &armcompute.SSHConfiguration{
						PublicKeys: []*armcompute.SSHPublicKey{{
							Path:    to.Ptr("/home/" + runnerAdminUsername + "/.ssh/authorized_keys"),
							KeyData: to.Ptr(sshKey),
						}},
					},
				},
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{{
					ID: to.Ptr(nicID),
					Properties: &armcompute.NetworkInterfaceReferenceProperties{
						Primary:      to.Ptr(true),
						DeleteOption: to.Ptr(armcompute.DeleteOptionsDelete),
					},
				}},
			},
			SecurityProfile: &armcompute.SecurityProfile{
				SecurityType: to.Ptr(armcompute.SecurityTypesTrustedLaunch),
				UefiSettings: &armcompute.UefiSettings{
					SecureBootEnabled: to.Ptr(true),
					VTpmEnabled:       to.Ptr(true),
				},
			},
		},
	}
}

func (p *Provider) resourceTags(request provider.CreateRequest) map[string]*string {
	tags := map[string]*string{
		tagKeyName:           to.Ptr("superplane-runner"),
		TagKeyFleetManagerID: to.Ptr(p.config.FleetManagerID),
		TagKeyFleetID:        to.Ptr(request.FleetID),
		TagKeyRunnerID:       to.Ptr(request.RunnerID),
		TagKeyRunnerVersion:  to.Ptr(request.RunnerVersion),
		TagKeyArchitecture:   to.Ptr(p.config.Architecture),
	}
	keys := make([]string, 0, len(p.config.ResourceTags))
	for key := range p.config.ResourceTags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		tags[key] = to.Ptr(p.config.ResourceTags[key])
	}
	return tags
}

func virtualMachineName(runnerID string) string {
	cleaned := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(runnerID)), "-", "")
	if len(cleaned) > 32 {
		cleaned = cleaned[:32]
	}
	return "sprunner-" + cleaned
}

func nicName(vmName string) string {
	return vmName + "-nic"
}

func tagValue(tags map[string]*string, key string) string {
	if tags == nil {
		return ""
	}
	value, ok := tags[key]
	if !ok || value == nil {
		return ""
	}
	return *value
}

func provisioningState(vm armcompute.VirtualMachine) string {
	if vm.Properties == nil || vm.Properties.ProvisioningState == nil {
		return ""
	}
	return *vm.Properties.ProvisioningState
}

func createdAt(vm armcompute.VirtualMachine) time.Time {
	if vm.Properties == nil || vm.Properties.TimeCreated == nil {
		return time.Time{}
	}
	return vm.Properties.TimeCreated.UTC()
}

func isInsufficientCapacity(err error) bool {
	var responseErr *azcore.ResponseError
	if !errors.As(err, &responseErr) {
		return false
	}
	switch responseErr.ErrorCode {
	case "AllocationFailed", "ZonalAllocationFailed", "SkuNotAvailable":
		return true
	default:
		return false
	}
}

func isNotFound(err error) bool {
	var responseErr *azcore.ResponseError
	return errors.As(err, &responseErr) && responseErr.StatusCode == 404
}

func isReservedTag(key string) bool {
	switch key {
	case tagKeyName,
		tagKeyLegacyManaged,
		TagKeyFleetManagerID,
		TagKeyFleetID,
		TagKeyRunnerID,
		TagKeyRunnerVersion,
		TagKeyArchitecture:
		return true
	default:
		return false
	}
}

func nonEmpty(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func throwawaySSHPublicKey() (string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	public, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		return "", err
	}
	return string(ssh.MarshalAuthorizedKey(public)), nil
}
