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

type DeleteLoadBalancer struct{}

type DeleteLoadBalancerConfiguration struct {
	Region          string `json:"region" mapstructure:"region"`
	LoadBalancerARN string `json:"loadBalancer" mapstructure:"loadBalancer"`
}

type DeleteLoadBalancerNodeMetadata struct {
	Region           string `json:"region" mapstructure:"region"`
	LoadBalancerName string `json:"loadBalancerName" mapstructure:"loadBalancerName"`
}

type DeleteLoadBalancerExecutionMetadata struct {
	LoadBalancerARN string `json:"loadBalancerArn" mapstructure:"loadBalancerArn"`
	PollErrors      int    `json:"pollErrors" mapstructure:"pollErrors"`
	PollAttempts    int    `json:"pollAttempts" mapstructure:"pollAttempts"`
}

func (c *DeleteLoadBalancer) Name() string {
	return "aws.ec2.deleteLoadBalancer"
}

func (c *DeleteLoadBalancer) Label() string {
	return "EC2 • Delete Load Balancer"
}

func (c *DeleteLoadBalancer) Description() string {
	return "Delete an Elastic Load Balancer and wait for deletion to complete"
}

func (c *DeleteLoadBalancer) Documentation() string {
	return `The Delete Load Balancer component deletes an Elastic Load Balancer (ELBv2) and waits until AWS confirms deletion.

## Use Cases

- **Environment teardown**: Remove load balancers when decommissioning an environment
- **Cost control**: Delete unused load balancers after a workflow finishes
- **Blue/green cleanup**: Remove the old load balancer after a successful deployment

## Configuration

- **Region**: AWS region where the load balancer resides
- **Load Balancer**: ARN of the load balancer to delete

## Output

Emits a deletion payload on the default output channel once AWS confirms the load balancer is gone:
- ` + "`loadBalancerArn`" + ` — ARN of the deleted load balancer
- ` + "`state`" + ` — ` + "`deleted`" + `
`
}

func (c *DeleteLoadBalancer) Icon() string {
	return "aws"
}

func (c *DeleteLoadBalancer) Color() string {
	return "gray"
}

func (c *DeleteLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteLoadBalancer) Configuration() []configuration.Field {
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
			Name:        "loadBalancer",
			Label:       "Load Balancer",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Load balancer to delete",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.loadBalancer",
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

func (c *DeleteLoadBalancer) Setup(ctx core.SetupContext) error {
	config := DeleteLoadBalancerConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	if _, err := requireLoadBalancerARN(config.LoadBalancerARN); err != nil {
		return err
	}

	return ctx.Metadata.Set(resolveDeleteLoadBalancerNodeMetadata(ctx, config, region))
}

func (c *DeleteLoadBalancer) Execute(ctx core.ExecutionContext) error {
	config := DeleteLoadBalancerConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	lbARN, err := requireLoadBalancerARN(config.LoadBalancerARN)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get AWS credentials: %v", err))
	}

	client := NewClient(ctx.HTTP, creds, region)
	if _, err := client.DeleteLoadBalancer(lbARN); err != nil {
		if IsLoadBalancerNotFound(err) {
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteLoadBalancerPayloadType, []any{
				map[string]any{
					"loadBalancerArn": lbARN,
					"state":           LoadBalancerStateDeleted,
				},
			})
		}
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete load balancer: %v", err))
	}

	if err := ctx.Metadata.Set(DeleteLoadBalancerExecutionMetadata{LoadBalancerARN: lbARN}); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to save execution metadata: %v", err))
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, loadBalancerPollInterval)
}

func (c *DeleteLoadBalancer) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "poll", Type: core.HookTypeInternal},
	}
}

func (c *DeleteLoadBalancer) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != "poll" {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("unknown action: %s", ctx.Name))
	}

	return c.poll(ctx)
}

func (c *DeleteLoadBalancer) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata DeleteLoadBalancerExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode metadata: %v", err))
	}
	if metadata.LoadBalancerARN == "" {
		return ctx.ExecutionState.Fail("error", "poll metadata is missing loadBalancerArn: execution state may be corrupted")
	}

	config := DeleteLoadBalancerConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get AWS credentials: %v", err))
	}

	client := NewClient(ctx.HTTP, creds, region)
	lb, err := client.DescribeLoadBalancer(metadata.LoadBalancerARN)
	if err != nil {
		if IsLoadBalancerNotFound(err) {
			return c.emitDeleted(ctx, metadata.LoadBalancerARN)
		}

		metadata.PollErrors++
		ctx.Logger.Warnf("failed to describe load balancer %s (attempt %d/%d): %v",
			metadata.LoadBalancerARN, metadata.PollErrors, maxLoadBalancerPollErrors, err)
		if metadata.PollErrors >= maxLoadBalancerPollErrors {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("giving up polling load balancer %s after %d consecutive errors: %v",
				metadata.LoadBalancerARN, maxLoadBalancerPollErrors, err))
		}
		if err := ctx.Metadata.Set(metadata); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to save poll error count: %v", err))
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, loadBalancerPollInterval)
	}

	metadata.PollErrors = 0
	metadata.PollAttempts++
	if err := ctx.Metadata.Set(metadata); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to save poll attempt count: %v", err))
	}

	if metadata.PollAttempts >= maxLoadBalancerPollAttempts {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("timed out waiting for load balancer %s to be deleted after %d poll attempts (state: %s)",
			metadata.LoadBalancerARN, metadata.PollAttempts, lb.State))
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, loadBalancerPollInterval)
}

func (c *DeleteLoadBalancer) emitDeleted(ctx core.ActionHookContext, lbARN string) error {
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteLoadBalancerPayloadType, []any{
		map[string]any{
			"loadBalancerArn": lbARN,
			"state":           LoadBalancerStateDeleted,
		},
	})
}

func (c *DeleteLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func resolveDeleteLoadBalancerNodeMetadata(ctx core.SetupContext, config DeleteLoadBalancerConfiguration, region string) DeleteLoadBalancerNodeMetadata {
	metadata := DeleteLoadBalancerNodeMetadata{
		Region: region,
	}

	lbARN := strings.TrimSpace(config.LoadBalancerARN)
	if lbARN == "" {
		return metadata
	}

	if ctx.HTTP == nil || ctx.Integration == nil {
		metadata.LoadBalancerName = lbARN
		return metadata
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		metadata.LoadBalancerName = lbARN
		return metadata
	}

	client := NewClient(ctx.HTTP, creds, region)
	lb, err := client.DescribeLoadBalancer(lbARN)
	if err != nil {
		metadata.LoadBalancerName = lbARN
		return metadata
	}

	name := strings.TrimSpace(lb.Name)
	if name != "" {
		metadata.LoadBalancerName = name
		return metadata
	}

	metadata.LoadBalancerName = lbARN
	return metadata
}
