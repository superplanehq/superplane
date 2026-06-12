package ec2

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	defaultRootVolumeSizeGiB  = 8
	defaultRootVolumeType     = "gp3"
	securityGroupModeCreate   = "create"
	securityGroupModeExisting = "existing"
)

var rootVolumeTypeOptions = []configuration.FieldOption{
	{Label: "General Purpose SSD (gp3)", Value: "gp3"},
	{Label: "General Purpose SSD (gp2)", Value: "gp2"},
	{Label: "Provisioned IOPS SSD (io1)", Value: "io1"},
	{Label: "Provisioned IOPS SSD (io2)", Value: "io2"},
	{Label: "Magnetic (standard)", Value: "standard"},
}

type CreateInstance struct{}

type CreateInstanceConfiguration struct {
	Name                     string `json:"name" mapstructure:"name"`
	Region                   string `json:"region" mapstructure:"region"`
	ImageOS                  string `json:"imageOs" mapstructure:"imageOs"`
	ImageID                  string `json:"image" mapstructure:"image"`
	InstanceType             string `json:"instanceType" mapstructure:"instanceType"`
	SubnetID                 string `json:"subnet" mapstructure:"subnet"`
	SecurityGroupMode        string `json:"securityGroupMode" mapstructure:"securityGroupMode"`
	SecurityGroupID          string `json:"securityGroup" mapstructure:"securityGroup"`
	AllowSSHFromInternet     bool   `json:"allowSshFromInternet" mapstructure:"allowSshFromInternet"`
	AllowHTTPFromInternet    bool   `json:"allowHttpFromInternet" mapstructure:"allowHttpFromInternet"`
	AllowHTTPSFromInternet   bool   `json:"allowHttpsFromInternet" mapstructure:"allowHttpsFromInternet"`
	KeyName                  string `json:"keyName" mapstructure:"keyName"`
	UserData                 string `json:"userData" mapstructure:"userData"`
	AssociatePublicIPAddress bool   `json:"associatePublicIpAddress" mapstructure:"associatePublicIpAddress"`
	ConfigureRootVolume      bool   `json:"configureRootVolume" mapstructure:"configureRootVolume"`
	VolumeSizeGiB            int    `json:"volumeSizeGiB" mapstructure:"volumeSizeGiB"`
	VolumeType               string `json:"volumeType" mapstructure:"volumeType"`
	VolumeIops               int    `json:"volumeIops" mapstructure:"volumeIops"`
}

type CreateInstanceNodeMetadata struct {
	Region            string `json:"region" mapstructure:"region"`
	Name              string `json:"name,omitempty" mapstructure:"name"`
	ImageOS           string `json:"imageOs,omitempty" mapstructure:"imageOs"`
	ImageOSLabel      string `json:"imageOsLabel,omitempty" mapstructure:"imageOsLabel"`
	InstanceType      string `json:"instanceType,omitempty" mapstructure:"instanceType"`
	ImageID           string `json:"image,omitempty" mapstructure:"image"`
	SubnetID          string `json:"subnet,omitempty" mapstructure:"subnet"`
	SecurityGroupID   string `json:"securityGroup,omitempty" mapstructure:"securityGroup"`
	SecurityGroupMode string `json:"securityGroupMode,omitempty" mapstructure:"securityGroupMode"`
	SubnetName        string `json:"subnetName,omitempty" mapstructure:"subnetName"`
	SecurityGroupName string `json:"securityGroupName,omitempty" mapstructure:"securityGroupName"`
	ImageName         string `json:"imageName,omitempty" mapstructure:"imageName"`
}

type CreateInstanceExecutionMetadata struct {
	InstanceID   string `json:"instanceId" mapstructure:"instanceId"`
	PollErrors   int    `json:"pollErrors" mapstructure:"pollErrors"`
	PollAttempts int    `json:"pollAttempts" mapstructure:"pollAttempts"`
}

func (c *CreateInstance) Name() string {
	return "aws.ec2.createInstance"
}

func (c *CreateInstance) Label() string {
	return "EC2 • Create Instance"
}

func (c *CreateInstance) Description() string {
	return "Launch a new EC2 instance and wait for it to reach running state"
}

func (c *CreateInstance) Documentation() string {
	return `The Create Instance component launches a new Amazon EC2 instance and waits until it reaches **running** state before emitting.

## Use Cases

- **Ephemeral compute**: Provision temporary VMs for tests, builds, or automation
- **Environment provisioning**: Launch instances as part of deployment workflows
- **On-demand capacity**: Create additional compute when triggered by events

## Configuration

- **Name**: Required value for the ` + "`Name`" + ` tag
- **Region**: AWS region where the instance will be launched
- **Operating System**: Quick Start operating system family, similar to the AWS launch wizard
- **Image**: Public AMI for the selected operating system. These are filtered to only show currently available images in the selected region for the chosen OS family
- **Instance Type**: EC2 instance type from the current generation catalog
- **Subnet**: VPC subnet for the primary network interface
- **Firewall**: Create a launch security group (like the AWS launch wizard) or use an existing one
- **Allow SSH/HTTP/HTTPS from the internet**: Ingress rules when creating a new security group
- **Configure Root Volume** (optional): Override the AMI root volume size and type
- **Key Pair** (optional): EC2 key pair for SSH access. These are filtered to show only key pairs available in the selected region
- **User Data** (optional): Shell script or cloud-init payload executed at launch
- **Associate Public IP Address**: Assign a public IPv4 address when the subnet supports it

## Output

Emits instance details on the default output channel, including:
- ` + "`instanceId`" + ` — EC2 instance ID
- ` + "`state`" + ` — should be ` + "`running`" + `
- ` + "`publicIpAddress`" + ` / ` + "`privateIpAddress`" + ` — network addresses when available
- ` + "`publicDnsName`" + ` / ` + "`privateDnsName`" + ` — DNS names when available
`
}

func (c *CreateInstance) Icon() string {
	return "aws"
}

func (c *CreateInstance) Color() string {
	return "gray"
}

func (c *CreateInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Value for the Name tag",
			Placeholder: "my-instance",
		},
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "imageOs",
			Label:       "Operating System",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Quick Start operating system family",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeImageOS,
				},
			},
		},
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Public AMI for the selected operating system. These only show currently available images in the selected region for the chosen OS family",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
				{Field: "imageOs", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.image",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
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
			Name:        "instanceType",
			Label:       "Instance Type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "EC2 instance type",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.instanceType",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:        "subnet",
			Label:       "Subnet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Subnet for the primary network interface",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.subnet",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:        "securityGroupMode",
			Label:       "Firewall",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     securityGroupModeCreate,
			Description: "Create a new security group or select an existing one",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
				{Field: "subnet", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Create security group", Value: securityGroupModeCreate},
						{Label: "Select existing security group", Value: securityGroupModeExisting},
					},
				},
			},
		},
		{
			Name:        "securityGroup",
			Label:       "Security Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Existing security group attached to the instance",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
				{Field: "subnet", Values: []string{"*"}},
				{Field: "securityGroupMode", Values: []string{securityGroupModeExisting}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "securityGroupMode", Values: []string{securityGroupModeExisting}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.securityGroup",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
						{
							Name: "subnetId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "subnet",
							},
						},
					},
				},
			},
		},
		{
			Name:        "allowSshFromInternet",
			Label:       "Allow SSH traffic from the internet",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Allow TCP port 22 from anywhere (0.0.0.0/0)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
				{Field: "subnet", Values: []string{"*"}},
				{Field: "securityGroupMode", Values: []string{securityGroupModeCreate}},
			},
		},
		{
			Name:        "allowHttpFromInternet",
			Label:       "Allow HTTP traffic from the internet",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Allow TCP port 80 from anywhere (0.0.0.0/0)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
				{Field: "subnet", Values: []string{"*"}},
				{Field: "securityGroupMode", Values: []string{securityGroupModeCreate}},
			},
		},
		{
			Name:        "allowHttpsFromInternet",
			Label:       "Allow HTTPS traffic from the internet",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Allow TCP port 443 from anywhere (0.0.0.0/0)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
				{Field: "subnet", Values: []string{"*"}},
				{Field: "securityGroupMode", Values: []string{securityGroupModeCreate}},
			},
		},
		{
			Name:        "configureRootVolume",
			Label:       "Configure Root Volume",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Default:     false,
			Description: "Override the AMI root volume size and type",
		},
		{
			Name:        "volumeSizeGiB",
			Label:       "Volume Size (GiB)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultRootVolumeSizeGiB,
			Description: "Root volume size in GiB",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "configureRootVolume", Values: []string{"true"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "configureRootVolume", Values: []string{"true"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { value := 1; return &value }(),
				},
			},
		},
		{
			Name:        "volumeType",
			Label:       "Volume Type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     defaultRootVolumeType,
			Description: "Root volume EBS type",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "configureRootVolume", Values: []string{"true"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "configureRootVolume", Values: []string{"true"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: rootVolumeTypeOptions,
				},
			},
		},
		{
			Name:        "volumeIops",
			Label:       "Volume IOPS",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Provisioned IOPS for io1 or io2 volume types. io1 minimum 100, maximum 64000; io2 minimum 100, maximum 256000.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "configureRootVolume", Values: []string{"true"}},
				{Field: "volumeType", Values: []string{"io1", "io2"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "volumeType", Values: []string{"io1", "io2"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { value := 100; return &value }(),
				},
			},
		},
		{
			Name:        "keyName",
			Label:       "Key Pair",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "EC2 key pair for SSH access. These only show key pairs available in the selected region",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.keyPair",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:        "userData",
			Label:       "User Data",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "Shell script or cloud-init payload executed at launch",
			TypeOptions: &configuration.TypeOptions{
				Text: &configuration.TextTypeOptions{
					Language: "shell",
				},
			},
		},
		{
			Name:        "associatePublicIpAddress",
			Label:       "Associate Public IP Address",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Assign a public IPv4 address to the instance",
		},
	}
}

func (c *CreateInstance) Setup(ctx core.SetupContext) error {
	config, err := decodeCreateInstanceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	config = normalizeCreateInstanceConfiguration(config, ctx.Configuration)

	if err := validateCreateInstanceConfiguration(config); err != nil {
		return err
	}

	return ctx.Metadata.Set(resolveCreateInstanceNodeMetadata(ctx, config))
}

func (c *CreateInstance) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCreateInstanceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	config = normalizeCreateInstanceConfiguration(config, ctx.Configuration)

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)

	securityGroupID, err := client.resolveLaunchSecurityGroup(config)
	if err != nil {
		return err
	}

	runInput := RunInstancesInput{
		ImageID:                  config.ImageID,
		InstanceType:             config.InstanceType,
		SubnetID:                 config.SubnetID,
		SecurityGroupIDs:         []string{securityGroupID},
		KeyName:                  config.KeyName,
		UserData:                 config.UserData,
		Name:                     config.Name,
		AssociatePublicIPAddress: config.AssociatePublicIPAddress,
	}

	if config.ConfigureRootVolume {
		image, err := client.DescribeImage(config.ImageID)
		if err != nil {
			return fmt.Errorf("failed to describe image: %w", err)
		}

		volumeSize := config.VolumeSizeGiB
		if volumeSize < 1 {
			volumeSize = defaultRootVolumeSizeGiB
		}

		volumeType := strings.TrimSpace(config.VolumeType)
		if volumeType == "" {
			volumeType = defaultRootVolumeType
		}

		runInput.RootVolume = &RootVolumeConfig{
			DeviceName: image.RootDeviceName,
			VolumeSize: volumeSize,
			VolumeType: volumeType,
			Iops:       config.VolumeIops,
		}
	}

	output, err := client.RunInstances(runInput)
	if err != nil {
		return fmt.Errorf("failed to run instance: %w", err)
	}

	if err := ctx.Metadata.Set(CreateInstanceExecutionMetadata{InstanceID: output.InstanceID}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
}

func (c *CreateInstance) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "poll", Type: core.HookTypeInternal},
	}
}

func (c *CreateInstance) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	return c.poll(ctx)
}

func (c *CreateInstance) poll(ctx core.ActionHookContext) error {
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

	config, err := decodeCreateInstanceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	instance, err := client.DescribeInstance(metadata.InstanceID)
	if err != nil {
		metadata.PollErrors++
		ctx.Logger.Warnf("failed to describe instance %s (attempt %d/%d): %v", metadata.InstanceID, metadata.PollErrors, maxInstancePollErrors, err)
		if metadata.PollErrors >= maxInstancePollErrors {
			return fmt.Errorf("giving up polling instance %s after %d consecutive errors: %w", metadata.InstanceID, maxInstancePollErrors, err)
		}
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
	}

	metadata.PollErrors = 0
	metadata.PollAttempts++
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	switch instance.State {
	case InstanceStateRunning:
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateInstancePayloadType, []any{instanceDetailsToMap(instance)})
	case InstanceStateTerminated, InstanceStateStopped, InstanceStateStopping:
		if ctx.ExecutionState.IsFinished() {
			return nil
		}
		return fmt.Errorf("instance %s entered state %q and will not reach running without intervention", instance.InstanceID, instance.State)
	default:
		if metadata.PollAttempts >= maxInstancePollAttempts {
			return fmt.Errorf("timed out waiting for instance %s to reach running after %d poll attempts (state: %s)", instance.InstanceID, metadata.PollAttempts, instance.State)
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
	}
}

func (c *CreateInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateInstance) Cancel(ctx core.ExecutionContext) error {
	var metadata CreateInstanceExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.InstanceID == "" {
		return nil
	}

	config, err := decodeCreateInstanceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	if _, err := client.TerminateInstances(metadata.InstanceID); err != nil && !IsInstanceNotFound(err) {
		return fmt.Errorf("failed to terminate instance %s during cancel: %w", metadata.InstanceID, err)
	}

	return nil
}

func (c *CreateInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func decodeCreateInstanceConfiguration(raw any) (CreateInstanceConfiguration, error) {
	config := CreateInstanceConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return CreateInstanceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	return config, nil
}

func normalizeCreateInstanceConfiguration(config CreateInstanceConfiguration, raw any) CreateInstanceConfiguration {
	if securityGroupMode(config) != securityGroupModeCreate {
		return config
	}

	if rawMap, ok := raw.(map[string]any); ok {
		if _, exists := rawMap["allowSshFromInternet"]; !exists {
			config.AllowSSHFromInternet = true
		}
	} else {
		config.AllowSSHFromInternet = true
	}

	return config
}

func validateCreateInstanceConfiguration(config CreateInstanceConfiguration) error {
	if _, err := requireName(config.Name); err != nil {
		return err
	}
	if _, err := requireRegion(config.Region); err != nil {
		return err
	}
	if _, err := requireImageOS(config.ImageOS); err != nil {
		return err
	}
	if _, err := requireImageID(config.ImageID); err != nil {
		return err
	}
	if _, err := requireInstanceType(config.InstanceType); err != nil {
		return err
	}
	if _, err := requireSubnetID(config.SubnetID); err != nil {
		return err
	}
	if securityGroupMode(config) == securityGroupModeExisting {
		if _, err := requireSecurityGroupID(config.SecurityGroupID); err != nil {
			return err
		}
	}
	if config.ConfigureRootVolume {
		if config.VolumeSizeGiB < 1 {
			return fmt.Errorf("volume size must be at least 1 GiB")
		}
		volumeType := strings.TrimSpace(config.VolumeType)
		if volumeType == "" {
			return fmt.Errorf("volume type is required when configuring the root volume")
		}
		validBootVolumeTypes := []string{"gp3", "gp2", "io1", "io2", "standard"}
		if !slices.Contains(validBootVolumeTypes, volumeType) {
			return fmt.Errorf("volume type %q cannot be used as a boot volume; valid types are: gp3, gp2, io1, io2, standard", volumeType)
		}
		if (volumeType == "io1" || volumeType == "io2") && config.VolumeIops < 100 {
			return fmt.Errorf("IOPS must be at least 100 for %s volume type", volumeType)
		}
	}
	return nil
}

func resolveCreateInstanceNodeMetadata(ctx core.SetupContext, config CreateInstanceConfiguration) CreateInstanceNodeMetadata {
	metadata := CreateInstanceNodeMetadata{
		Region:            strings.TrimSpace(config.Region),
		Name:              strings.TrimSpace(config.Name),
		ImageOS:           strings.TrimSpace(config.ImageOS),
		ImageOSLabel:      imageOSLabel(config.ImageOS),
		InstanceType:      strings.TrimSpace(config.InstanceType),
		ImageID:           strings.TrimSpace(config.ImageID),
		SubnetID:          strings.TrimSpace(config.SubnetID),
		SecurityGroupID:   strings.TrimSpace(config.SecurityGroupID),
		SecurityGroupMode: securityGroupMode(config),
	}

	if ctx.HTTP == nil || ctx.Integration == nil {
		return metadata
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil || metadata.Region == "" {
		return metadata
	}

	client := NewClient(ctx.HTTP, creds, metadata.Region)

	if metadata.ImageID != "" {
		if image, err := client.DescribeImage(metadata.ImageID); err == nil {
			metadata.ImageName = image.Name
		}
	}

	if metadata.SubnetID != "" {
		if subnet, err := client.DescribeSubnet(metadata.SubnetID); err == nil {
			metadata.SubnetName = subnetResourceName(*subnet)
		}
	}

	if metadata.SecurityGroupID != "" {
		if groups, err := client.ListSecurityGroups(); err == nil {
			for _, group := range groups {
				if group.GroupID == metadata.SecurityGroupID {
					metadata.SecurityGroupName = securityGroupResourceName(group)
					break
				}
			}
		}
	}

	return metadata
}

type SecurityGroupIngressRule struct {
	Protocol string
	FromPort int
	ToPort   int
	CidrIPv4 string
}

func securityGroupMode(config CreateInstanceConfiguration) string {
	mode := strings.TrimSpace(config.SecurityGroupMode)
	if mode == securityGroupModeCreate || mode == securityGroupModeExisting {
		return mode
	}

	if strings.TrimSpace(config.SecurityGroupID) != "" {
		return securityGroupModeExisting
	}

	return securityGroupModeCreate
}

func firewallIngressRules(config CreateInstanceConfiguration) []SecurityGroupIngressRule {
	rules := []SecurityGroupIngressRule{}

	if config.AllowSSHFromInternet {
		rules = append(rules, SecurityGroupIngressRule{
			Protocol: "tcp",
			FromPort: 22,
			ToPort:   22,
			CidrIPv4: "0.0.0.0/0",
		})
	}

	if config.AllowHTTPFromInternet {
		rules = append(rules, SecurityGroupIngressRule{
			Protocol: "tcp",
			FromPort: 80,
			ToPort:   80,
			CidrIPv4: "0.0.0.0/0",
		})
	}

	if config.AllowHTTPSFromInternet {
		rules = append(rules, SecurityGroupIngressRule{
			Protocol: "tcp",
			FromPort: 443,
			ToPort:   443,
			CidrIPv4: "0.0.0.0/0",
		})
	}

	return rules
}

func (c *Client) resolveLaunchSecurityGroup(config CreateInstanceConfiguration) (string, error) {
	rules := firewallIngressRules(config)

	switch securityGroupMode(config) {
	case securityGroupModeExisting:
		securityGroupID, err := requireSecurityGroupID(config.SecurityGroupID)
		if err != nil {
			return "", err
		}
		return securityGroupID, nil
	case securityGroupModeCreate:
		subnet, err := c.DescribeSubnet(config.SubnetID)
		if err != nil {
			return "", fmt.Errorf("failed to describe subnet: %w", err)
		}

		groupName := launchSecurityGroupName(config.Name)
		description := fmt.Sprintf("SuperPlane launch security group for %s", strings.TrimSpace(config.Name))
		securityGroupID, err := c.createLaunchSecurityGroup(groupName, description, subnet.VpcID)
		if err != nil {
			return "", fmt.Errorf("failed to create security group: %w", err)
		}

		if len(rules) > 0 {
			if err := c.EnsureSecurityGroupIngressRules(securityGroupID, rules); err != nil {
				return "", fmt.Errorf("failed to authorize security group rules: %w", err)
			}
		}

		return securityGroupID, nil
	default:
		return "", fmt.Errorf("unsupported security group mode: %s", config.SecurityGroupMode)
	}
}

var securityGroupNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9-]+`)

func launchSecurityGroupName(instanceName string) string {
	sanitized := strings.Trim(securityGroupNameSanitizer.ReplaceAllString(strings.TrimSpace(instanceName), "-"), "-")
	if sanitized == "" {
		sanitized = "instance"
	}

	name := fmt.Sprintf("launch-wizard-%s", sanitized)
	if len(name) > 255 {
		return name[:255]
	}

	return name
}

func (c *Client) createLaunchSecurityGroup(groupName, description, vpcID string) (string, error) {
	groupID, err := c.CreateSecurityGroup(groupName, description, vpcID)
	if err == nil {
		return groupID, nil
	}

	if !IsSecurityGroupDuplicate(err) {
		return "", err
	}

	suffix := strings.ReplaceAll(strings.Split(uuid.NewString(), "-")[0], "_", "")
	fallbackName := fmt.Sprintf("%s-%s", groupName, suffix)
	if len(fallbackName) > 255 {
		fallbackName = fallbackName[:255]
	}

	return c.CreateSecurityGroup(fallbackName, description, vpcID)
}

func IsSecurityGroupDuplicate(err error) bool {
	var awsErr *common.Error
	return errors.As(err, &awsErr) && strings.TrimSpace(awsErr.Code) == "InvalidGroup.Duplicate"
}

func IsSecurityGroupRuleDuplicate(err error) bool {
	var awsErr *common.Error
	return errors.As(err, &awsErr) && strings.TrimSpace(awsErr.Code) == "InvalidPermission.Duplicate"
}
