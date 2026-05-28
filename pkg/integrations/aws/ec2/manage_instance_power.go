package ec2

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	powerOperationStart     = "start"
	powerOperationStop      = "stop"
	powerOperationReboot    = "reboot"
	powerOperationHibernate = "hibernate"
)

var validPowerOperations = []string{
	powerOperationStart,
	powerOperationStop,
	powerOperationReboot,
	powerOperationHibernate,
}

type ManageInstancePower struct{}

type ManageInstancePowerConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	InstanceID string `json:"instance" mapstructure:"instance"`
	Operation  string `json:"operation" mapstructure:"operation"`
}

type ManageInstancePowerNodeMetadata struct {
	Region       string `json:"region" mapstructure:"region"`
	InstanceName string `json:"instanceName" mapstructure:"instanceName"`
}

type ManageInstancePowerExecutionMetadata struct {
	InstanceID   string `json:"instanceId" mapstructure:"instanceId"`
	Operation    string `json:"operation" mapstructure:"operation"`
	PollErrors   int    `json:"pollErrors" mapstructure:"pollErrors"`
	PollAttempts int    `json:"pollAttempts" mapstructure:"pollAttempts"`
}

func (c *ManageInstancePower) Name() string {
	return "aws.ec2.manageInstancePower"
}

func (c *ManageInstancePower) Label() string {
	return "EC2 • Manage Instance Power"
}

func (c *ManageInstancePower) Description() string {
	return "Perform power operations on an EC2 instance"
}

func (c *ManageInstancePower) Documentation() string {
	return `The Manage Instance Power component performs power management operations on an EC2 instance.

## Use Cases

- **Scheduled workloads**: Start or stop instances on demand from a workflow
- **Cost optimisation**: Stop or hibernate non-production instances outside business hours
- **Maintenance workflows**: Stop an instance before resizing or patching, start it after
- **Recovery**: Reboot an instance experiencing issues

## Configuration

- **Region**: AWS region where the instance runs
- **Instance**: EC2 instance to manage
- **Operation**: The power operation to perform:
  - **Start**: Start a stopped instance and wait for it to reach running state
  - **Stop**: Gracefully stop a running instance and wait for stopped state
  - **Reboot**: Reboot a running instance (completes once the reboot signal is sent)
  - **Hibernate**: Stop an instance and save RAM to disk (instance must have hibernation enabled)

## Output

Emits instance details on the default output channel once the operation completes:
- ` + "`instanceId`" + `, ` + "`state`" + `, ` + "`region`" + `
- ` + "`publicIpAddress`" + `, ` + "`privateIpAddress`" + ` (start only)

## Important Notes

- **Hibernate** requires the instance to have been launched with hibernation enabled
- **Reboot** emits immediately after the reboot signal is accepted; it does not wait for the OS to come back online
`
}

func (c *ManageInstancePower) Icon() string {
	return "aws"
}

func (c *ManageInstancePower) Color() string {
	return "gray"
}

func (c *ManageInstancePower) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ManageInstancePower) Configuration() []configuration.Field {
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
			Description: "EC2 instance to manage",
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
			Name:        "operation",
			Label:       "Operation",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The power operation to perform",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Start", Value: powerOperationStart},
						{Label: "Stop", Value: powerOperationStop},
						{Label: "Reboot", Value: powerOperationReboot},
						{Label: "Hibernate", Value: powerOperationHibernate},
					},
				},
			},
		},
	}
}

func (c *ManageInstancePower) Setup(ctx core.SetupContext) error {
	config := ManageInstancePowerConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}
	if _, err := requireInstanceID(config.InstanceID); err != nil {
		return err
	}
	if !slices.Contains(validPowerOperations, strings.TrimSpace(config.Operation)) {
		return fmt.Errorf("invalid operation %q: must be one of start, stop, reboot, hibernate", config.Operation)
	}

	return ctx.Metadata.Set(ManageInstancePowerNodeMetadata{
		Region:       region,
		InstanceName: resolveInstanceName(ctx, region, strings.TrimSpace(config.InstanceID)),
	})
}

func (c *ManageInstancePower) Execute(ctx core.ExecutionContext) error {
	config := ManageInstancePowerConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}
	instanceID, err := requireInstanceID(config.InstanceID)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	operation := strings.TrimSpace(config.Operation)

	switch operation {
	case powerOperationStart:
		if _, err := client.StartInstances(instanceID); err != nil {
			return fmt.Errorf("failed to start instance: %w", err)
		}
	case powerOperationStop:
		if _, err := client.StopInstances(instanceID); err != nil {
			return fmt.Errorf("failed to stop instance: %w", err)
		}
	case powerOperationHibernate:
		if _, err := client.HibernateInstances(instanceID); err != nil {
			return fmt.Errorf("failed to hibernate instance: %w", err)
		}
	case powerOperationReboot:
		if err := client.RebootInstances(instanceID); err != nil {
			return fmt.Errorf("failed to reboot instance: %w", err)
		}
		// Reboot is fire-and-forget: the instance stays running throughout.
		// Emit immediately with the current state rather than polling.
		instance, err := client.DescribeInstance(instanceID)
		if err != nil {
			return fmt.Errorf("failed to describe instance after reboot: %w", err)
		}
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			ManageInstancePowerRebootPayloadType,
			[]any{instanceDetailsToMap(instance)},
		)
	default:
		return fmt.Errorf("invalid operation %q", operation)
	}

	if err := ctx.Metadata.Set(ManageInstancePowerExecutionMetadata{
		InstanceID: instanceID,
		Operation:  operation,
	}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
}

func (c *ManageInstancePower) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "poll", Type: core.HookTypeInternal},
	}
}

func (c *ManageInstancePower) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}

	return c.poll(ctx)
}

func (c *ManageInstancePower) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata ManageInstancePowerExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}
	if metadata.InstanceID == "" {
		return fmt.Errorf("poll metadata is missing instanceId: execution state may be corrupted")
	}

	config := ManageInstancePowerConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
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

	switch metadata.Operation {
	case powerOperationStart:
		return c.pollStart(ctx, instance, metadata)
	case powerOperationStop, powerOperationHibernate:
		return c.pollStop(ctx, instance, metadata)
	default:
		return fmt.Errorf("unknown operation in poll metadata: %q", metadata.Operation)
	}
}

func (c *ManageInstancePower) pollStart(ctx core.ActionHookContext, instance *InstanceDetails, metadata ManageInstancePowerExecutionMetadata) error {
	switch instance.State {
	case InstanceStateRunning:
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			ManageInstancePowerStartPayloadType,
			[]any{instanceDetailsToMap(instance)},
		)
	case InstanceStateTerminated, InstanceStateShuttingDown, InstanceStateStopped, InstanceStateStopping:
		return fmt.Errorf("instance %s entered state %q and will not reach running without intervention", instance.InstanceID, instance.State)
	}

	if metadata.PollAttempts >= maxInstancePollAttempts {
		return fmt.Errorf("timed out waiting for instance %s to start after %d poll attempts (state: %s)", instance.InstanceID, metadata.PollAttempts, instance.State)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
}

func (c *ManageInstancePower) pollStop(ctx core.ActionHookContext, instance *InstanceDetails, metadata ManageInstancePowerExecutionMetadata) error {
	switch instance.State {
	case InstanceStateStopped:
		payloadType := ManageInstancePowerStopPayloadType
		if metadata.Operation == powerOperationHibernate {
			payloadType = ManageInstancePowerHibernatePayloadType
		}
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			payloadType,
			[]any{instanceDetailsToMap(instance)},
		)
	case InstanceStateTerminated, InstanceStateShuttingDown:
		return fmt.Errorf("instance %s entered state %q unexpectedly while stopping", instance.InstanceID, instance.State)
	}

	if metadata.PollAttempts >= maxInstancePollAttempts {
		return fmt.Errorf("timed out waiting for instance %s to stop after %d poll attempts (state: %s)", instance.InstanceID, metadata.PollAttempts, instance.State)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
}

func (c *ManageInstancePower) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ManageInstancePower) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ManageInstancePower) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ManageInstancePower) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
