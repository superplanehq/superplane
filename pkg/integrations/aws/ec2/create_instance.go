package ec2

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
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
	defaultWaitTimeoutSeconds = 300
	maxWaitTimeoutSeconds     = 1800
	securityGroupModeCreate   = "create"
	securityGroupModeExisting = "existing"
	createInstanceCreated     = "created"
	createInstanceFailed      = "failed"
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
	Name                         string                     `json:"name" mapstructure:"name"`
	Region                       string                     `json:"region" mapstructure:"region"`
	ImageOS                      string                     `json:"imageOs" mapstructure:"imageOs"`
	ImageID                      string                     `json:"image" mapstructure:"image"`
	InstanceType                 string                     `json:"instanceType" mapstructure:"instanceType"`
	SubnetID                     string                     `json:"subnet" mapstructure:"subnet"`
	SecurityGroupMode            string                     `json:"securityGroupMode" mapstructure:"securityGroupMode"`
	SecurityGroupID              string                     `json:"-" mapstructure:"-"`
	SecurityGroupIDs             []string                   `json:"securityGroup" mapstructure:"securityGroup"`
	IAMInstanceProfile           string                     `json:"iamInstanceProfile" mapstructure:"iamInstanceProfile"`
	AllowSSHFromInternet         bool                       `json:"allowSshFromInternet" mapstructure:"allowSshFromInternet"`
	AllowHTTPFromInternet        bool                       `json:"allowHttpFromInternet" mapstructure:"allowHttpFromInternet"`
	AllowHTTPSFromInternet       bool                       `json:"allowHttpsFromInternet" mapstructure:"allowHttpsFromInternet"`
	KeyName                      string                     `json:"keyName" mapstructure:"keyName"`
	UserData                     string                     `json:"userData" mapstructure:"userData"`
	AssociatePublicIPAddress     bool                       `json:"associatePublicIpAddress" mapstructure:"associatePublicIpAddress"`
	ConfigureRootVolume          bool                       `json:"configureRootVolume" mapstructure:"configureRootVolume"`
	RootDeviceName               string                     `json:"rootDeviceName" mapstructure:"rootDeviceName"`
	VolumeSizeGiB                int                        `json:"volumeSizeGiB" mapstructure:"volumeSizeGiB"`
	VolumeType                   string                     `json:"volumeType" mapstructure:"volumeType"`
	VolumeIops                   int                        `json:"volumeIops" mapstructure:"volumeIops"`
	RootDeleteOnTermination      bool                       `json:"rootDeleteOnTermination" mapstructure:"rootDeleteOnTermination"`
	RootEncrypted                bool                       `json:"rootEncrypted" mapstructure:"rootEncrypted"`
	RootKmsKeyID                 string                     `json:"rootKmsKeyId" mapstructure:"rootKmsKeyId"`
	Tags                         []common.Tag               `json:"tags" mapstructure:"tags"`
	AdditionalBlockDevices       []BlockDeviceConfiguration `json:"additionalBlockDevices" mapstructure:"additionalBlockDevices"`
	WaitForRunningTimeoutSeconds int                        `json:"waitForRunningTimeoutSeconds" mapstructure:"waitForRunningTimeoutSeconds"`
	WaitForStatusChecks          bool                       `json:"waitForStatusChecks" mapstructure:"waitForStatusChecks"`
}

type BlockDeviceConfiguration struct {
	DeviceName          string `json:"deviceName" mapstructure:"deviceName"`
	VolumeSizeGiB       int    `json:"volumeSizeGiB" mapstructure:"volumeSizeGiB"`
	VolumeType          string `json:"volumeType" mapstructure:"volumeType"`
	VolumeIops          int    `json:"volumeIops" mapstructure:"volumeIops"`
	DeleteOnTermination bool   `json:"deleteOnTermination" mapstructure:"deleteOnTermination"`
	Encrypted           bool   `json:"encrypted" mapstructure:"encrypted"`
	KmsKeyID            string `json:"kmsKeyId" mapstructure:"kmsKeyId"`
}

type CreateInstanceNodeMetadata struct {
	Region            string   `json:"region" mapstructure:"region"`
	Name              string   `json:"name,omitempty" mapstructure:"name"`
	ImageOS           string   `json:"imageOs,omitempty" mapstructure:"imageOs"`
	ImageOSLabel      string   `json:"imageOsLabel,omitempty" mapstructure:"imageOsLabel"`
	InstanceType      string   `json:"instanceType,omitempty" mapstructure:"instanceType"`
	ImageID           string   `json:"image,omitempty" mapstructure:"image"`
	SubnetID          string   `json:"subnet,omitempty" mapstructure:"subnet"`
	SecurityGroupID   string   `json:"securityGroup,omitempty" mapstructure:"securityGroup"`
	SecurityGroupIDs  []string `json:"securityGroups,omitempty" mapstructure:"securityGroups"`
	SecurityGroupMode string   `json:"securityGroupMode,omitempty" mapstructure:"securityGroupMode"`
	SubnetName        string   `json:"subnetName,omitempty" mapstructure:"subnetName"`
	SecurityGroupName string   `json:"securityGroupName,omitempty" mapstructure:"securityGroupName"`
	ImageName         string   `json:"imageName,omitempty" mapstructure:"imageName"`
}

type CreateInstanceExecutionMetadata struct {
	InstanceID        string `json:"instanceId" mapstructure:"instanceId"`
	PollErrors        int    `json:"pollErrors" mapstructure:"pollErrors"`
	PollAttempts      int    `json:"pollAttempts" mapstructure:"pollAttempts"`
	LastObservedState string `json:"lastObservedState,omitempty" mapstructure:"lastObservedState"`
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
- **Firewall**: Create a launch security group (like the AWS launch wizard) or use one or more existing security groups
- **IAM Instance Profile** (optional): IAM instance profile name or ARN attached to the launched instance
- **Allow SSH/HTTP/HTTPS from the internet**: Ingress rules when creating a new security group
- **Configure Root Volume** (optional): Override the AMI root volume device name, size, type, IOPS, delete-on-termination, and encryption settings
- **Additional Block Devices** (optional): Extra EBS volumes attached at launch
- **Key Pair** (optional): EC2 key pair for SSH access. These are filtered to show only key pairs available in the selected region
- **User Data** (optional): Shell script or cloud-init payload executed at launch
- **Associate Public IP Address**: Assign a public IPv4 address when the subnet supports it
- **Tags** (optional): Additional tags applied to both instance and volume resources. The ` + "`Name`" + ` tag is set from the Name field
- **Wait For Running Timeout**: Maximum time to wait for the instance to reach running state
- **Wait For Status Checks**: Also wait for EC2 instance and system status checks to pass

## Output

Emits instance details on the created output channel, including:
- ` + "`instanceId`" + ` — EC2 instance ID
- ` + "`state`" + ` — should be ` + "`running`" + `
- ` + "`publicIpAddress`" + ` / ` + "`privateIpAddress`" + ` — network addresses when available
- ` + "`publicDnsName`" + ` / ` + "`privateDnsName`" + ` — DNS names when available
- ` + "`name`" + ` — the Name tag value
- ` + "`availabilityZone`" + ` — EC2 availability zone
- ` + "`tags`" + ` — instance tags as key/value pairs

## runnerBash Migration

| ` + "`aws ec2 run-instances`" + ` flag | ` + "`aws.ec2.createInstance`" + ` field |
|---|---|
| ` + "`--image-id`" + ` | ` + "`image`" + ` |
| ` + "`--instance-type`" + ` | ` + "`instanceType`" + ` |
| ` + "`--subnet-id`" + ` | ` + "`subnet`" + ` |
| ` + "`--security-group-ids`" + ` | ` + "`securityGroup`" + ` |
| ` + "`--iam-instance-profile Name=...`" + ` | ` + "`iamInstanceProfile`" + ` |
| ` + "`--associate-public-ip-address`" + ` | ` + "`associatePublicIpAddress`" + ` |
| ` + "`--user-data file://...`" + ` | ` + "`userData`" + ` |
| ` + "`--block-device-mappings`" + ` | ` + "`configureRootVolume`" + ` and ` + "`additionalBlockDevices`" + ` |
| ` + "`--tag-specifications`" + ` | ` + "`name`" + ` and ` + "`tags`" + ` |
`
}

func (c *CreateInstance) Icon() string {
	return "aws"
}

func (c *CreateInstance) Color() string {
	return "gray"
}

func (c *CreateInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: createInstanceCreated, Label: "Created", Description: "Instance reached the requested ready state"},
		{Name: createInstanceFailed, Label: "Failed", Description: "Launch or wait failure"},
	}
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
			Label:       "Security Groups",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Existing security groups attached to the instance",
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
					Type:  "ec2.securityGroup",
					Multi: true,
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
			Name:        "iamInstanceProfile",
			Label:       "IAM Instance Profile",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "IAM instance profile attached to the launched instance",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "iam.instanceProfile",
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
			Name:        "rootDeviceName",
			Label:       "Root Device Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Device name for the root volume. Defaults to the AMI root device name.",
			Placeholder: "/dev/xvda",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "configureRootVolume", Values: []string{"true"}},
			},
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
			Name:        "rootDeleteOnTermination",
			Label:       "Delete Root Volume On Termination",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Delete the root EBS volume when the instance terminates",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "configureRootVolume", Values: []string{"true"}},
			},
		},
		{
			Name:        "rootEncrypted",
			Label:       "Encrypt Root Volume",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Encrypt the root EBS volume",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "configureRootVolume", Values: []string{"true"}},
			},
		},
		{
			Name:        "rootKmsKeyId",
			Label:       "Root Volume KMS Key ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "KMS key ID or ARN used to encrypt the root volume",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "configureRootVolume", Values: []string{"true"}},
				{Field: "rootEncrypted", Values: []string{"true"}},
			},
		},
		{
			Name:        "additionalBlockDevices",
			Label:       "Additional Block Devices",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Additional EBS volumes attached at launch",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Block Device",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: blockDeviceFieldSchema(),
					},
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
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Additional tags applied to the instance (Name is set separately via the Name field)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: tagFieldSchema(),
					},
				},
			},
		},
		{
			Name:        "waitForRunningTimeoutSeconds",
			Label:       "Wait For Running Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultWaitTimeoutSeconds,
			Description: "Maximum time to wait for the instance to reach running state",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(maxWaitTimeoutSeconds),
				},
			},
		},
		{
			Name:        "waitForStatusChecks",
			Label:       "Wait For Status Checks",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Wait until EC2 system and instance status checks pass before emitting",
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

	securityGroupIDs, err := client.resolveLaunchSecurityGroups(config)
	if err != nil {
		return emitCreateInstanceFailure(ctx.ExecutionState, createInstanceFailurePayload(err, "", ""))
	}

	runInput := RunInstancesInput{
		ImageID:                  config.ImageID,
		InstanceType:             config.InstanceType,
		SubnetID:                 config.SubnetID,
		SecurityGroupIDs:         securityGroupIDs,
		IAMInstanceProfile:       config.IAMInstanceProfile,
		KeyName:                  config.KeyName,
		UserData:                 config.UserData,
		Name:                     config.Name,
		Tags:                     mergeInstanceTags(config.Name, config.Tags),
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
			DeviceName:          firstNonEmpty(config.RootDeviceName, image.RootDeviceName),
			VolumeSize:          volumeSize,
			VolumeType:          volumeType,
			Iops:                config.VolumeIops,
			DeleteOnTermination: config.RootDeleteOnTermination,
			Encrypted:           config.RootEncrypted,
			KmsKeyID:            config.RootKmsKeyID,
		}
	}
	runInput.AdditionalBlockDevices = blockDevicesFromConfiguration(config.AdditionalBlockDevices)

	output, err := client.RunInstances(runInput)
	if err != nil {
		return emitCreateInstanceFailure(ctx.ExecutionState, createInstanceFailurePayload(fmt.Errorf("failed to run instance: %w", err), "", ""))
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
	instance, err := client.DescribeInstance(metadata.InstanceID)
	if err != nil {
		metadata.PollErrors++
		ctx.Logger.Warnf("failed to describe instance %s (attempt %d/%d): %v", metadata.InstanceID, metadata.PollErrors, maxInstancePollErrors, err)
		if metadata.PollErrors >= maxInstancePollErrors {
			return emitCreateInstanceFailure(
				ctx.ExecutionState,
				createInstanceFailurePayload(fmt.Errorf("giving up polling instance %s after %d consecutive errors: %w", metadata.InstanceID, maxInstancePollErrors, err), metadata.InstanceID, metadata.LastObservedState),
			)
		}
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
	}

	metadata.PollErrors = 0
	metadata.PollAttempts++
	metadata.LastObservedState = instance.State
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	switch instance.State {
	case InstanceStateRunning:
		if config.WaitForStatusChecks {
			status, err := client.DescribeInstanceStatus(instance.InstanceID)
			if err != nil {
				metadata.PollErrors++
				if err := ctx.Metadata.Set(metadata); err != nil {
					return err
				}
				if metadata.PollErrors >= maxInstancePollErrors {
					return emitCreateInstanceFailure(
						ctx.ExecutionState,
						createInstanceFailurePayload(fmt.Errorf("giving up waiting for status checks on instance %s after %d consecutive errors: %w", instance.InstanceID, maxInstancePollErrors, err), instance.InstanceID, instance.State),
					)
				}
				return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
			}
			if !status.OK() {
				if createInstanceTimedOut(metadata.PollAttempts, config.WaitForRunningTimeoutSeconds) {
					return emitCreateInstanceFailure(
						ctx.ExecutionState,
						createInstanceFailurePayload(fmt.Errorf("timed out waiting for status checks on instance %s after %d seconds (instance status: %s, system status: %s)", instance.InstanceID, config.WaitForRunningTimeoutSeconds, status.InstanceStatus, status.SystemStatus), instance.InstanceID, instance.State),
					)
				}
				return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
			}
		}
		return ctx.ExecutionState.Emit(createInstanceCreated, CreateInstancePayloadType, []any{instanceDetailsToMap(instance)})
	case InstanceStateTerminated, InstanceStateStopped, InstanceStateStopping:
		if ctx.ExecutionState.IsFinished() {
			return nil
		}
		return emitCreateInstanceFailure(
			ctx.ExecutionState,
			createInstanceFailurePayload(fmt.Errorf("instance %s entered state %q and will not reach running without intervention", instance.InstanceID, instance.State), instance.InstanceID, instance.State),
		)
	default:
		if createInstanceTimedOut(metadata.PollAttempts, config.WaitForRunningTimeoutSeconds) {
			return emitCreateInstanceFailure(
				ctx.ExecutionState,
				createInstanceFailurePayload(fmt.Errorf("timed out waiting for instance %s to reach running after %d seconds (state: %s)", instance.InstanceID, config.WaitForRunningTimeoutSeconds, instance.State), instance.InstanceID, instance.State),
			)
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
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &config,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return CreateInstanceConfiguration{}, fmt.Errorf("failed to build configuration decoder: %w", err)
	}
	if err := decoder.Decode(raw); err != nil {
		return CreateInstanceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	return config, nil
}

func normalizeCreateInstanceConfiguration(config CreateInstanceConfiguration, raw any) CreateInstanceConfiguration {
	config.SecurityGroupIDs = normalizeSecurityGroupIDs(config, raw)
	if len(config.SecurityGroupIDs) > 0 {
		config.SecurityGroupID = config.SecurityGroupIDs[0]
	}

	if config.WaitForRunningTimeoutSeconds <= 0 {
		config.WaitForRunningTimeoutSeconds = defaultWaitTimeoutSeconds
	}

	config.RootDeleteOnTermination = normalizeBoolDefault(raw, "rootDeleteOnTermination", true)
	for i := range config.AdditionalBlockDevices {
		config.AdditionalBlockDevices[i].DeleteOnTermination = normalizeBlockDeviceDeleteOnTermination(raw, i)
	}

	if securityGroupMode(config) != securityGroupModeCreate {
		return config
	}

	config.AllowSSHFromInternet = normalizeBoolDefault(raw, "allowSshFromInternet", true)

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
		if len(config.SecurityGroupIDs) == 0 {
			return fmt.Errorf("security group is required")
		}
		for _, securityGroupID := range config.SecurityGroupIDs {
			if _, err := requireSecurityGroupID(securityGroupID); err != nil {
				return err
			}
		}
	}
	if config.WaitForRunningTimeoutSeconds < 1 {
		return fmt.Errorf("wait timeout must be at least 1 second")
	}
	if config.WaitForRunningTimeoutSeconds > maxWaitTimeoutSeconds {
		return fmt.Errorf("wait timeout must be at most %d seconds", maxWaitTimeoutSeconds)
	}
	for _, tag := range config.Tags {
		if strings.TrimSpace(tag.Key) == "" {
			return fmt.Errorf("tag key is required")
		}
		if strings.TrimSpace(tag.Value) == "" {
			return fmt.Errorf("tag value is required")
		}
	}
	for _, device := range config.AdditionalBlockDevices {
		if strings.TrimSpace(device.DeviceName) == "" {
			return fmt.Errorf("additional block device name is required")
		}
		if device.VolumeSizeGiB < 1 {
			return fmt.Errorf("additional block device volume size must be at least 1 GiB")
		}
		if err := validateVolumeType(device.VolumeType, device.VolumeIops, "additional block device"); err != nil {
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
		if err := validateVolumeType(volumeType, config.VolumeIops, "root volume"); err != nil {
			return err
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
		SecurityGroupIDs:  config.SecurityGroupIDs,
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

	if len(metadata.SecurityGroupIDs) > 0 {
		if groups, err := client.ListSecurityGroups(); err == nil {
			for _, group := range groups {
				if group.GroupID == metadata.SecurityGroupIDs[0] {
					metadata.SecurityGroupName = securityGroupResourceName(group)
					break
				}
			}
		}
	}

	return metadata
}

func tagFieldSchema() []configuration.Field {
	return []configuration.Field{
		{Name: "key", Label: "Key", Type: configuration.FieldTypeString, Required: true},
		{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true},
	}
}

func blockDeviceFieldSchema() []configuration.Field {
	return []configuration.Field{
		{Name: "deviceName", Label: "Device Name", Type: configuration.FieldTypeString, Required: true, Placeholder: "/dev/sdf"},
		{
			Name:        "volumeSizeGiB",
			Label:       "Volume Size (GiB)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			TypeOptions: &configuration.TypeOptions{Number: &configuration.NumberTypeOptions{Min: intPtr(1)}},
		},
		{
			Name:     "volumeType",
			Label:    "Volume Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  defaultRootVolumeType,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: rootVolumeTypeOptions},
			},
		},
		{
			Name:        "volumeIops",
			Label:       "Volume IOPS",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Provisioned IOPS for io1 or io2 volume types",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "volumeType", Values: []string{"io1", "io2"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "volumeType", Values: []string{"io1", "io2"}},
			},
			TypeOptions: &configuration.TypeOptions{Number: &configuration.NumberTypeOptions{Min: intPtr(100)}},
		},
		{Name: "deleteOnTermination", Label: "Delete On Termination", Type: configuration.FieldTypeBool, Required: false, Default: true},
		{Name: "encrypted", Label: "Encrypted", Type: configuration.FieldTypeBool, Required: false, Default: false},
		{
			Name:        "kmsKeyId",
			Label:       "KMS Key ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "KMS key ID or ARN used to encrypt the volume",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "encrypted", Values: []string{"true"}},
			},
		},
	}
}

func validateVolumeType(volumeType string, iops int, label string) error {
	trimmed := strings.TrimSpace(volumeType)
	validBootVolumeTypes := []string{"gp3", "gp2", "io1", "io2", "standard"}
	if !slices.Contains(validBootVolumeTypes, trimmed) {
		return fmt.Errorf("%s volume type %q is not supported; valid types are: gp3, gp2, io1, io2, standard", label, volumeType)
	}
	if (trimmed == "io1" || trimmed == "io2") && iops < 100 {
		return fmt.Errorf("%s IOPS must be at least 100 for %s volume type", label, trimmed)
	}
	return nil
}

func normalizeSecurityGroupIDs(config CreateInstanceConfiguration, raw any) []string {
	if rawMap, ok := raw.(map[string]any); ok {
		if rawValue, exists := rawMap["securityGroup"]; exists {
			return stringsFromRawValue(rawValue)
		}
	}

	if len(config.SecurityGroupIDs) > 0 {
		return compactStrings(config.SecurityGroupIDs)
	}

	return compactStrings([]string{config.SecurityGroupID})
}

func stringsFromRawValue(value any) []string {
	switch typed := value.(type) {
	case string:
		return compactStrings([]string{typed})
	case []string:
		return compactStrings(typed)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			values = append(values, fmt.Sprint(item))
		}
		return compactStrings(values)
	default:
		return nil
	}
}

func compactStrings(values []string) []string {
	compacted := []string{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			compacted = append(compacted, trimmed)
		}
	}
	return compacted
}

func normalizeBoolDefault(raw any, key string, defaultValue bool) bool {
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return defaultValue
	}

	value, exists := rawMap[key]
	if !exists {
		return defaultValue
	}

	boolValue, ok := value.(bool)
	if ok {
		return boolValue
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(fmt.Sprint(value)))
	if err != nil {
		return defaultValue
	}
	return parsed
}

func normalizeBlockDeviceDeleteOnTermination(raw any, index int) bool {
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return true
	}

	rawDevices, ok := rawMap["additionalBlockDevices"].([]any)
	if !ok || index >= len(rawDevices) {
		return true
	}

	rawDevice, ok := rawDevices[index].(map[string]any)
	if !ok {
		return true
	}

	value, exists := rawDevice["deleteOnTermination"]
	if !exists {
		return true
	}

	boolValue, ok := value.(bool)
	if ok {
		return boolValue
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(fmt.Sprint(value)))
	if err != nil {
		return true
	}
	return parsed
}

func mergeInstanceTags(name string, tags []common.Tag) []common.Tag {
	merged := []common.Tag{{Key: "Name", Value: strings.TrimSpace(name)}}
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		if key == "" || key == "Name" {
			continue
		}
		merged = append(merged, common.Tag{Key: key, Value: strings.TrimSpace(tag.Value)})
	}
	return merged
}

func blockDevicesFromConfiguration(devices []BlockDeviceConfiguration) []BlockDeviceConfig {
	blockDevices := make([]BlockDeviceConfig, 0, len(devices))
	for _, device := range devices {
		volumeType := strings.TrimSpace(device.VolumeType)
		if volumeType == "" {
			volumeType = defaultRootVolumeType
		}
		blockDevices = append(blockDevices, BlockDeviceConfig{
			DeviceName:          device.DeviceName,
			VolumeSize:          device.VolumeSizeGiB,
			VolumeType:          volumeType,
			Iops:                device.VolumeIops,
			DeleteOnTermination: device.DeleteOnTermination,
			Encrypted:           device.Encrypted,
			KmsKeyID:            device.KmsKeyID,
		})
	}
	return blockDevices
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func createInstanceTimedOut(pollAttempts int, timeoutSeconds int) bool {
	return pollAttempts*int(instancePollInterval.Seconds()) >= timeoutSeconds
}

func createInstanceFailurePayload(err error, instanceID string, lastObservedState string) map[string]any {
	payload := map[string]any{
		"error":             err.Error(),
		"awsErrorCode":      "",
		"instanceId":        strings.TrimSpace(instanceID),
		"lastObservedState": strings.TrimSpace(lastObservedState),
	}

	var awsErr *common.Error
	if errors.As(err, &awsErr) {
		payload["awsErrorCode"] = strings.TrimSpace(awsErr.Code)
	}

	return payload
}

func emitCreateInstanceFailure(executionState core.ExecutionStateContext, payload map[string]any) error {
	if executionState == nil {
		return fmt.Errorf("%s", payload["error"])
	}
	return executionState.Emit(createInstanceFailed, CreateInstancePayloadType, []any{payload})
}

func intPtr(value int) *int {
	return &value
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

	if len(normalizeSecurityGroupIDs(config, nil)) > 0 {
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

func (c *Client) resolveLaunchSecurityGroups(config CreateInstanceConfiguration) ([]string, error) {
	rules := firewallIngressRules(config)

	switch securityGroupMode(config) {
	case securityGroupModeExisting:
		if len(config.SecurityGroupIDs) == 0 {
			return nil, fmt.Errorf("security group is required")
		}
		return config.SecurityGroupIDs, nil
	case securityGroupModeCreate:
		subnet, err := c.DescribeSubnet(config.SubnetID)
		if err != nil {
			return nil, fmt.Errorf("failed to describe subnet: %w", err)
		}

		groupName := launchSecurityGroupName(config.Name)
		description := fmt.Sprintf("SuperPlane launch security group for %s", strings.TrimSpace(config.Name))
		securityGroupID, err := c.createLaunchSecurityGroup(groupName, description, subnet.VpcID)
		if err != nil {
			return nil, fmt.Errorf("failed to create security group: %w", err)
		}

		if len(rules) > 0 {
			if err := c.EnsureSecurityGroupIngressRules(securityGroupID, rules); err != nil {
				return nil, fmt.Errorf("failed to authorize security group rules: %w", err)
			}
		}

		return []string{securityGroupID}, nil
	default:
		return nil, fmt.Errorf("unsupported security group mode: %s", config.SecurityGroupMode)
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
