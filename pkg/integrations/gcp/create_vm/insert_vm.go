package createvm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	compute "google.golang.org/api/compute/v1"
)

const (
	defaultOperationWaitTimeout = 10 * time.Minute
	operationPollInterval       = 3 * time.Second
	defaultOAuthScope           = "https://www.googleapis.com/auth/cloud-platform"
)

const (
	opStatusDone    = "DONE"
	opStatusPending = "PENDING"
	opStatusRunning = "RUNNING"
)

func bootDiskConfigFromOSConfig(project, zone string, c OSAndStorageConfig) BootDiskConfig {
	cfg := BootDiskConfig{
		DiskType:          strings.TrimSpace(c.BootDiskType),
		SizeGb:            c.BootDiskSizeGb,
		SnapshotSchedule:  strings.TrimSpace(c.BootDiskSnapshotSchedule),
		AutoDelete:        c.BootDiskAutoDelete,
		DiskEncryptionKey: strings.TrimSpace(c.BootDiskEncryptionKey),
	}
	if cfg.DiskType == "" {
		cfg.DiskType = DefaultDiskType
	}

	sourceType := strings.TrimSpace(c.BootDiskSourceType)
	switch sourceType {
	case BootDiskSourceExistingDisk:
		s := strings.TrimSpace(c.BootDiskExistingDisk)
		if s != "" {
			cfg.SourceDisk = resolveDiskURL(project, zone, s)
		}
	case BootDiskSourceSnapshot:
		s := strings.TrimSpace(c.BootDiskSnapshot)
		if s != "" {
			cfg.SourceSnapshot = resolveSnapshotURL(project, s)
		}
	case BootDiskSourceCustomImage:
		s := strings.TrimSpace(c.BootDiskCustomImage)
		if s != "" {
			cfg.SourceImage = resolveImageURL(project, s)
		}
	case BootDiskSourcePublicImage:
		fallthrough
	default:
		s := strings.TrimSpace(c.BootDiskPublicImage)
		if s != "" {
			cfg.SourceImage = resolveImageURL(project, s)
		}
	}
	return cfg
}

func additionalDisksFromOSConfig(c OSAndStorageConfig) []AdditionalDisk {
	if len(c.AdditionalDisks) == 0 {
		return nil
	}
	out := make([]AdditionalDisk, 0, len(c.AdditionalDisks))
	for _, e := range c.AdditionalDisks {
		if e.Mode == AdditionalDiskModeExisting && strings.TrimSpace(e.ExistingDisk) != "" {
			out = append(out, AdditionalDisk{
				SourceDisk: e.ExistingDisk,
				AutoDelete: e.AutoDelete,
			})
			continue
		}
		out = append(out, AdditionalDisk{
			Name:       strings.TrimSpace(e.Name),
			SizeGb:     e.SizeGb,
			DiskType:   strings.TrimSpace(e.DiskType),
			AutoDelete: e.AutoDelete,
		})
		if out[len(out)-1].DiskType == "" {
			out[len(out)-1].DiskType = DefaultDiskType
		}
	}
	return out
}

func managementConfigFromCreateVMConfig(c CreateVMConfig) ManagementConfig {
	ar := &c.AutomaticRestart
	return ManagementConfig{
		MetadataItems:     c.MetadataItems,
		StartupScript:     c.StartupScript,
		ShutdownScript:    c.ShutdownScript,
		AutomaticRestart:  ar,
		OnHostMaintenance: c.OnHostMaintenance,
		MaintenancePolicy: c.MaintenancePolicy,
	}
}

func advancedConfigFromCreateVMConfig(c CreateVMConfig) AdvancedConfig {
	return AdvancedConfig{
		GuestAccelerators:      c.GuestAccelerators,
		NodeAffinities:         c.NodeAffinities,
		ResourcePolicies:       c.ResourcePolicies,
		MinNodeCpus:            c.MinNodeCpus,
		Labels:                 c.Labels,
		EnableDisplayDevice:    c.EnableDisplayDevice,
		EnableSerialPortAccess: c.EnableSerialPortAccess,
	}
}

func resolveImageURL(project, imageRef string) string {
	if strings.Contains(imageRef, "/") {
		return imageRef
	}
	if project == "" {
		return imageRef
	}
	return fmt.Sprintf("projects/%s/global/images/%s", project, imageRef)
}

func resolveSnapshotURL(project, snapshotRef string) string {
	if strings.Contains(snapshotRef, "/") {
		return snapshotRef
	}
	if project == "" {
		return snapshotRef
	}
	return fmt.Sprintf("projects/%s/global/snapshots/%s", project, snapshotRef)
}

func resolveDiskURL(project, zone, diskRef string) string {
	if strings.Contains(diskRef, "/") {
		return diskRef
	}
	if project == "" || zone == "" {
		return diskRef
	}
	return fmt.Sprintf("projects/%s/zones/%s/disks/%s", project, zone, diskRef)
}

func deriveRegionFromZone(zone string) string {
	if !strings.Contains(zone, "-") {
		return ""
	}
	parts := strings.Split(zone, "-")
	if len(parts) < 2 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], "-")
}

func buildDisks(project, zone string, config CreateVMConfig) ([]*compute.AttachedDisk, error) {
	bootCfg := bootDiskConfigFromOSConfig(project, zone, config.OSAndStorageConfig)
	bootDisk := BuildBootDisk(project, zone, bootCfg)
	if bootDisk == nil {
		return nil, fmt.Errorf("boot disk could not be built: ensure a boot disk source (image, snapshot, or existing disk) is set")
	}
	disks := []*compute.AttachedDisk{bootDisk}

	additional := additionalDisksFromOSConfig(config.OSAndStorageConfig)
	for i := range additional {
		if additional[i].SourceDisk != "" && !strings.Contains(additional[i].SourceDisk, "/") {
			additional[i].SourceDisk = resolveDiskURL(project, zone, additional[i].SourceDisk)
		}
	}
	disks = append(disks, BuildAdditionalDisks(project, zone, additional)...)
	if config.LocalSSDCount > 0 {
		disks = append(disks, BuildLocalSSDDisks(project, zone, int(config.LocalSSDCount))...)
	}
	return disks, nil
}

func buildSchedulingAndResourcePolicies(zone string, config CreateVMConfig) (*compute.Scheduling, []string) {
	mgmt := managementConfigFromCreateVMConfig(config)
	scheduling := BuildScheduling(mgmt)
	adv := advancedConfigFromCreateVMConfig(config)
	ApplyAdvancedScheduling(scheduling, adv)

	resourcePolicies := BuildInstanceResourcePolicies(adv)
	if strings.TrimSpace(config.MaintenancePolicy) != "" {
		resourcePolicies = append([]string{strings.TrimSpace(config.MaintenancePolicy)}, resourcePolicies...)
	}

	provisioningModel := strings.TrimSpace(config.ProvisioningModel)
	if provisioningModel == "" {
		provisioningModel = string(ProvisioningStandard)
	}
	if provisioningModel == string(ProvisioningSpot) {
		scheduling.Preemptible = true
		scheduling.ProvisioningModel = string(ProvisioningSpot)
		scheduling.OnHostMaintenance = OnHostMaintenanceTerminate
		automaticRestart := false
		scheduling.AutomaticRestart = &automaticRestart
		return scheduling, resourcePolicies
	}
	scheduling.ProvisioningModel = string(ProvisioningStandard)
	return scheduling, resourcePolicies
}

func buildInstanceMetadataFromConfig(mgmt ManagementConfig, config CreateVMConfig) *compute.Metadata {
	metadata := BuildInstanceMetadata(mgmt)
	if config.BlockProjectSSHKeys {
		metadata = ensureMetadataItem(metadata, "block-project-ssh-keys", "true")
	}
	if config.EnableOSLogin {
		metadata = ensureMetadataItem(metadata, "enable-oslogin", "true")
	}
	if config.EnableSerialPortAccess {
		metadata = ensureMetadataItem(metadata, "serial-port-enable", "true")
	}
	return metadata
}

func ensureMetadataItem(m *compute.Metadata, key, value string) *compute.Metadata {
	if m == nil {
		return &compute.Metadata{Items: []*compute.MetadataItems{{Key: key, Value: &value}}}
	}
	for _, it := range m.Items {
		if it != nil && it.Key == key {
			it.Value = &value
			return m
		}
	}
	m.Items = append(m.Items, &compute.MetadataItems{Key: key, Value: &value})
	return m
}

func BuildInstanceFromConfig(project, zone, region string, config CreateVMConfig) (*compute.Instance, error) {
	name := strings.TrimSpace(config.InstanceName)
	if name == "" {
		return nil, fmt.Errorf("instance name is required")
	}

	machineType := strings.TrimSpace(config.MachineType)
	if machineType != "" && !strings.Contains(machineType, "/") {
		machineType = fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType)
	}

	disks, err := buildDisks(project, zone, config)
	if err != nil {
		return nil, err
	}

	networkIfs := BuildNetworkInterfaces(project, region, config.NetworkingConfig)
	if len(networkIfs) == 0 {
		return nil, fmt.Errorf("at least one network interface is required")
	}

	scheduling, resourcePolicies := buildSchedulingAndResourcePolicies(zone, config)
	mgmt := managementConfigFromCreateVMConfig(config)
	adv := advancedConfigFromCreateVMConfig(config)
	metadata := buildInstanceMetadataFromConfig(mgmt, config)

	var serviceAccounts []*compute.ServiceAccount
	if strings.TrimSpace(config.ServiceAccount) != "" || len(NormalizeOAuthScopes(config.OAuthScopes)) > 0 {
		email := strings.TrimSpace(config.ServiceAccount)
		scopes := NormalizeOAuthScopes(config.OAuthScopes)
		if len(scopes) == 0 {
			scopes = []string{defaultOAuthScope}
		}
		serviceAccounts = []*compute.ServiceAccount{{Email: email, Scopes: scopes}}
	}

	guestAccel := BuildGuestAccelerators(adv)
	for _, a := range guestAccel {
		if a != nil && a.AcceleratorType != "" && !strings.Contains(a.AcceleratorType, "/") {
			a.AcceleratorType = fmt.Sprintf("zones/%s/acceleratorTypes/%s", zone, a.AcceleratorType)
		}
	}

	var displayDevice *compute.DisplayDevice
	if config.EnableDisplayDevice {
		displayDevice = &compute.DisplayDevice{EnableDisplay: true}
	}

	instance := &compute.Instance{
		Name:                       name,
		MachineType:                machineType,
		Disks:                      disks,
		NetworkInterfaces:          networkIfs,
		Scheduling:                 scheduling,
		Metadata:                   metadata,
		Tags:                       &compute.Tags{Items: ParseNetworkTags(config.NetworkTags)},
		Labels:                     BuildLabels(adv),
		ShieldedInstanceConfig:     BuildShieldedInstanceConfig(config.SecurityConfig),
		ConfidentialInstanceConfig: BuildConfidentialInstanceConfig(config.SecurityConfig),
		GuestAccelerators:          guestAccel,
		ResourcePolicies:           resourcePolicies,
		DisplayDevice:              displayDevice,
	}
	if len(serviceAccounts) > 0 {
		instance.ServiceAccounts = serviceAccounts
	}
	return instance, nil
}

func InsertInstance(ctx context.Context, client Client, project, zone string, instance *compute.Instance) ([]byte, error) {
	if project == "" {
		project = client.ProjectID()
	}
	path := fmt.Sprintf("projects/%s/zones/%s/instances", project, zone)
	return client.Post(ctx, path, instance)
}

type zoneOperationResp struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  *struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	} `json:"error"`
}

func WaitForZoneOperation(ctx context.Context, client Client, project, zone, operationName string) error {
	path := fmt.Sprintf("projects/%s/zones/%s/operations/%s", project, zone, operationName)
	deadline := time.Now().Add(defaultOperationWaitTimeout)
	ticker := time.NewTicker(operationPollInterval)
	defer ticker.Stop()
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for operation %s", operationName)
		}
		body, err := client.Get(ctx, path)
		if err != nil {
			return err
		}
		var op zoneOperationResp
		if err := json.Unmarshal(body, &op); err != nil {
			return fmt.Errorf("parse operation response: %w", err)
		}
		switch op.Status {
		case opStatusDone:
			if op.Error != nil && len(op.Error.Errors) > 0 {
				msg := op.Error.Errors[0].Message
				if msg == "" {
					msg = op.Error.Errors[0].Code
				}
				return fmt.Errorf("operation failed: %s", msg)
			}
			return nil
		case opStatusPending, opStatusRunning:
		default:
			return fmt.Errorf("unexpected operation status: %s", op.Status)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

type instanceGetResp struct {
	Id                uint64 `json:"id,string"`
	Name              string `json:"name"`
	SelfLink          string `json:"selfLink"`
	Status            string `json:"status"`
	Zone              string `json:"zone"`
	MachineType       string `json:"machineType"`
	NetworkInterfaces []struct {
		NetworkIP     string `json:"networkIP"`
		AccessConfigs []struct {
			NatIP string `json:"natIP"`
		} `json:"accessConfigs"`
	} `json:"networkInterfaces"`
}

func GetInstance(ctx context.Context, client Client, project, zone, name string) ([]byte, error) {
	if project == "" {
		project = client.ProjectID()
	}
	path := fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, zone, name)
	return client.Get(ctx, path)
}

func InstancePayloadFromGetResponse(body []byte, zone string) (map[string]any, error) {
	var inst instanceGetResp
	if err := json.Unmarshal(body, &inst); err != nil {
		return nil, fmt.Errorf("parse instance response: %w", err)
	}
	payload := map[string]any{
		"instanceId":  fmt.Sprintf("%d", inst.Id),
		"selfLink":    inst.SelfLink,
		"status":      inst.Status,
		"zone":        lastSegment(inst.Zone),
		"name":        inst.Name,
		"machineType": lastSegment(inst.MachineType),
	}
	if len(inst.NetworkInterfaces) > 0 {
		ni := inst.NetworkInterfaces[0]
		payload["internalIP"] = ni.NetworkIP
		if len(ni.AccessConfigs) > 0 && ni.AccessConfigs[0].NatIP != "" {
			payload["externalIP"] = ni.AccessConfigs[0].NatIP
		}
	}
	if payload["zone"] == "" && zone != "" {
		payload["zone"] = zone
	}
	return payload, nil
}

func CreateVMAndWait(ctx context.Context, client Client, config CreateVMConfig) (map[string]any, error) {
	project := strings.TrimSpace(config.Project)
	if project == "" {
		project = client.ProjectID()
	}
	zone := strings.TrimSpace(config.Zone)
	region := strings.TrimSpace(config.Region)
	if zone == "" {
		return nil, fmt.Errorf("zone is required")
	}
	if region == "" {
		region = deriveRegionFromZone(zone)
	}
	if region == "" {
		region = zone
	}
	zone = lastSegment(zone)
	region = lastSegment(region)

	if config.InternalIPType == InternalIPStatic && strings.TrimSpace(config.InternalIPAddress) != "" {
		resolved, err := ResolveInternalIPAddress(ctx, client, project, region, config.InternalIPAddress)
		if err != nil {
			return nil, fmt.Errorf("reserved internal IP: %w", err)
		}
		config.InternalIPAddress = resolved
	}

	instance, err := BuildInstanceFromConfig(project, zone, region, config)
	if err != nil {
		return nil, err
	}

	if len(config.FirewallRules) > 0 {
		firewallTags, err := ResolveFirewallRuleTags(ctx, client, project, config.FirewallRules)
		if err != nil {
			return nil, err
		}
		instance.Tags = &compute.Tags{Items: BuildInstanceTags(config.NetworkTags, firewallTags)}
	}

	body, err := InsertInstance(ctx, client, project, zone, instance)
	if err != nil {
		return nil, err
	}
	var opResp struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &opResp); err != nil || opResp.Name == "" {
		return nil, fmt.Errorf("parse insert operation response: %w", err)
	}

	if err := WaitForZoneOperation(ctx, client, project, zone, lastSegment(opResp.Name)); err != nil {
		return nil, err
	}

	instBody, err := GetInstance(ctx, client, project, zone, instance.Name)
	if err != nil {
		return nil, fmt.Errorf("fetch created instance: %w", err)
	}
	return InstancePayloadFromGetResponse(instBody, zone)
}
