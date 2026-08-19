package cloudwatch

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type GetAlarmMuteRule struct{}

type GetAlarmMuteRuleConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Name   string `json:"name" mapstructure:"name"`
}

type GetAlarmMuteRuleNodeMetadata struct {
	Region string `json:"region" mapstructure:"region"`
	Name   string `json:"name" mapstructure:"name"`
}

func (c *GetAlarmMuteRule) Name() string {
	return "aws.cloudwatch.getAlarmMuteRule"
}

func (c *GetAlarmMuteRule) Label() string {
	return "CloudWatch • Get Alarm Mute Rule"
}

func (c *GetAlarmMuteRule) Description() string {
	return "Fetch the current configuration and status of a CloudWatch alarm mute rule"
}

func (c *GetAlarmMuteRule) Documentation() string {
	return `The Get Alarm Mute Rule component describes a CloudWatch alarm mute rule and emits its current details.

## Use Cases

- **Status inspection**: Check whether a mute rule is SCHEDULED, ACTIVE or EXPIRED before taking action
- **Audit**: Record a mute rule's targeted alarms and schedule at a point in time

## Configuration

- **Region**: AWS region the mute rule lives in
- **Name**: Name of the mute rule to describe

## Output

Emits the mute rule details on the default output channel:
- ` + "`name`" + `, ` + "`alarmMuteRuleArn`" + `, ` + "`description`" + `
- ` + "`scheduleExpression`" + `, ` + "`scheduleDuration`" + `, ` + "`scheduleTimezone`" + `
- ` + "`alarmNames`" + `, ` + "`startDate`" + `, ` + "`expireDate`" + `
- ` + "`status`" + `, ` + "`muteType`" + `, ` + "`lastUpdatedTimestamp`" + `
`
}

func (c *GetAlarmMuteRule) Icon() string {
	return "aws"
}

func (c *GetAlarmMuteRule) Color() string {
	return "gray"
}

func (c *GetAlarmMuteRule) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetAlarmMuteRule) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the mute rule to describe",
		},
	}
}

func (c *GetAlarmMuteRule) Setup(ctx core.SetupContext) error {
	config := GetAlarmMuteRuleConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	name, err := requireAlarmMuteRuleName(config.Name)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(GetAlarmMuteRuleNodeMetadata{
		Region: region,
		Name:   name,
	})
}

func (c *GetAlarmMuteRule) Execute(ctx core.ExecutionContext) error {
	config := GetAlarmMuteRuleConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	name, err := requireAlarmMuteRuleName(config.Name)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	rule, err := client.GetAlarmMuteRule(name)
	if err != nil {
		return fmt.Errorf("failed to get alarm mute rule: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetAlarmMuteRulePayloadType,
		[]any{alarmMuteRuleToMap(rule)},
	)
}

func (c *GetAlarmMuteRule) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetAlarmMuteRule) HandleHook(_ core.ActionHookContext) error {
	return nil
}

func (c *GetAlarmMuteRule) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *GetAlarmMuteRule) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *GetAlarmMuteRule) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
