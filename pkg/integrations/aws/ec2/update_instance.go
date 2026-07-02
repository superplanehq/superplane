package ec2

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	updateInstancePhaseStopping = "stopping"
	updateInstancePhaseStarting = "starting"
)

type UpdateInstance struct{}

type UpdateInstanceConfiguration struct {
	Region             string `json:"region" mapstructure:"region"`
	InstanceID         string `json:"instance" mapstructure:"instance"`
	NewInstanceType    string `json:"instanceType" mapstructure:"instanceType"`
	SecurityGroupID    string `json:"securityGroups" mapstructure:"securityGroups"`
	RestartAfterResize bool   `json:"restartAfterResize" mapstructure:"restartAfterResize"`
}

type UpdateInstanceNodeMetadata struct {
	Region       string `json:"region" mapstructure:"region"`
	InstanceID   string `json:"instanceId" mapstructure:"instanceId"`
	InstanceName string `json:"instanceName" mapstructure:"instanceName"`
}

type UpdateInstanceExecutionMetadata struct {
	InstanceID        string `json:"instanceId" mapstructure:"instanceId"`
	NewInstanceType   string `json:"newInstanceType" mapstructure:"newInstanceType"`
	WasRunning        bool   `json:"wasRunning" mapstructure:"wasRunning"`
	Phase             string `json:"phase" mapstructure:"phase"`
	PollErrors        int    `json:"pollErrors" mapstructure:"pollErrors"`
	StopPollAttempts  int    `json:"stopPollAttempts" mapstructure:"stopPollAttempts"`
	StartPollAttempts int    `json:"startPollAttempts" mapstructure:"startPollAttempts"`
}

func (c *UpdateInstance) Name() string {
	return "aws.ec2.updateInstance"
}

func (c *UpdateInstance) Label() string {
	return "EC2 • Update Instance"
}

func (c *UpdateInstance) Description() string {
	return "Resize an EC2 instance or update its security groups"
}

func (c *UpdateInstance) Documentation() string {
	return `The Update Instance component resizes an EC2 instance to a new instance type and/or updates its security groups.

## Use Cases

- **Right-sizing**: Scale an instance up or down based on observed utilisation
- **Automated resizing**: Use the output of Get Instance Metrics to conditionally resize
- **Security posture updates**: Replace the security groups attached to a running instance

## Configuration

- **Region**: AWS region where the instance runs
- **Instance**: EC2 instance to update
- **New Instance Type**: Target instance type (e.g. ` + "`t3.medium`" + `). The instance is stopped automatically, resized, and restarted. At least one of this field or Security Groups must be set.
- **Security Group**: Replace the instance's security group. Takes effect immediately for VPC instances; does not require a stop/start cycle.
- **Restart After Resize**: Whether to start the instance after changing its type (default true). If unchecked the instance remains stopped after the type change.

## Instance Type Change Lifecycle

1. The component records the current state of the instance.
2. If the instance is running it is stopped automatically.
3. Once stopped, ` + "`ModifyInstanceAttribute`" + ` is called with the new instance type.
4. If the instance was originally running and **Restart After Resize** is enabled, the instance is started again and the component waits for it to reach ` + "`running`" + ` state.

## Output

Emits updated instance details on the default output channel:
- ` + "`instanceId`" + `, ` + "`instanceType`" + `, ` + "`state`" + `, ` + "`region`" + `
- ` + "`publicIpAddress`" + `, ` + "`privateIpAddress`" + `, ` + "`name`" + `
`
}

func (c *UpdateInstance) Icon() string {
	return "aws"
}

func (c *UpdateInstance) Color() string {
	return "gray"
}

func (c *UpdateInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateInstance) Configuration() []configuration.Field {
	return []configuration.Field{
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
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "EC2 instance to update",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.instance",
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
			Name:        "instanceType",
			Label:       "New Instance Type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Target instance type for the resize. The instance will be stopped, resized, and restarted automatically. At least one of New Instance Type or Security Group must be set.",
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
			Name:        "securityGroups",
			Label:       "Security Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Replace the instance's security group. Takes effect immediately without a stop/start cycle.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
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
					},
				},
			},
		},
		{
			Name:        "restartAfterResize",
			Label:       "Restart After Resize",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     "true",
			Description: "Start the instance after changing its type. If unchecked the instance remains stopped.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "instanceType", Values: []string{"*"}},
			},
		},
	}
}

func (c *UpdateInstance) Setup(ctx core.SetupContext) error {
	config, err := decodeUpdateInstanceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	instanceID, err := requireInstanceID(config.InstanceID)
	if err != nil {
		return err
	}

	if err := validateUpdateInstanceConfiguration(config); err != nil {
		return err
	}

	return ctx.Metadata.Set(UpdateInstanceNodeMetadata{
		Region:       region,
		InstanceID:   instanceID,
		InstanceName: resolveInstanceName(ctx, region, instanceID),
	})
}

func validateUpdateInstanceConfiguration(config UpdateInstanceConfiguration) error {
	hasTypeChange := strings.TrimSpace(config.NewInstanceType) != ""
	hasSGChange := strings.TrimSpace(config.SecurityGroupID) != ""

	if !hasTypeChange && !hasSGChange {
		return errors.New("at least one of instanceType or securityGroup must be set")
	}

	return nil
}

func decodeUpdateInstanceConfiguration(raw any) (UpdateInstanceConfiguration, error) {
	config := UpdateInstanceConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return UpdateInstanceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if !hasConfigKey(raw, "restartAfterResize") {
		config.RestartAfterResize = true
	}

	return config, nil
}

func (c *UpdateInstance) Execute(ctx core.ExecutionContext) error {
	config, err := decodeUpdateInstanceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	instanceID, err := requireInstanceID(config.InstanceID)
	if err != nil {
		return err
	}

	if err := validateUpdateInstanceConfiguration(config); err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)

	instance, err := client.DescribeInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to describe instance: %w", err)
	}

	securityGroupID := strings.TrimSpace(config.SecurityGroupID)
	newInstanceType := strings.TrimSpace(config.NewInstanceType)
	hasSGChange := securityGroupID != ""
	needsTypeResize := newInstanceType != "" && newInstanceType != instance.InstanceType

	// Security group updates take effect immediately and do not require stopping.
	if hasSGChange {
		if err := client.ModifySecurityGroups(instanceID, []string{securityGroupID}); err != nil {
			return fmt.Errorf("failed to update security groups: %w", err)
		}
	}

	// If the configured type matches the current type, skip stop/resize.
	if !needsTypeResize {
		updated, err := client.DescribeInstance(instanceID)
		if err != nil {
			return fmt.Errorf("failed to describe instance after update: %w", err)
		}

		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			UpdateInstancePayloadType,
			[]any{instanceDetailsToMap(updated)},
		)
	}

	wasRunning := instance.State == InstanceStateRunning

	metadata := UpdateInstanceExecutionMetadata{
		InstanceID:      instanceID,
		NewInstanceType: newInstanceType,
		WasRunning:      wasRunning,
	}

	// If already stopped, modify the type immediately without a stop cycle.
	if instance.State == InstanceStateStopped {
		return c.applyTypeChange(ctx, client, config, metadata)
	}

	if instance.State != InstanceStateRunning {
		return fmt.Errorf(
			"instance %s is in state %q; it must be running or stopped to resize",
			instanceID, instance.State,
		)
	}

	metadata.Phase = updateInstancePhaseStopping
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if _, err := client.StopInstances(instanceID); err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
}

// applyTypeChange calls ModifyInstanceAttribute and either emits immediately
// (if restart is disabled or the instance was originally stopped) or starts the
// instance and moves to the "starting" poll phase.
func (c *UpdateInstance) applyTypeChange(ctx core.ExecutionContext, client *Client, config UpdateInstanceConfiguration, metadata UpdateInstanceExecutionMetadata) error {
	if err := client.ModifyInstanceType(metadata.InstanceID, metadata.NewInstanceType); err != nil {
		return fmt.Errorf("failed to modify instance type: %w", err)
	}

	if !metadata.WasRunning || !config.RestartAfterResize {
		instance, err := client.DescribeInstance(metadata.InstanceID)
		if err != nil {
			return fmt.Errorf("failed to describe instance after type change: %w", err)
		}

		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			UpdateInstancePayloadType,
			[]any{instanceDetailsToMap(instance)},
		)
	}

	if _, err := client.StartInstances(metadata.InstanceID); err != nil {
		return fmt.Errorf("failed to start instance after resize: %w", err)
	}

	metadata.Phase = updateInstancePhaseStarting
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
}

func (c *UpdateInstance) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "poll", Type: core.HookTypeInternal},
	}
}

func (c *UpdateInstance) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}

	return c.poll(ctx)
}

func (c *UpdateInstance) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata UpdateInstanceExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.InstanceID == "" {
		return fmt.Errorf("poll metadata is missing instanceId: execution state may be corrupted")
	}

	config, err := decodeUpdateInstanceConfiguration(ctx.Configuration)
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
	switch metadata.Phase {
	case updateInstancePhaseStopping:
		metadata.StopPollAttempts++
	case updateInstancePhaseStarting:
		metadata.StartPollAttempts++
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	switch metadata.Phase {
	case updateInstancePhaseStopping:
		return c.pollStopping(ctx, client, config, instance, metadata)
	case updateInstancePhaseStarting:
		return c.pollStarting(ctx, instance, metadata)
	default:
		return fmt.Errorf("unknown phase in poll metadata: %q", metadata.Phase)
	}
}

func (c *UpdateInstance) pollStopping(ctx core.ActionHookContext, client *Client, config UpdateInstanceConfiguration, instance *InstanceDetails, metadata UpdateInstanceExecutionMetadata) error {
	switch instance.State {
	case InstanceStateStopped:
		// Instance is stopped; apply the type change now.
		if err := client.ModifyInstanceType(metadata.InstanceID, metadata.NewInstanceType); err != nil {
			return fmt.Errorf("failed to modify instance type: %w", err)
		}

		if !metadata.WasRunning || !config.RestartAfterResize {
			updated, err := client.DescribeInstance(metadata.InstanceID)
			if err != nil {
				return fmt.Errorf("failed to describe instance after type change: %w", err)
			}
			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				UpdateInstancePayloadType,
				[]any{instanceDetailsToMap(updated)},
			)
		}

		if _, err := client.StartInstances(metadata.InstanceID); err != nil {
			return fmt.Errorf("failed to start instance after resize: %w", err)
		}

		metadata.Phase = updateInstancePhaseStarting
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)

	case InstanceStateTerminated, InstanceStateShuttingDown:
		return fmt.Errorf("instance %s entered state %q unexpectedly while stopping for resize", instance.InstanceID, instance.State)
	}

	if metadata.StopPollAttempts >= maxInstancePollAttempts {
		return fmt.Errorf("timed out waiting for instance %s to stop after %d poll attempts (state: %s)", instance.InstanceID, metadata.StopPollAttempts, instance.State)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
}

func (c *UpdateInstance) pollStarting(ctx core.ActionHookContext, instance *InstanceDetails, metadata UpdateInstanceExecutionMetadata) error {
	switch instance.State {
	case InstanceStateRunning:
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			UpdateInstancePayloadType,
			[]any{instanceDetailsToMap(instance)},
		)
	case InstanceStateTerminated, InstanceStateShuttingDown, InstanceStateStopped, InstanceStateStopping:
		return fmt.Errorf("instance %s entered state %q and will not reach running without intervention", instance.InstanceID, instance.State)
	}

	if metadata.StartPollAttempts >= maxInstancePollAttempts {
		return fmt.Errorf("timed out waiting for instance %s to reach running state after %d poll attempts (state: %s)", instance.InstanceID, metadata.StartPollAttempts, instance.State)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
}

func (c *UpdateInstance) Cancel(ctx core.ExecutionContext) error {
	config, err := decodeUpdateInstanceConfiguration(ctx.Configuration)
	if err != nil || !config.RestartAfterResize {
		return nil
	}

	instanceID, metadata := decodeUpdateInstanceCancelMetadata(ctx.Metadata.Get())
	if instanceID == "" {
		return nil
	}

	shouldRestart := metadata.WasRunning && metadata.Phase == updateInstancePhaseStopping
	if !shouldRestart && metadata.Phase == "" && strings.TrimSpace(config.NewInstanceType) != "" {
		region, regionErr := requireRegion(config.Region)
		if regionErr != nil {
			return nil
		}

		creds, credsErr := common.CredentialsFromInstallation(ctx.Integration)
		if credsErr != nil {
			return nil
		}

		client := NewClient(ctx.HTTP, creds, region)
		instance, describeErr := client.DescribeInstance(instanceID)
		if describeErr == nil && instance.State == InstanceStateStopping {
			shouldRestart = true
		}
	}

	if !shouldRestart {
		return nil
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return nil
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil
	}

	client := NewClient(ctx.HTTP, creds, region)
	_, _ = client.StartInstances(instanceID)
	return nil
}

func decodeUpdateInstanceCancelMetadata(raw any) (string, UpdateInstanceExecutionMetadata) {
	var metadata UpdateInstanceExecutionMetadata
	if err := mapstructure.Decode(raw, &metadata); err != nil {
		return "", UpdateInstanceExecutionMetadata{}
	}

	if metadata.InstanceID != "" {
		return metadata.InstanceID, metadata
	}

	var nodeMetadata UpdateInstanceNodeMetadata
	if err := mapstructure.Decode(raw, &nodeMetadata); err != nil {
		return "", UpdateInstanceExecutionMetadata{}
	}

	return nodeMetadata.InstanceID, metadata
}

func (c *UpdateInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
