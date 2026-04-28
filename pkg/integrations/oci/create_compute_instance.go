package oci

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ComputeInstancePayloadType = "oci.computeInstanceCreated"
	instanceStateRunning       = "RUNNING"
	instanceStateTerminated    = "TERMINATED"
	instanceStateTerminating   = "TERMINATING"
	createInstancePollInterval = 10 * time.Second
	maxPollErrors              = 10
	maxPollAttempts            = 180 // 30 minutes at 10s interval
)

type CreateComputeInstance struct{}

type CreateComputeInstanceSpec struct {
	CompartmentID               string   `json:"compartmentId" mapstructure:"compartmentId"`
	AvailabilityDomain          string   `json:"availabilityDomain" mapstructure:"availabilityDomain"`
	DisplayName                 string   `json:"displayName" mapstructure:"displayName"`
	ImageOs                     string   `json:"imageOs" mapstructure:"imageOs"`
	Shape                       string   `json:"shape" mapstructure:"shape"`
	ImageID                     string   `json:"imageId" mapstructure:"imageId"`
	SubnetID                    string   `json:"subnetId" mapstructure:"subnetId"`
	SSHPublicKey                string   `json:"sshPublicKey" mapstructure:"sshPublicKey"`
	OCPUs                       *float64 `json:"ocpus" mapstructure:"ocpus"`
	MemoryInGBs                 *float64 `json:"memoryInGBs" mapstructure:"memoryInGBs"`
	EnableShieldedInstance      bool     `json:"enableShieldedInstance" mapstructure:"enableShieldedInstance"`
	EnableConfidentialComputing bool     `json:"enableConfidentialComputing" mapstructure:"enableConfidentialComputing"`
	BootVolumeSizeGB            *float64 `json:"bootVolumeSizeGB" mapstructure:"bootVolumeSizeGB"`
	BootVolumeVpusPerGB         *float64 `json:"bootVolumeVpusPerGB" mapstructure:"bootVolumeVpusPerGB"`
	AttachBlockVolume           bool     `json:"attachBlockVolume" mapstructure:"attachBlockVolume"`
	BlockVolumeID               string   `json:"blockVolumeId" mapstructure:"blockVolumeId"`
}

type CreateInstanceExecutionMetadata struct {
	InstanceID              string `json:"instanceId" mapstructure:"instanceId"`
	CompartmentID           string `json:"compartmentId" mapstructure:"compartmentId"`
	BlockVolumeID           string `json:"blockVolumeId" mapstructure:"blockVolumeId"`
	BlockVolumeAttachmentID string `json:"blockVolumeAttachmentId" mapstructure:"blockVolumeAttachmentId"`
	PollErrors              int    `json:"pollErrors" mapstructure:"pollErrors"`
	PollAttempts            int    `json:"pollAttempts" mapstructure:"pollAttempts"`
	StartedAt               string `json:"startedAt" mapstructure:"startedAt"`
}

func (c *CreateComputeInstance) Name() string {
	return "oci.createComputeInstance"
}

func (c *CreateComputeInstance) Label() string {
	return "Create Compute Instance"
}

func (c *CreateComputeInstance) Description() string {
	return "Provision a new OCI Compute instance and wait for it to reach RUNNING state"
}

func (c *CreateComputeInstance) Documentation() string {
	return `The Create Compute Instance component provisions a new Oracle Cloud Infrastructure Compute instance and waits until it reaches **RUNNING** state before emitting.

## Use Cases

- **Environment provisioning**: Spin up instances as part of deployment or testing workflows
- **On-demand compute**: Launch instances when triggered by events in other systems
- **Auto-scaling workflows**: Create additional capacity in response to metrics or alerts

## Configuration

- **Compartment OCID**: The compartment where the instance will be created
- **Availability Domain**: The OCI availability domain (e.g. ` + "`Uocm:PHX-AD-1`" + `)
- **Display Name**: Human-readable name for the instance
- **Shape**: Compute shape (e.g. ` + "`VM.Standard.E4.Flex`" + `, ` + "`VM.Standard2.1`" + `)
- **Image OCID**: OCID of the platform or custom image to boot from
- **Subnet OCID**: OCID of the subnet where the primary VNIC will be placed
- **SSH Public Key**: Optional SSH public key added to ` + "`~/.ssh/authorized_keys`" + ` on the instance
- **OCPUs / Memory**: For flex shapes, the number of OCPUs and memory in GB

## Output

Emits the created instance details on the default output channel, including:
- ` + "`instanceId`" + ` — instance OCID
- ` + "`displayName`" + ` — instance display name
- ` + "`lifecycleState`" + ` — should be ` + "`RUNNING`" + `
- ` + "`publicIp`" + ` — public IP address (if assigned)
- ` + "`privateIp`" + ` — primary private IP address
- ` + "`shape`" + ` — the instance shape
- ` + "`availabilityDomain`" + ` — the availability domain
- ` + "`compartmentId`" + ` — the compartment OCID
- ` + "`region`" + ` — the region
- ` + "`timeCreated`" + ` — ISO-8601 creation timestamp
`
}

func (c *CreateComputeInstance) Icon() string {
	return "oci"
}

func (c *CreateComputeInstance) Color() string {
	return "red"
}

func (c *CreateComputeInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateComputeInstance) ExampleOutput() map[string]any {
	return exampleOutputCreateComputeInstance()
}

func (c *CreateComputeInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartmentId",
			Label:       "Compartment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The compartment where the instance will be created",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCompartment,
				},
			},
		},
		{
			Name:        "availabilityDomain",
			Label:       "Availability Domain",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The availability domain where the instance will be placed",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeAvailabilityDomain,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartmentId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Human-readable name for the instance",
			Placeholder: "my-instance",
		},
		{
			Name:        "shape",
			Label:       "Shape",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Compute shape for the instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeShape,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartmentId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "ocpus",
			Label:       "OCPUs",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Number of OCPUs (required for flex shapes)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { v := 1; return &v }(),
				},
			},
		},
		{
			Name:        "memoryInGBs",
			Label:       "Memory (GB)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Memory in GB (required for flex shapes)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { v := 1; return &v }(),
				},
			},
		},
		{
			Name:        "imageOs",
			Label:       "Image OS",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Operating system family for the boot image",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Oracle Linux", Value: "Oracle Linux"},
						{Label: "Ubuntu", Value: "Canonical Ubuntu"},
						{Label: "Red Hat", Value: "Red Hat Enterprise Linux"},
						{Label: "CentOS", Value: "CentOS"},
						{Label: "AlmaLinux", Value: "AlmaLinux"},
						{Label: "Rocky Linux", Value: "Rocky Linux"},
						{Label: "Windows", Value: "Windows"},
					},
				},
			},
		},
		{
			Name:        "imageId",
			Label:       "Image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "OS image to boot from",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeImage,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartmentId",
							},
						},
						{
							Name: "imageOs",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "imageOs",
							},
						},
					},
				},
			},
		},
		{
			Name:        "subnetId",
			Label:       "Subnet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Subnet for the primary VNIC",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeSubnet,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartmentId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "sshPublicKey",
			Label:       "SSH Public Key",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "SSH public key to add to the instance for remote access",
			Placeholder: "ssh-rsa AAAA…",
		},
		{
			Name:        "bootVolumeSizeGB",
			Label:       "Boot Volume Size (GB)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Custom boot volume size in GB. Defaults to the image minimum if not set.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { v := 50; return &v }(),
				},
			},
		},
		{
			Name:        "bootVolumeVpusPerGB",
			Label:       "Boot Volume Performance (VPUs/GB)",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Boot volume performance tier",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Lower Cost (0 VPUs/GB)", Value: "0"},
						{Label: "Balanced (10 VPUs/GB)", Value: "10"},
						{Label: "Higher Performance (20 VPUs/GB)", Value: "20"},
						{Label: "Ultra High Performance (30 VPUs/GB)", Value: "30"},
					},
				},
			},
		},
		{
			Name:        "attachBlockVolume",
			Label:       "Attach Block Volume",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Default:     false,
			Description: "Attach an existing block volume to the instance after launch",
		},
		{
			Name:        "blockVolumeId",
			Label:       "Block Volume",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Existing block volume to attach",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "attachBlockVolume", Values: []string{"true"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "attachBlockVolume", Values: []string{"true"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeBlockVolume,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartmentId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "enableShieldedInstance",
			Label:       "Enable Shielded Instance",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Default:     false,
			Description: "Enables Secure Boot, vTPM, and Measured Boot for hardware-based instance integrity verification",
		},
		{
			Name:        "enableConfidentialComputing",
			Label:       "Enable Confidential Computing",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Default:     false,
			Description: "Encrypts data in-use by isolating the instance in a hardware-protected enclave (requires a supported shape)",
		},
	}
}

func (c *CreateComputeInstance) Setup(ctx core.SetupContext) error {
	spec := CreateComputeInstanceSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.CompartmentID) == "" {
		return errors.New("compartmentId is required")
	}
	if strings.TrimSpace(spec.AvailabilityDomain) == "" {
		return errors.New("availabilityDomain is required")
	}
	if strings.TrimSpace(spec.Shape) == "" {
		return errors.New("shape is required")
	}
	if strings.TrimSpace(spec.ImageID) == "" {
		return errors.New("imageId is required")
	}
	if strings.TrimSpace(spec.SubnetID) == "" {
		return errors.New("subnetId is required")
	}
	if strings.TrimSpace(spec.ImageOs) == "" {
		return errors.New("imageOs is required")
	}

	return nil
}

func (c *CreateComputeInstance) Execute(ctx core.ExecutionContext) error {
	spec := CreateComputeInstanceSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	req := LaunchInstanceRequest{
		CompartmentID:      spec.CompartmentID,
		AvailabilityDomain: spec.AvailabilityDomain,
		DisplayName:        spec.DisplayName,
		Shape:              spec.Shape,
		SourceDetails: InstanceSourceDetails{
			SourceType:          "image",
			ImageID:             spec.ImageID,
			BootVolumeSizeInGBs: spec.BootVolumeSizeGB,
			BootVolumeVpusPerGB: spec.BootVolumeVpusPerGB,
		},
		CreateVnicDetails: &CreateVnicDetails{
			SubnetID: spec.SubnetID,
		},
	}

	if strings.TrimSpace(spec.SSHPublicKey) != "" {
		req.Metadata = map[string]string{
			"ssh_authorized_keys": spec.SSHPublicKey,
		}
	}

	if spec.OCPUs != nil || spec.MemoryInGBs != nil {
		req.ShapeConfig = &InstanceShapeConfig{
			OCPUs:       spec.OCPUs,
			MemoryInGBs: spec.MemoryInGBs,
		}
	}

	if spec.EnableShieldedInstance {
		req.ShieldedInstanceConfig = &ShieldedInstanceConfig{
			IsSecureBootEnabled:            true,
			IsMeasuredBootEnabled:          true,
			IsTrustedPlatformModuleEnabled: true,
		}
	}

	if spec.EnableConfidentialComputing {
		req.ConfidentialInstanceOptions = &ConfidentialInstanceOptions{
			IsEnabled: true,
		}
	}

	instance, err := client.LaunchInstance(req)
	if err != nil {
		return fmt.Errorf("failed to launch instance: %w", err)
	}

	if err := ctx.Metadata.Set(CreateInstanceExecutionMetadata{
		InstanceID:    instance.ID,
		CompartmentID: spec.CompartmentID,
		BlockVolumeID: spec.BlockVolumeID,
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createInstancePollInterval)
}

func (c *CreateComputeInstance) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "poll", Type: core.HookTypeInternal},
	}
}

func (c *CreateComputeInstance) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == "poll" {
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *CreateComputeInstance) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata CreateInstanceExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}
	if metadata.InstanceID == "" {
		return fmt.Errorf("poll metadata is missing instanceId: execution state may be corrupted")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	instance, err := client.GetInstance(metadata.InstanceID)
	if err != nil {
		metadata.PollErrors++
		ctx.Logger.Warnf("failed to get instance %s (attempt %d/%d): %v", metadata.InstanceID, metadata.PollErrors, maxPollErrors, err)
		if metadata.PollErrors >= maxPollErrors {
			return fmt.Errorf("giving up polling instance %s after %d consecutive errors: %w", metadata.InstanceID, maxPollErrors, err)
		}
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createInstancePollInterval)
	}
	metadata.PollErrors = 0
	metadata.PollAttempts++
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	switch instance.LifecycleState {
	case instanceStateRunning:
		return c.emitInstance(ctx, client, instance, metadata)
	case instanceStateTerminated, instanceStateTerminating:
		return fmt.Errorf("instance %s entered state %s unexpectedly", instance.ID, instance.LifecycleState)
	default:
		if metadata.PollAttempts >= maxPollAttempts {
			return fmt.Errorf("timed out waiting for instance %s to reach RUNNING after %d poll attempts (state: %s)", instance.ID, metadata.PollAttempts, instance.LifecycleState)
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createInstancePollInterval)
	}
}

func (c *CreateComputeInstance) emitInstance(ctx core.ActionHookContext, client *Client, instance *Instance, metadata CreateInstanceExecutionMetadata) error {
	payload := instanceToMap(instance)

	c.enrichWithVNICIPs(ctx, client, instance, payload)

	if err := c.ensureBlockVolumeAttached(ctx, client, instance, &metadata, payload); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, ComputeInstancePayloadType, []any{payload})
}

// enrichWithVNICIPs looks up the primary VNIC for the instance and adds publicIp / privateIp
// to payload. Errors are logged as warnings so a transient VNIC-lookup failure does not block
// the execution from completing.
func (c *CreateComputeInstance) enrichWithVNICIPs(ctx core.ActionHookContext, client *Client, instance *Instance, payload map[string]any) {
	attachments, err := client.ListVNICAttachments(instance.CompartmentID, instance.ID)
	if err != nil {
		ctx.Logger.Warnf("failed to list VNIC attachments for instance %s: %v", instance.ID, err)
		return
	}

	for _, att := range attachments {
		if att.LifecycleState != "ATTACHED" || att.VNICID == "" {
			continue
		}

		vnic, err := client.GetVNIC(att.VNICID)
		if err != nil {
			ctx.Logger.Warnf("failed to get VNIC %s for instance %s: %v", att.VNICID, instance.ID, err)
			return
		}

		payload["publicIp"] = vnic.PublicIP
		payload["privateIp"] = vnic.PrivateIP
		return
	}
}

// ensureBlockVolumeAttached attaches the configured block volume (if any) to the instance and
// records the resulting attachment ID in metadata before returning, making the step idempotent
// across retries.
func (c *CreateComputeInstance) ensureBlockVolumeAttached(ctx core.ActionHookContext, client *Client, instance *Instance, metadata *CreateInstanceExecutionMetadata, payload map[string]any) error {
	if metadata.BlockVolumeID == "" {
		return nil
	}

	attachmentID := metadata.BlockVolumeAttachmentID
	if attachmentID == "" {
		attachment, err := client.AttachVolume(instance.ID, metadata.BlockVolumeID)
		if err != nil {
			return fmt.Errorf("failed to attach block volume %q to instance %q: %w", metadata.BlockVolumeID, instance.ID, err)
		}
		attachmentID = attachment.ID
		metadata.BlockVolumeAttachmentID = attachmentID
		if err := ctx.Metadata.Set(*metadata); err != nil {
			return fmt.Errorf("failed to persist block volume attachment ID: %w", err)
		}
	}

	payload["blockVolumeAttachmentId"] = attachmentID
	payload["blockVolumeId"] = metadata.BlockVolumeID
	return nil
}

func instanceToMap(instance *Instance) map[string]any {
	return map[string]any{
		"instanceId":         instance.ID,
		"displayName":        instance.DisplayName,
		"lifecycleState":     instance.LifecycleState,
		"shape":              instance.Shape,
		"availabilityDomain": instance.AvailabilityDomain,
		"compartmentId":      instance.CompartmentID,
		"region":             instance.Region,
		"timeCreated":        instance.TimeCreated,
	}
}

func (c *CreateComputeInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateComputeInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateComputeInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateComputeInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}
