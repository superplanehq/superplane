package ec2

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type DeleteInstance struct{}

type DeleteInstanceConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	InstanceID string `json:"instance" mapstructure:"instance"`
}

type DeleteInstanceNodeMetadata struct {
	Region       string `json:"region" mapstructure:"region"`
	InstanceName string `json:"instanceName" mapstructure:"instanceName"`
}

type DeleteInstanceExecutionMetadata struct {
	InstanceID   string `json:"instanceId" mapstructure:"instanceId"`
	PollErrors   int    `json:"pollErrors" mapstructure:"pollErrors"`
	PollAttempts int    `json:"pollAttempts" mapstructure:"pollAttempts"`
}

func (c *DeleteInstance) Name() string {
	return "aws.ec2.deleteInstance"
}

func (c *DeleteInstance) Label() string {
	return "EC2 • Delete Instance"
}

func (c *DeleteInstance) Description() string {
	return "Terminate an EC2 instance and wait for termination"
}

func (c *DeleteInstance) Documentation() string {
	return `The Delete Instance component terminates an Amazon EC2 instance and waits until AWS reports it as **terminated**.

## Use Cases

- **Ephemeral cleanup**: Tear down temporary instances after a workflow finishes
- **Cost controls**: Delete unused instances from automation
- **Incident remediation**: Terminate unhealthy instances after replacement capacity exists

## Configuration

- **Region**: AWS region where the instance runs
- **Instance**: EC2 instance ID to terminate

## Output

Emits a deletion payload on the default output channel:
- ` + "`state`" + ` — ` + "`terminated`" + `
`
}

func (c *DeleteInstance) Icon() string {
	return "aws"
}

func (c *DeleteInstance) Color() string {
	return "gray"
}

func (c *DeleteInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteInstance) Configuration() []configuration.Field {
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
			Description: "EC2 instance ID to terminate",
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
	}
}

func (c *DeleteInstance) Setup(ctx core.SetupContext) error {
	config := DeleteInstanceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return err
	}
	if _, err := requireInstanceID(config.InstanceID); err != nil {
		return err
	}

	return ctx.Metadata.Set(resolveDeleteInstanceNodeMetadata(ctx, config))
}

func (c *DeleteInstance) Execute(ctx core.ExecutionContext) error {
	config := DeleteInstanceConfiguration{}
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
	if _, err := client.TerminateInstances(instanceID); err != nil {
		if IsInstanceNotFound(err) {
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteInstancePayloadType, []any{
				map[string]any{
					"instanceId": instanceID,
					"state":      InstanceStateTerminated,
				},
			})
		}
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	if err := ctx.Metadata.Set(DeleteInstanceExecutionMetadata{InstanceID: instanceID}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
}

func (c *DeleteInstance) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "poll", Type: core.HookTypeInternal},
	}
}

func (c *DeleteInstance) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	return c.poll(ctx)
}

func (c *DeleteInstance) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata DeleteInstanceExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}
	if metadata.InstanceID == "" {
		return fmt.Errorf("poll metadata is missing instanceId: execution state may be corrupted")
	}

	config := DeleteInstanceConfiguration{}
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
		if IsInstanceNotFound(err) {
			return c.emitTerminated(ctx, metadata.InstanceID)
		}

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

	if instance.State == InstanceStateTerminated {
		return c.emitTerminated(ctx, metadata.InstanceID)
	}
	if metadata.PollAttempts >= maxInstancePollAttempts {
		return fmt.Errorf("timed out waiting for instance %s to terminate after %d poll attempts (state: %s)", metadata.InstanceID, metadata.PollAttempts, instance.State)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, instancePollInterval)
}

func (c *DeleteInstance) emitTerminated(ctx core.ActionHookContext, instanceID string) error {
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteInstancePayloadType, []any{
		map[string]any{
			"instanceId": instanceID,
			"state":      InstanceStateTerminated,
		},
	})
}

func (c *DeleteInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func resolveDeleteInstanceNodeMetadata(ctx core.SetupContext, config DeleteInstanceConfiguration) DeleteInstanceNodeMetadata {
	metadata := DeleteInstanceNodeMetadata{
		Region: strings.TrimSpace(config.Region),
	}

	instanceID := strings.TrimSpace(config.InstanceID)
	if instanceID == "" {
		return metadata
	}

	if ctx.HTTP == nil || ctx.Integration == nil || metadata.Region == "" {
		metadata.InstanceName = instanceID
		return metadata
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		metadata.InstanceName = instanceID
		return metadata
	}

	client := NewClient(ctx.HTTP, creds, metadata.Region)
	instance, err := client.DescribeInstance(instanceID)
	if err != nil {
		metadata.InstanceName = instanceID
		return metadata
	}

	name := strings.TrimSpace(instance.Name)
	if name != "" {
		metadata.InstanceName = name
		return metadata
	}

	metadata.InstanceName = instanceID
	return metadata
}
