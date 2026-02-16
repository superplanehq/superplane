package createvm

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

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
	groups := [][]configuration.Field{
		CreateVMMachineConfigFields(),
		CreateVMOSAndStorageConfigFields(),
		CreateVMSecurityConfigFields(),
		CreateVMIdentityConfigFields(),
		CreateVMNetworkingConfigFields(),
		CreateVMManagementConfigFields(),
		CreateVMAdvancedConfigFields(),
	}
	var fields []configuration.Field
	for _, g := range groups {
		fields = append(fields, g...)
	}
	return fields
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
	Project                string                  `mapstructure:"project"`
	Region                 string                  `mapstructure:"region"`
	Zone                   string                  `mapstructure:"zone"`
	MachineFamily          string                  `mapstructure:"machineFamily"`
	MachineType            string                  `mapstructure:"machineType"`
	ProvisioningModel      string                  `mapstructure:"provisioningModel"`
	AutomaticRestart       bool                    `mapstructure:"automaticRestart"`
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
