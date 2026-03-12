package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	compute "google.golang.org/api/compute/v1"
)

type Client interface {
	Get(ctx context.Context, path string) ([]byte, error)
	Post(ctx context.Context, path string, body any) ([]byte, error)
	GetURL(ctx context.Context, fullURL string) ([]byte, error)
	ProjectID() string
}

var (
	clientFactoryMu sync.RWMutex
	clientFactory   func(ctx core.ExecutionContext) (Client, error)
)

func SetClientFactory(fn func(ctx core.ExecutionContext) (Client, error)) {
	clientFactoryMu.Lock()
	defer clientFactoryMu.Unlock()
	clientFactory = fn
}

func getClient(ctx core.ExecutionContext) (Client, error) {
	clientFactoryMu.RLock()
	fn := clientFactory
	clientFactoryMu.RUnlock()
	if fn == nil {
		panic("gcp compute: SetClientFactory was not called by the gcp integration")
	}
	return fn(ctx)
}

type ProvisioningModel string

const (
	ProvisioningStandard ProvisioningModel = "STANDARD"
	ProvisioningSpot     ProvisioningModel = "SPOT"
)

var publicImageOSOptions = []configuration.FieldOption{
	{Label: "CentOS", Value: "centos-cloud"},
	{Label: "Container-Optimized OS", Value: "cos-cloud"},
	{Label: "Debian", Value: "debian-cloud"},
	{Label: "Deep learning on Linux", Value: "deeplearning-platform-release"},
	{Label: "Fedora", Value: "fedora-cloud"},
	{Label: "openSUSE", Value: "opensuse-cloud"},
	{Label: "Oracle Linux", Value: "oracle-linux-cloud"},
	{Label: "Red Hat Linux", Value: "rhel-cloud"},
	{Label: "Red Hat Linux for SAP", Value: "rhel-sap-cloud"},
	{Label: "Rocky Linux", Value: "rocky-linux-cloud"},
	{Label: "Rocky Linux Accelerator Optimized", Value: "rocky-linux-accelerator-cloud"},
	{Label: "SQL Server on Windows Server", Value: "windows-sql-cloud"},
	{Label: "SUSE Linux Enterprise BYOS", Value: "suse-byos-cloud"},
	{Label: "SUSE Linux Enterprise Server", Value: "suse-cloud"},
	{Label: "SUSE Linux Enterprise Server for SAP", Value: "suse-sap-cloud"},
	{Label: "Ubuntu", Value: "ubuntu-os-cloud"},
	{Label: "Ubuntu Accelerator Optimized", Value: "ubuntu-os-accelerator-images"},
	{Label: "Ubuntu Pro", Value: "ubuntu-os-pro-cloud"},
	{Label: "Windows Server", Value: "windows-cloud"},
}

const (
	BootDiskSourcePublicImage  = "publicImage"
	BootDiskSourceCustomImage  = "customImage"
	BootDiskSourceSnapshot     = "snapshot"
	BootDiskSourceExistingDisk = "existingDisk"
)

const (
	AdditionalDiskModeNew      = "newDisk"
	AdditionalDiskModeExisting = "existingDisk"
)

func intPtr(n int) *int {
	return &n
}

func strPtr(s string) *string {
	return &s
}

const (
	DefaultDiskType   = "pd-balanced"
	DefaultDiskSizeGb = 10
)

type OSAndStorageConfig struct {
	BootDiskSourceType       string                `mapstructure:"bootDiskSourceType"`
	BootDiskOS               string                `mapstructure:"bootDiskOS"`
	BootDiskPublicImage      string                `mapstructure:"bootDiskPublicImage"`
	BootDiskCustomImage      string                `mapstructure:"bootDiskCustomImage"`
	BootDiskSnapshot         string                `mapstructure:"bootDiskSnapshot"`
	BootDiskExistingDisk     string                `mapstructure:"bootDiskExistingDisk"`
	BootDiskType             string                `mapstructure:"bootDiskType"`
	BootDiskSizeGb           int64                 `mapstructure:"bootDiskSizeGb"`
	BootDiskEncryptionKey    string                `mapstructure:"bootDiskEncryptionKey"`
	BootDiskSnapshotSchedule string                `mapstructure:"bootDiskSnapshotSchedule"`
	BootDiskAutoDelete       bool                  `mapstructure:"bootDiskAutoDelete"`
	LocalSSDCount            int64                 `mapstructure:"localSSDCount"`
	AdditionalDisks          []AdditionalDiskEntry `mapstructure:"additionalDisks"`
}

type AdditionalDiskEntry struct {
	Mode         string `mapstructure:"mode"`
	Name         string `mapstructure:"name"`
	SizeGb       int64  `mapstructure:"sizeGb"`
	DiskType     string `mapstructure:"diskType"`
	ExistingDisk string `mapstructure:"existingDisk"`
	AutoDelete   bool   `mapstructure:"autoDelete"`
}

type BootDiskConfig struct {
	Name              string
	DiskType          string
	SizeGb            int64
	SourceImage       string
	SourceSnapshot    string
	SourceDisk        string
	SnapshotSchedule  string
	AutoDelete        bool
	DiskEncryptionKey string
}

type AdditionalDisk struct {
	Name       string
	SizeGb     int64
	DiskType   string
	SourceDisk string
	AutoDelete bool
}

func BuildBootDisk(project, zone string, config BootDiskConfig) *compute.AttachedDisk {
	diskType := strings.TrimSpace(config.DiskType)
	if diskType == "" {
		diskType = DefaultDiskType
	}
	sizeGb := config.SizeGb
	if sizeGb < 1 {
		sizeGb = DefaultDiskSizeGb
	}
	autoDelete := config.AutoDelete
	diskEncryptionKey := buildDiskEncryptionKey(config.DiskEncryptionKey)

	if strings.TrimSpace(config.SourceDisk) != "" {
		att := &compute.AttachedDisk{
			Source:     config.SourceDisk,
			Boot:       true,
			AutoDelete: autoDelete,
			DeviceName: strings.TrimSpace(config.Name),
		}
		if diskEncryptionKey != nil {
			att.DiskEncryptionKey = diskEncryptionKey
		}
		return att
	}

	params := &compute.AttachedDiskInitializeParams{
		DiskName:   strings.TrimSpace(config.Name),
		DiskSizeGb: sizeGb,
		DiskType:   resolveDiskTypeURL(project, zone, diskType),
	}
	if config.SourceImage != "" {
		params.SourceImage = strings.TrimSpace(config.SourceImage)
	}
	if config.SourceSnapshot != "" {
		params.SourceSnapshot = strings.TrimSpace(config.SourceSnapshot)
	}
	if strings.TrimSpace(config.SnapshotSchedule) != "" {
		params.ResourcePolicies = []string{strings.TrimSpace(config.SnapshotSchedule)}
	}

	att := &compute.AttachedDisk{
		Boot:             true,
		AutoDelete:       autoDelete,
		InitializeParams: params,
		DeviceName:       strings.TrimSpace(config.Name),
	}
	if diskEncryptionKey != nil {
		att.DiskEncryptionKey = diskEncryptionKey
	}
	return att
}

func buildDiskEncryptionKey(kmsKeyName string) *compute.CustomerEncryptionKey {
	kmsKeyName = strings.TrimSpace(kmsKeyName)
	if kmsKeyName == "" {
		return nil
	}
	return &compute.CustomerEncryptionKey{KmsKeyName: kmsKeyName}
}

func BuildAdditionalDisks(project, zone string, disks []AdditionalDisk) []*compute.AttachedDisk {
	if len(disks) == 0 {
		return nil
	}
	out := make([]*compute.AttachedDisk, 0, len(disks))
	for _, d := range disks {
		att := buildOneAdditionalDisk(project, zone, d)
		if att != nil {
			out = append(out, att)
		}
	}
	return out
}

func buildOneAdditionalDisk(project, zone string, d AdditionalDisk) *compute.AttachedDisk {
	name := strings.TrimSpace(d.Name)
	if strings.TrimSpace(d.SourceDisk) != "" {
		return &compute.AttachedDisk{
			Source:     d.SourceDisk,
			Boot:       false,
			AutoDelete: d.AutoDelete,
			DeviceName: name,
		}
	}
	diskType := strings.TrimSpace(d.DiskType)
	if diskType == "" {
		diskType = DefaultDiskType
	}
	isLocalSSD := diskType == "local-ssd"
	params := &compute.AttachedDiskInitializeParams{
		DiskName: name,
		DiskType: resolveDiskTypeURL(project, zone, diskType),
	}
	if !isLocalSSD {
		sizeGb := d.SizeGb
		if sizeGb < 1 {
			sizeGb = DefaultDiskSizeGb
		}
		params.DiskSizeGb = sizeGb
	}
	att := &compute.AttachedDisk{
		Boot:             false,
		AutoDelete:       d.AutoDelete,
		InitializeParams: params,
		DeviceName:       name,
	}
	if isLocalSSD {
		att.Interface = "NVME"
		att.Type = "SCRATCH"
	}
	return att
}

func BuildLocalSSDDisks(project, zone string, count int) []*compute.AttachedDisk {
	if count <= 0 {
		return nil
	}
	if count > 8 {
		count = 8
	}
	out := make([]*compute.AttachedDisk, 0, count)
	diskTypeURL := resolveDiskTypeURL(project, zone, "local-ssd")
	for i := 0; i < count; i++ {
		deviceName := fmt.Sprintf("local-ssd-%d", i)
		out = append(out, &compute.AttachedDisk{
			Type:             "SCRATCH",
			Interface:        "NVME",
			AutoDelete:       true,
			DeviceName:       deviceName,
			InitializeParams: &compute.AttachedDiskInitializeParams{DiskType: diskTypeURL},
		})
	}
	return out
}

func resolveDiskTypeURL(project, zone, diskType string) string {
	if strings.Contains(diskType, "/") {
		return diskType
	}
	if project == "" || zone == "" {
		return diskType
	}
	return fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", project, zone, diskType)
}

const (
	ConfidentialInstanceTypeSEV    = "SEV"     // AMD Secure Encrypted Virtualization
	ConfidentialInstanceTypeSEVSNP = "SEV_SNP" // AMD SEV - Secure Nested Paging
	ConfidentialInstanceTypeTDX    = "TDX"     // Intel Trust Domain eXtension
)

const (
	fieldNameShieldedVM     = "shieldedVM"
	fieldNameConfidentialVM = "confidentialVM"
)

var (
	visibleWhenShieldedVM = []configuration.VisibilityCondition{
		{Field: fieldNameShieldedVM, Values: []string{"true"}},
	}
	visibleWhenConfidentialVM = []configuration.VisibilityCondition{
		{Field: fieldNameConfidentialVM, Values: []string{"true"}},
	}
)

type SecurityConfig struct {
	ShieldedVM                          bool   `mapstructure:"shieldedVM"`
	ShieldedVMEnableSecureBoot          bool   `mapstructure:"shieldedVMEnableSecureBoot"`
	ShieldedVMEnableVtpm                bool   `mapstructure:"shieldedVMEnableVtpm"`
	ShieldedVMEnableIntegrityMonitoring bool   `mapstructure:"shieldedVMEnableIntegrityMonitoring"`
	ConfidentialVM                      bool   `mapstructure:"confidentialVM"`
	ConfidentialVMType                  string `mapstructure:"confidentialVMType"`
}

func BuildShieldedInstanceConfig(config SecurityConfig) *compute.ShieldedInstanceConfig {
	if !config.ShieldedVM {
		return nil
	}
	return &compute.ShieldedInstanceConfig{
		EnableSecureBoot:          config.ShieldedVMEnableSecureBoot,
		EnableVtpm:                config.ShieldedVMEnableVtpm,
		EnableIntegrityMonitoring: config.ShieldedVMEnableIntegrityMonitoring,
	}
}

func BuildConfidentialInstanceConfig(config SecurityConfig) *compute.ConfidentialInstanceConfig {
	if !config.ConfidentialVM {
		return nil
	}
	confidentialType := config.ConfidentialVMType
	if confidentialType == "" {
		confidentialType = ConfidentialInstanceTypeSEV
	}
	return &compute.ConfidentialInstanceConfig{
		EnableConfidentialCompute: true,
		ConfidentialInstanceType:  confidentialType,
	}
}

type IdentityConfig struct {
	ServiceAccount      string   `mapstructure:"serviceAccount"`
	OAuthScopes         []string `mapstructure:"oauthScopes"`
	BlockProjectSSHKeys bool     `mapstructure:"blockProjectSSHKeys"`
	EnableOSLogin       bool     `mapstructure:"enableOSLogin"`
}

func NormalizeOAuthScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	result := make([]string, 0, len(scopes))
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

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

func getAddressIP(ctx context.Context, c Client, project, region, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", nil
	}
	var body []byte
	var err error
	if strings.Contains(id, "://") {
		body, err = c.GetURL(ctx, id)
	} else if strings.HasPrefix(id, "projects/") {
		body, err = c.Get(ctx, id)
	} else {
		project = ensureProject(project, c)
		region, errR := ensureNonEmptyRegion(region, "region is required to resolve address by name")
		if errR != nil {
			return "", errR
		}
		path := fmt.Sprintf("projects/%s/regions/%s/addresses/%s", project, region, id)
		body, err = c.Get(ctx, path)
	}
	if err != nil {
		return "", err
	}
	var a addressItem
	if err := json.Unmarshal(body, &a); err != nil {
		return "", fmt.Errorf("parse address: %w", err)
	}
	return strings.TrimSpace(a.Address), nil
}

// ResolveInternalIPAddress returns a literal IP for use as networkIP. If value is a literal IP (no slash),
// it is returned as-is. If value is a full URL or resource path to a reserved address, it is fetched and
// the address field (literal IP) is returned. Compute Engine networkInterfaces.networkIP accepts only literal IPs.
func ResolveInternalIPAddress(ctx context.Context, c Client, project, region, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !strings.Contains(value, "/") {
		return value, nil
	}
	return getAddressIP(ctx, c, project, region, value)
}

func BuildInstanceTags(networkTags string, firewallTags []string) []string {
	out := ParseNetworkTags(networkTags)
	seen := make(map[string]struct{})
	for _, t := range out {
		seen[t] = struct{}{}
	}
	for _, t := range firewallTags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// parseAllowed parses an "allowed" string like "tcp:22" or "tcp:80,tcp:443" into GCP FirewallAllowed entries.
// Format: comma-separated protocol:port (e.g. tcp:22, udp:53). Same protocol can appear multiple times; ports are grouped.
func parseAllowed(s string) ([]*compute.FirewallAllowed, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("allowed is required")
	}
	// Group by protocol: map[protocol][]port
	byProto := make(map[string][]string)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.LastIndex(part, ":")
		if idx < 0 {
			return nil, fmt.Errorf("invalid allowed %q: expected protocol:port (e.g. tcp:22)", part)
		}
		proto := strings.TrimSpace(part[:idx])
		port := strings.TrimSpace(part[idx+1:])
		if proto == "" {
			return nil, fmt.Errorf("invalid allowed %q: protocol is empty", part)
		}
		if port != "" {
			byProto[proto] = append(byProto[proto], port)
		} else {
			byProto[proto] = nil // all ports
		}
	}
	if len(byProto) == 0 {
		return nil, fmt.Errorf("allowed is required")
	}
	out := make([]*compute.FirewallAllowed, 0, len(byProto))
	for proto, ports := range byProto {
		a := &compute.FirewallAllowed{IPProtocol: proto}
		if len(ports) > 0 {
			a.Ports = ports
		}
		out = append(out, a)
	}
	return out, nil
}

// CreateFirewallRule creates a single firewall rule in the project. If the rule already exists (409), it is treated as success.
func CreateFirewallRule(ctx context.Context, c Client, project, network string, rule CreateFirewallRuleEntry) error {
	name := strings.TrimSpace(rule.Name)
	if name == "" {
		return fmt.Errorf("firewall rule name is required")
	}
	allowed, err := parseAllowed(rule.Allowed)
	if err != nil {
		return err
	}
	sourceRanges := strings.Split(rule.SourceRanges, ",")
	for i := range sourceRanges {
		sourceRanges[i] = strings.TrimSpace(sourceRanges[i])
		if sourceRanges[i] == "" {
			continue
		}
	}
	trimmed := make([]string, 0, len(sourceRanges))
	for _, r := range sourceRanges {
		if r != "" {
			trimmed = append(trimmed, r)
		}
	}
	if len(trimmed) == 0 {
		return fmt.Errorf("sourceRanges is required")
	}
	targetTag := strings.TrimSpace(rule.TargetTag)
	if targetTag == "" {
		return fmt.Errorf("targetTag is required")
	}
	project = ensureProject(project, c)
	networkURL := resolveNetworkURL(project, network)
	if networkURL == "" {
		networkURL = fmt.Sprintf("projects/%s/global/networks/default", project)
	}
	fw := &compute.Firewall{
		Name:         name,
		Network:      networkURL,
		Direction:    "INGRESS",
		Allowed:      allowed,
		SourceRanges: trimmed,
		TargetTags:   []string{targetTag},
	}
	path := fmt.Sprintf("projects/%s/global/firewalls", project)
	_, err = c.Post(ctx, path, fw)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "already exists") || strings.Contains(errStr, "409") {
			return nil
		}
		return err
	}
	return nil
}

// EnsureFirewallRules creates each rule and returns the list of target tags to apply to the instance.
func EnsureFirewallRules(ctx context.Context, c Client, project, network string, rules []CreateFirewallRuleEntry) ([]string, error) {
	if len(rules) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{})
	var tags []string
	for _, r := range rules {
		if strings.TrimSpace(r.Name) == "" {
			continue
		}
		if err := CreateFirewallRule(ctx, c, project, network, r); err != nil {
			return nil, fmt.Errorf("create firewall rule %q: %w", r.Name, err)
		}
		tag := strings.TrimSpace(r.TargetTag)
		if tag != "" {
			if _, ok := seen[tag]; !ok {
				seen[tag] = struct{}{}
				tags = append(tags, tag)
			}
		}
	}
	return tags, nil
}

type NetworkingConfig struct {
	Network             string                    `mapstructure:"network"`
	Subnetwork          string                    `mapstructure:"subnetwork"`
	NicType             string                    `mapstructure:"nicType"`
	InternalIPType      string                    `mapstructure:"internalIPType"`
	InternalIPAddress   string                    `mapstructure:"internalIPAddress"`
	ExternalIPType      string                    `mapstructure:"externalIPType"`
	ExternalIPAddress   string                    `mapstructure:"externalIPAddress"`
	NetworkTags         string                    `mapstructure:"networkTags"`
	StackType           string                    `mapstructure:"stackType"`
	CreateFirewallRules []CreateFirewallRuleEntry `mapstructure:"createFirewallRules"`
}

type CreateFirewallRuleEntry struct {
	Name         string `mapstructure:"name"`
	Allowed      string `mapstructure:"allowed"`
	SourceRanges string `mapstructure:"sourceRanges"`
	TargetTag    string `mapstructure:"targetTag"`
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

const (
	OnHostMaintenanceMigrate   = "MIGRATE"
	OnHostMaintenanceTerminate = "TERMINATE"
)

const (
	metadataKeyStartupScript  = "startup-script"
	metadataKeyShutdownScript = "shutdown-script"
)

type ManagementConfig struct {
	MetadataItems     []MetadataKeyValue `mapstructure:"metadataItems"`
	StartupScript     string             `mapstructure:"startupScript"`
	ShutdownScript    string             `mapstructure:"shutdownScript"`
	AutomaticRestart  *bool              `mapstructure:"automaticRestart"`
	OnHostMaintenance string             `mapstructure:"onHostMaintenance"`
	MaintenancePolicy string             `mapstructure:"maintenancePolicy"`
}

type MetadataKeyValue struct {
	Key   string `mapstructure:"key"`
	Value string `mapstructure:"value"`
}

func BuildInstanceMetadata(config ManagementConfig) *compute.Metadata {
	var items []*compute.MetadataItems
	seen := make(map[string]bool)

	if script := strings.TrimSpace(config.StartupScript); script != "" {
		items = append(items, &compute.MetadataItems{Key: metadataKeyStartupScript, Value: &script})
		seen[metadataKeyStartupScript] = true
	}
	if script := strings.TrimSpace(config.ShutdownScript); script != "" {
		items = append(items, &compute.MetadataItems{Key: metadataKeyShutdownScript, Value: &script})
		seen[metadataKeyShutdownScript] = true
	}

	for _, m := range config.MetadataItems {
		k := strings.TrimSpace(m.Key)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		v := strings.TrimSpace(m.Value)
		vCopy := v
		items = append(items, &compute.MetadataItems{Key: k, Value: &vCopy})
	}

	if len(items) == 0 {
		return nil
	}
	return &compute.Metadata{Items: items}
}

func BuildScheduling(config ManagementConfig) *compute.Scheduling {
	automaticRestart := true
	if config.AutomaticRestart != nil {
		automaticRestart = *config.AutomaticRestart
	}
	onHostMaintenance := OnHostMaintenanceMigrate
	if strings.TrimSpace(config.OnHostMaintenance) == OnHostMaintenanceTerminate {
		onHostMaintenance = OnHostMaintenanceTerminate
	}
	return &compute.Scheduling{
		AutomaticRestart:  &automaticRestart,
		OnHostMaintenance: onHostMaintenance,
	}
}

const (
	NodeAffinityOperatorIn    = "IN"
	NodeAffinityOperatorNotIn = "NOT_IN"
)

type AdvancedConfig struct {
	GuestAccelerators      []GuestAcceleratorEntry `mapstructure:"guestAccelerators"`
	NodeAffinities         []NodeAffinityEntry     `mapstructure:"nodeAffinities"`
	ResourcePolicies       []string                `mapstructure:"resourcePolicies"`
	MinNodeCpus            int64                   `mapstructure:"minNodeCpus"`
	Labels                 []LabelEntry            `mapstructure:"labels"`
	EnableDisplayDevice    bool                    `mapstructure:"enableDisplayDevice"`
	EnableSerialPortAccess bool                    `mapstructure:"enableSerialPortAccess"`
}

type LabelEntry struct {
	Key   string `mapstructure:"key"`
	Value string `mapstructure:"value"`
}

type GuestAcceleratorEntry struct {
	AcceleratorType  string `mapstructure:"acceleratorType"`
	AcceleratorCount int64  `mapstructure:"acceleratorCount"`
}

type NodeAffinityEntry struct {
	Key      string   `mapstructure:"key"`
	Operator string   `mapstructure:"operator"`
	Values   []string `mapstructure:"values"`
}

func trimmedNonEmptyStrings(ss []string) []string {
	var out []string
	for _, s := range ss {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func normalizeNodeAffinityOperator(op string) string {
	if strings.TrimSpace(op) == NodeAffinityOperatorNotIn {
		return NodeAffinityOperatorNotIn
	}
	return NodeAffinityOperatorIn
}

func BuildGuestAccelerators(config AdvancedConfig) []*compute.AcceleratorConfig {
	var out []*compute.AcceleratorConfig
	for _, e := range config.GuestAccelerators {
		t := strings.TrimSpace(e.AcceleratorType)
		if t == "" || e.AcceleratorCount < 1 {
			continue
		}
		out = append(out, &compute.AcceleratorConfig{
			AcceleratorType:  t,
			AcceleratorCount: e.AcceleratorCount,
		})
	}
	return out
}

func BuildNodeAffinities(config AdvancedConfig) []*compute.SchedulingNodeAffinity {
	var out []*compute.SchedulingNodeAffinity
	for _, e := range config.NodeAffinities {
		key := strings.TrimSpace(e.Key)
		values := trimmedNonEmptyStrings(e.Values)
		if key == "" || len(values) == 0 {
			continue
		}
		op := normalizeNodeAffinityOperator(e.Operator)
		out = append(out, &compute.SchedulingNodeAffinity{
			Key:      key,
			Operator: op,
			Values:   values,
		})
	}
	return out
}

func BuildInstanceResourcePolicies(config AdvancedConfig) []string {
	return trimmedNonEmptyStrings(config.ResourcePolicies)
}

func BuildLabels(config AdvancedConfig) map[string]string {
	if len(config.Labels) == 0 {
		return nil
	}
	out := make(map[string]string)
	for _, e := range config.Labels {
		k := strings.TrimSpace(e.Key)
		if k == "" {
			continue
		}
		if _, exists := out[k]; exists {
			continue
		}
		out[k] = strings.TrimSpace(e.Value)
	}
	return out
}

func ApplyAdvancedScheduling(s *compute.Scheduling, config AdvancedConfig) {
	if s == nil {
		return
	}
	if affinities := BuildNodeAffinities(config); len(affinities) > 0 {
		s.NodeAffinities = affinities
	}
	if config.MinNodeCpus > 0 {
		s.MinNodeCpus = config.MinNodeCpus
	}
}

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
	return ManagementConfig{
		MetadataItems:     c.MetadataItems,
		StartupScript:     c.StartupScript,
		ShutdownScript:    c.ShutdownScript,
		AutomaticRestart:  c.AutomaticRestart,
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
	project := client.ProjectID()
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

	var firewallTags []string
	if len(config.CreateFirewallRules) > 0 {
		createdTags, err := EnsureFirewallRules(ctx, client, project, config.Network, config.CreateFirewallRules)
		if err != nil {
			return nil, err
		}
		firewallTags = append(firewallTags, createdTags...)
	}
	if len(firewallTags) > 0 {
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

var gcpInstanceNameRegex = regexp.MustCompile(`^[a-z](?:[-a-z0-9]{0,61}[a-z0-9])?$`)

const (
	createVMPayloadType   = "gcp.createVM.completed"
	createVMOutputChannel = "default"
)

type CreateVM struct{}

func (c *CreateVM) Name() string {
	return "gcp.createVM"
}

func (c *CreateVM) Label() string {
	return "Create Virtual Machine"
}

func (c *CreateVM) Description() string {
	return "Create a Google Compute Engine VM. Configure machine type, zone, provisioning model, and more."
}

func (c *CreateVM) Documentation() string {
	return `Creates a new Google Compute Engine VM.

## Steps

1. **Machine Configuration** – Region, zone, machine type, provisioning model (Spot/Standard), instance name.
2. **OS & Storage** – Boot disk source (public/custom image, snapshot, existing disk), disk type, size, snapshot schedule.
3. **Security** – Shielded VM (secure boot, vTPM, integrity monitoring), Confidential VM (AMD SEV/SEV-SNP, Intel TDX).
4. **Identity & API access** – VM service account, OAuth scopes, OS Login, block project-wide SSH keys.
5. **Networking** – VPC, subnet, NIC type, internal/external IP (including static), network tags, firewall rules.
6. **Management** – Metadata, startup script, automatic restart, on host maintenance, maintenance policy.
7. **Advanced** – GPU accelerators, placement policy (min node CPUs), sole-tenant/host affinity, resource policies.

## Output

Emits a payload with instance details: instanceId, selfLink, internalIP, externalIP, status, zone, name, machineType.`
}

func (c *CreateVM) Icon() string {
	return "server"
}

func (c *CreateVM) Color() string {
	return "gray"
}

func (c *CreateVM) ExampleOutput() map[string]any {
	return map[string]any{
		"instanceId":  "1234567890123456789",
		"selfLink":    "https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/instances/my-vm",
		"internalIP":  "10.0.0.2",
		"externalIP":  "34.1.2.3",
		"status":      "RUNNING",
		"zone":        "us-central1-a",
		"name":        "my-vm",
		"machineType": "e2-medium",
	}
}

func (c *CreateVM) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: createVMOutputChannel, Label: "Default"},
	}
}

func (c *CreateVM) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceName",
			Label:       "Instance name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Start with a letter; use only a-z, 0-9, and hyphens; end with a letter or digit. 1 to 63 characters length.",
			Placeholder: "e.g. my-vm-01",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "GCP region (e.g. us-central1). Used to filter zones.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRegion,
				},
			},
		},
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "GCP zone within the selected region (e.g. us-central1-a).",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeZone,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
					},
				},
			},
		},
		{
			Name:        "machineFamily",
			Label:       "Machine family",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional. Filter machine types by family (e.g. E2). Leave empty to see all.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeMachineFamily,
					Parameters: []configuration.ParameterRef{
						{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
					},
				},
			},
		},
		{
			Name:        "machineType",
			Label:       "Machine type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Machine type for the VM (e.g. e2-medium, n2-standard-4).",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeMachineType,
					Parameters: []configuration.ParameterRef{
						{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
						{Name: "machineFamily", ValueFrom: &configuration.ParameterValueFrom{Field: "machineFamily"}},
					},
				},
			},
		},
		{
			Name:        "provisioningModel",
			Label:       "Provisioning model",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Standard (on-demand) or Spot (preemptible).",
			Default:     string(ProvisioningStandard),
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Standard", Value: string(ProvisioningStandard)},
						{Label: "Spot", Value: string(ProvisioningSpot)},
					},
				},
			},
		},
		{
			Name:        "bootDiskSourceType",
			Label:       "Boot disk source",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Source for boot disk: public image, custom image, snapshot, or existing disk.",
			Default:     BootDiskSourcePublicImage,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Public image", Value: BootDiskSourcePublicImage},
						{Label: "Custom image", Value: BootDiskSourceCustomImage},
						{Label: "Snapshot", Value: BootDiskSourceSnapshot},
						{Label: "Existing disk", Value: BootDiskSourceExistingDisk},
					},
				},
			},
		},
		{
			Name:        "bootDiskOS",
			Label:       "Operating system",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Select the OS (e.g. Debian, Ubuntu). Then pick a version below.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: publicImageOSOptions,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourcePublicImage}},
			},
		},
		{
			Name:        "bootDiskPublicImage",
			Label:       "Version",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select the image version for the chosen operating system.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePublicImages,
					Parameters: []configuration.ParameterRef{
						{Name: "project", ValueFrom: &configuration.ParameterValueFrom{Field: "bootDiskOS"}},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourcePublicImage}},
			},
		},
		{
			Name:        "bootDiskCustomImage",
			Label:       "Custom image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select a custom image from your project.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeCustomImages,
					Parameters: []configuration.ParameterRef{},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourceCustomImage}},
			},
		},
		{
			Name:        "bootDiskSnapshot",
			Label:       "Snapshot",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select a snapshot to create the boot disk from.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeSnapshots,
					Parameters: []configuration.ParameterRef{},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourceSnapshot}},
			},
		},
		{
			Name:        "bootDiskExistingDisk",
			Label:       "Existing disk",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select an existing disk in the same zone to use as the boot disk.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeDisks,
					Parameters: []configuration.ParameterRef{
						{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourceExistingDisk}},
			},
		},
		{
			Name:        "bootDiskType",
			Label:       "Boot disk type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Balanced, SSD, or Standard persistent disk.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeDiskTypes,
					Parameters: []configuration.ParameterRef{
						{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
						{Name: "bootDiskOnly", Value: strPtr("true")},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourcePublicImage, BootDiskSourceCustomImage, BootDiskSourceSnapshot}},
			},
		},
		{
			Name:        "bootDiskSizeGb",
			Label:       "Boot disk size (GB)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Boot disk size in GB. Provision between 10 and 65536 GB.",
			Default:     DefaultDiskSizeGb,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{Min: intPtr(10), Max: intPtr(65536)},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourcePublicImage, BootDiskSourceCustomImage, BootDiskSourceSnapshot}},
			},
		},
		{
			Name:        "bootDiskEncryptionKey",
			Label:       "Disk encryption key (optional)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Cloud KMS key resource name for customer-managed encryption (CMEK). Leave empty for Google-managed encryption.",
			Placeholder: "e.g. projects/my-project/locations/region/keyRings/ring/cryptoKeys/key",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourcePublicImage, BootDiskSourceCustomImage, BootDiskSourceSnapshot}},
			},
		},
		{
			Name:        "bootDiskSnapshotSchedule",
			Label:       "Snapshot schedule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional resource policy for automatic snapshot schedule.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeSnapshotSchedules,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
					},
				},
			},
		},
		{
			Name:        "bootDiskAutoDelete",
			Label:       "Delete boot disk on termination",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Delete the boot disk when the instance is deleted.",
			Default:     true,
		},
		{
			Name:        "localSSDCount",
			Label:       "Local SSD count",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Number of local SSDs to attach (0–8). Each local SSD is ~375 GB, NVME interface.",
			Default:     0,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{Min: intPtr(0), Max: intPtr(8)},
			},
		},
		{
			Name:        "additionalDisks",
			Label:       "Additional disks",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Add disks: create new disks or attach existing ones from the same zone.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Disk",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "mode",
								Label:       "Type",
								Type:        configuration.FieldTypeSelect,
								Required:    true,
								Description: "Add a new disk or attach an existing disk.",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Add new disk", Value: AdditionalDiskModeNew},
											{Label: "Attach existing disk", Value: AdditionalDiskModeExisting},
										},
									},
								},
							},
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Device / disk name (e.g. data-disk-1).",
								Placeholder: "e.g. data-disk-1",
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "mode", Values: []string{AdditionalDiskModeNew}},
								},
							},
							{
								Name:        "sizeGb",
								Label:       "Size (GB)",
								Type:        configuration.FieldTypeNumber,
								Required:    false,
								Description: "Size of the new disk in GB. Not used for Local SSD (fixed size per disk).",
								Default:     10,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{Min: intPtr(1), Max: intPtr(65536)},
								},
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "mode", Values: []string{AdditionalDiskModeNew}},
									{Field: "diskType", Values: []string{"pd-balanced", "pd-ssd", "pd-standard", "hyperdisk-balanced", "hyperdisk-throughput"}},
								},
							},
							{
								Name:        "diskType",
								Label:       "Disk type",
								Type:        configuration.FieldTypeSelect,
								Required:    false,
								Description: "Disk type for the new disk. Resolved per zone when creating the VM.",
								Default:     DefaultDiskType,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Balanced persistent disk", Value: "pd-balanced"},
											{Label: "SSD persistent disk", Value: "pd-ssd"},
											{Label: "Standard persistent disk", Value: "pd-standard"},
											{Label: "Hyperdisk Balanced", Value: "hyperdisk-balanced"},
											{Label: "Hyperdisk Throughput", Value: "hyperdisk-throughput"},
											{Label: "Local SSD", Value: "local-ssd"},
										},
									},
								},
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "mode", Values: []string{AdditionalDiskModeNew}},
								},
							},
							{
								Name:        "existingDisk",
								Label:       "Existing disk",
								Type:        configuration.FieldTypeIntegrationResource,
								Required:    false,
								Description: "Select an existing disk in the same zone to attach.",
								TypeOptions: &configuration.TypeOptions{
									Resource: &configuration.ResourceTypeOptions{
										Type: ResourceTypeDisks,
										Parameters: []configuration.ParameterRef{
											{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
										},
									},
								},
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "mode", Values: []string{AdditionalDiskModeExisting}},
								},
							},
							{
								Name:        "autoDelete",
								Label:       "Delete on termination",
								Type:        configuration.FieldTypeBool,
								Required:    false,
								Description: "Delete this disk when the instance is terminated.",
								Default:     true,
							},
						},
					},
				},
			},
		},
		{
			Name:        fieldNameShieldedVM,
			Label:       "Shielded VM",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Use Shielded VM for verified boot and measured boot. Enables vTPM and integrity monitoring by default; you can optionally enable Secure Boot.",
			Default:     false,
		},
		{
			Name:                 "shieldedVMEnableSecureBoot",
			Label:                "Secure Boot",
			Type:                 configuration.FieldTypeBool,
			Required:             false,
			Description:          "Verify digital signatures of all boot components. Disabled by default due to possible compatibility issues with unsigned drivers.",
			Default:              false,
			VisibilityConditions: visibleWhenShieldedVM,
		},
		{
			Name:                 "shieldedVMEnableVtpm",
			Label:                "vTPM",
			Type:                 configuration.FieldTypeBool,
			Required:             false,
			Description:          "Virtual Trusted Platform Module for measured boot and key storage.",
			Default:              true,
			VisibilityConditions: visibleWhenShieldedVM,
		},
		{
			Name:                 "shieldedVMEnableIntegrityMonitoring",
			Label:                "Integrity monitoring",
			Type:                 configuration.FieldTypeBool,
			Required:             false,
			Description:          "Monitor boot integrity against a baseline from the trusted boot image.",
			Default:              true,
			VisibilityConditions: visibleWhenShieldedVM,
		},
		{
			Name:        fieldNameConfidentialVM,
			Label:       "Confidential VM",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Run the VM with Confidential Computing (memory encrypted by the host). Requires a supported machine type (e.g. N2D, C2D).",
			Default:     false,
		},
		{
			Name:                 "confidentialVMType",
			Label:                "Confidential instance type",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			Description:          "Technology used for confidential compute. SEV (AMD) is common; SEV-SNP and TDX (Intel) depend on machine type and availability.",
			Default:              ConfidentialInstanceTypeSEV,
			VisibilityConditions: visibleWhenConfidentialVM,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "AMD SEV", Value: ConfidentialInstanceTypeSEV},
						{Label: "AMD SEV-SNP", Value: ConfidentialInstanceTypeSEVSNP},
						{Label: "Intel TDX", Value: ConfidentialInstanceTypeTDX},
					},
				},
			},
		},
		{
			Name:        "serviceAccount",
			Label:       "Service account (VM identity)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Email of the service account this VM will run as. Leave empty to use the project's default Compute Engine service account.",
			Placeholder: "e.g. my-sa@my-project.iam.gserviceaccount.com",
		},
		{
			Name:        "oauthScopes",
			Label:       "OAuth scopes",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Access scopes for the VM (which APIs the instance can call). Leave empty for default (cloud-platform).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Scope",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "blockProjectSSHKeys",
			Label:       "Block project-wide SSH keys",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "If enabled, only instance-level SSH keys or OS Login will work; project-wide SSH keys are ignored.",
			Default:     false,
		},
		{
			Name:        "enableOSLogin",
			Label:       "Enable OS Login",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Use OS Login for SSH access (IAM-based). When enabled, SSH keys are managed via IAM and OS Login.",
			Default:     false,
		},
		{
			Name:        "network",
			Label:       "VPC network",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "VPC network for the VM. Leave empty to use the default network.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeNetwork,
					Parameters: []configuration.ParameterRef{},
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
			Name:        "createFirewallRules",
			Label:       "Create firewall rules",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Create new firewall rules in the project and apply their target tag to this instance (e.g. allow SSH from any IP, or HTTP/HTTPS from a specific IP).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Firewall rule to create",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Rule name",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Unique name for the firewall rule (lowercase, numbers, hyphens; 1–63 chars).",
								Placeholder: "e.g. allow-ssh",
							},
							{
								Name:        "allowed",
								Label:       "Allowed",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Protocol and ports: tcp:22 (SSH), tcp:80,tcp:443 (HTTP/HTTPS), or e.g. udp:53.",
								Placeholder: "e.g. tcp:22 or tcp:80,tcp:443",
							},
							{
								Name:        "sourceRanges",
								Label:       "Source ranges",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "CIDR ranges that can reach the VM (e.g. 0.0.0.0/0 for any IP, or 203.0.113.50/32 for one IP).",
								Placeholder: "e.g. 0.0.0.0/0",
							},
							{
								Name:        "targetTag",
								Label:       "Target tag",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Tag applied to this rule and to the VM so the rule applies (e.g. ssh or web).",
								Placeholder: "e.g. ssh",
							},
						},
					},
				},
			},
		},
		{
			Name:        "metadataItems",
			Label:       "Custom metadata",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Optional key-value metadata for the instance.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Metadata",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Label:       "Key",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Metadata key (e.g. my-config, role).",
								Placeholder: "e.g. my-config",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Metadata value.",
								Placeholder: "e.g. production",
							},
						},
					},
				},
			},
		},
		{
			Name:        "startupScript",
			Label:       "Startup script (optional)",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Script that runs when the instance boots.",
			Placeholder: "#!/bin/bash\necho 'Hello from startup script'",
		},
		{
			Name:        "shutdownScript",
			Label:       "Shutdown script (optional)",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Script that runs when the instance is shut down.",
			Placeholder: "#!/bin/bash\necho 'Goodbye from shutdown script'",
		},
		{
			Name:        "automaticRestart",
			Label:       "Automatic restart",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Restart the VM automatically when it crashes or is terminated by the system.",
			Default:     true,
		},
		{
			Name:        "onHostMaintenance",
			Label:       "On host maintenance",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "When the host undergoes maintenance: migrate the VM to another host, or terminate it.",
			Default:     OnHostMaintenanceMigrate,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Migrate VM (recommended)", Value: OnHostMaintenanceMigrate},
						{Label: "Terminate VM", Value: OnHostMaintenanceTerminate},
					},
				},
			},
		},
		{
			Name:        "maintenancePolicy",
			Label:       "Maintenance policy",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional resource policy URL for instance scheduling (e.g. start/stop windows).",
			Placeholder: "e.g. projects/my-project/regions/region/resourcePolicies/my-policy",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Key-value labels for the instance (billing, environment, team).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Label:       "Key",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label key (e.g. env, team, cost-center).",
								Placeholder: "e.g. env",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Label value.",
								Placeholder: "e.g. production",
							},
						},
					},
				},
			},
		},
		{
			Name:        "guestAccelerators",
			Label:       "GPU accelerators",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Optional GPU or other accelerator cards (e.g. NVIDIA T4, V100, A100, L4).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Accelerator",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "acceleratorType",
								Label:       "Accelerator type",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Type name or full URL (e.g. nvidia-tesla-t4, nvidia-l4).",
								Placeholder: "e.g. nvidia-tesla-t4",
							},
							{
								Name:        "acceleratorCount",
								Label:       "Count",
								Type:        configuration.FieldTypeNumber,
								Required:    true,
								Description: "Number of accelerator cards to attach.",
								Default:     1,
							},
						},
					},
				},
			},
		},
		{
			Name:        "minNodeCpus",
			Label:       "Min node CPUs (placement)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "For sole-tenant: minimum number of virtual CPUs this instance will consume on a node. Leave empty for shared tenancy.",
			Placeholder: "e.g. 4",
		},
		{
			Name:        "nodeAffinities",
			Label:       "Node affinity (sole-tenant / host)",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Constrain placement to specific nodes (e.g. sole-tenant node groups).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Affinity rule",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Label:       "Key",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label key of the node (e.g. compute.googleapis.com/node-group).",
								Placeholder: "e.g. compute.googleapis.com/node-group",
							},
							{
								Name:        "operator",
								Label:       "Operator",
								Type:        configuration.FieldTypeSelect,
								Required:    true,
								Description: "IN: instance must run on nodes with one of the values; NOT_IN: avoid those nodes.",
								Default:     NodeAffinityOperatorIn,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "IN (affinity)", Value: NodeAffinityOperatorIn},
											{Label: "NOT IN (anti-affinity)", Value: NodeAffinityOperatorNotIn},
										},
									},
								},
							},
							{
								Name:        "values",
								Label:       "Values",
								Type:        configuration.FieldTypeList,
								Required:    true,
								Description: "Node label values (e.g. node group names) to match.",
								TypeOptions: &configuration.TypeOptions{
									List: &configuration.ListTypeOptions{
										ItemLabel: "Value",
										ItemDefinition: &configuration.ListItemDefinition{
											Type: configuration.FieldTypeString,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "resourcePolicies",
			Label:       "Resource policies",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Instance resource policy URLs (e.g. for start/stop schedules).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Policy URL",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "enableDisplayDevice",
			Label:       "Enable display device",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Enable a virtual display device for the instance.",
			Default:     false,
		},
		{
			Name:        "enableSerialPortAccess",
			Label:       "Enable serial port access",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Allow connecting to the instance serial console.",
			Default:     false,
		},
	}
}

func (c *CreateVM) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateVM) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateVM) Execute(ctx core.ExecutionContext) error {
	var config CreateVMConfig
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	if msg, ok := validateCreateVMConfig(config); !ok {
		return ctx.ExecutionState.Fail("error", msg)
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	callCtx := context.Background()
	payload, err := CreateVMAndWait(callCtx, client, config)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	return ctx.ExecutionState.Emit(createVMOutputChannel, createVMPayloadType, []any{payload})
}

func (c *CreateVM) Actions() []core.Action {
	return nil
}

func (c *CreateVM) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateVM) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateVM) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateVM) Cleanup(ctx core.SetupContext) error {
	return nil
}

func validateCreateVMConfig(config CreateVMConfig) (invalidMessage string, ok bool) {
	name := strings.TrimSpace(config.InstanceName)
	if name == "" {
		return "instance name is required", false
	}
	if !gcpInstanceNameRegex.MatchString(name) {
		return "instance name must be 1–63 characters: start with a lowercase letter, use only lowercase letters (a-z), digits (0-9), and hyphens (-), and end with a letter or digit (e.g. my-vm-01)", false
	}
	if strings.TrimSpace(config.Zone) == "" {
		return "zone is required", false
	}
	if strings.TrimSpace(config.MachineType) == "" {
		return "machine type is required", false
	}
	return "", true
}

type CreateVMConfig struct {
	InstanceName           string                  `mapstructure:"instanceName"`
	Region                 string                  `mapstructure:"region"`
	Zone                   string                  `mapstructure:"zone"`
	MachineFamily          string                  `mapstructure:"machineFamily"`
	MachineType            string                  `mapstructure:"machineType"`
	ProvisioningModel      string                  `mapstructure:"provisioningModel"`
	AutomaticRestart       *bool                   `mapstructure:"automaticRestart"`
	OnHostMaintenance      string                  `mapstructure:"onHostMaintenance"`
	MetadataItems          []MetadataKeyValue      `mapstructure:"metadataItems"`
	StartupScript          string                  `mapstructure:"startupScript"`
	ShutdownScript         string                  `mapstructure:"shutdownScript"`
	MaintenancePolicy      string                  `mapstructure:"maintenancePolicy"`
	Labels                 []LabelEntry            `mapstructure:"labels"`
	GuestAccelerators      []GuestAcceleratorEntry `mapstructure:"guestAccelerators"`
	MinNodeCpus            int64                   `mapstructure:"minNodeCpus"`
	NodeAffinities         []NodeAffinityEntry     `mapstructure:"nodeAffinities"`
	ResourcePolicies       []string                `mapstructure:"resourcePolicies"`
	EnableDisplayDevice    bool                    `mapstructure:"enableDisplayDevice"`
	EnableSerialPortAccess bool                    `mapstructure:"enableSerialPortAccess"`
	SecurityConfig         `mapstructure:",squash"`
	IdentityConfig         `mapstructure:",squash"`
	NetworkingConfig       `mapstructure:",squash"`
	OSAndStorageConfig     `mapstructure:",squash"`
}
