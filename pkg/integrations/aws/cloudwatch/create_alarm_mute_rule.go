package cloudwatch

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type CreateAlarmMuteRule struct{}

type CreateAlarmMuteRuleConfiguration struct {
	Region             string   `json:"region" mapstructure:"region"`
	Name               string   `json:"name" mapstructure:"name"`
	Description        string   `json:"description" mapstructure:"description"`
	AlarmNames         []string `json:"alarmNames" mapstructure:"alarmNames"`
	ScheduleExpression string   `json:"scheduleExpression" mapstructure:"scheduleExpression"`
	Duration           string   `json:"duration" mapstructure:"duration"`
	Timezone           string   `json:"timezone" mapstructure:"timezone"`
	StartDate          string   `json:"startDate" mapstructure:"startDate"`
	ExpireDate         string   `json:"expireDate" mapstructure:"expireDate"`
}

type CreateAlarmMuteRuleNodeMetadata struct {
	Region string `json:"region" mapstructure:"region"`
	Name   string `json:"name" mapstructure:"name"`
}

func (c *CreateAlarmMuteRule) Name() string {
	return "aws.cloudwatch.createAlarmMuteRule"
}

func (c *CreateAlarmMuteRule) Label() string {
	return "CloudWatch • Create Alarm Mute Rule"
}

func (c *CreateAlarmMuteRule) Description() string {
	return "Mute one or more CloudWatch alarms during a scheduled window, like a Grafana silence"
}

func (c *CreateAlarmMuteRule) Documentation() string {
	return `The Create Alarm Mute Rule component creates or updates a CloudWatch alarm mute rule. While a rule is active, its targeted alarms keep evaluating and changing state, but their configured actions (SNS notifications, EC2 automations, etc.) are muted.

## Use Cases

- **Planned maintenance**: Silence noisy alarms during a deploy or maintenance window
- **Recurring quiet hours**: Mute non-critical alarms every night or every weekend
- **Incident response**: Silence a known-flapping alarm once, for a fixed duration

## Configuration

- **Region**: AWS region the alarms and the rule live in
- **Name**: Unique mute rule name within the region; creating a rule with an existing name updates it
- **Description** *(toggleable)*: Free-text description
- **Alarms**: Alarms to mute while the rule is active (up to 100)
- **Schedule Expression**: A recurring ` + "`cron(Minutes Hours Day-of-month Month Day-of-week)`" + ` expression (e.g. ` + "`cron(0 2 * * *)`" + `) or a one-time ` + "`at(yyyy-MM-ddThh:mm)`" + ` expression (e.g. ` + "`at(2026-01-20T14:00)`" + `)
- **Duration**: How long alarms stay muted once the schedule activates, in ISO 8601 duration format (e.g. ` + "`PT2H`" + ` for 2 hours), between ` + "`PT1M`" + ` and ` + "`P15D`" + `
- **Timezone**: Timezone the schedule expression runs in, and that offset-less start/expire dates are interpreted in (default UTC)
- **Starts At** *(toggleable)*: Rule takes effect immediately if left unset
- **Expires At** *(toggleable)*: Rule never expires if left unset

## Output

Emits the mute rule on the default output channel, including ` + "`name`" + `, ` + "`alarmMuteRuleArn`" + `,
` + "`scheduleExpression`" + `, ` + "`scheduleDuration`" + `, ` + "`alarmNames`" + `, ` + "`status`" + ` and ` + "`muteType`" + `.
`
}

func (c *CreateAlarmMuteRule) Icon() string {
	return "aws"
}

func (c *CreateAlarmMuteRule) Color() string {
	return "gray"
}

func (c *CreateAlarmMuteRule) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateAlarmMuteRule) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Unique mute rule name within the region; creating a rule with an existing name updates it",
		},
		{
			Name:      "description",
			Label:     "Description",
			Type:      configuration.FieldTypeText,
			Required:  false,
			Togglable: true,
		},
		{
			Name:        "alarmNames",
			Label:       "Alarms",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Alarms to mute while the rule is active (up to 100)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       "cloudwatch.alarm",
					Multi:      true,
					Parameters: []configuration.ParameterRef{regionParameter()},
				},
			},
		},
		{
			Name:        "scheduleExpression",
			Label:       "Schedule Expression",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: `A recurring "cron(Minutes Hours Day-of-month Month Day-of-week)" expression (e.g. cron(0 2 * * *)) or a one-time "at(yyyy-MM-ddThh:mm)" expression (e.g. at(2026-01-20T14:00))`,
		},
		{
			Name:        "duration",
			Label:       "Duration",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "PT1H",
			Description: "How long alarms stay muted once the schedule activates, in ISO 8601 duration format (e.g. PT2H). Between PT1M and P15D",
		},
		scheduleTimezoneField(),
		{
			Name:        "startDate",
			Label:       "Starts At",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Togglable:   true,
			Description: "Rule takes effect immediately if left unset",
		},
		{
			Name:        "expireDate",
			Label:       "Expires At",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Togglable:   true,
			Description: "Rule never expires if left unset",
		},
	}
}

func (c *CreateAlarmMuteRule) Setup(ctx core.SetupContext) error {
	config := CreateAlarmMuteRuleConfiguration{}
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

	if _, err := requireAlarmNames(config.AlarmNames); err != nil {
		return err
	}

	if _, err := requireScheduleExpression(config.ScheduleExpression); err != nil {
		return err
	}

	if _, err := requireScheduleDuration(config.Duration); err != nil {
		return err
	}

	timezone := effectiveTimezone(config.Timezone)
	if _, err := parseOptionalMuteRuleTimestamp(config.StartDate, timezone); err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}

	if _, err := parseOptionalMuteRuleTimestamp(config.ExpireDate, timezone); err != nil {
		return fmt.Errorf("invalid expire date: %w", err)
	}

	return ctx.Metadata.Set(CreateAlarmMuteRuleNodeMetadata{
		Region: region,
		Name:   name,
	})
}

func (c *CreateAlarmMuteRule) Execute(ctx core.ExecutionContext) error {
	config := CreateAlarmMuteRuleConfiguration{}
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

	alarmNames, err := requireAlarmNames(config.AlarmNames)
	if err != nil {
		return err
	}

	expression, err := requireScheduleExpression(config.ScheduleExpression)
	if err != nil {
		return err
	}

	duration, err := requireScheduleDuration(config.Duration)
	if err != nil {
		return err
	}

	timezone := effectiveTimezone(config.Timezone)
	startDate, err := parseOptionalMuteRuleTimestamp(config.StartDate, timezone)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}

	expireDate, err := parseOptionalMuteRuleTimestamp(config.ExpireDate, timezone)
	if err != nil {
		return fmt.Errorf("invalid expire date: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	err = client.PutAlarmMuteRule(PutAlarmMuteRuleInput{
		Name:        name,
		Description: config.Description,
		Schedule: AlarmMuteRuleSchedule{
			Expression: expression,
			Duration:   duration,
			Timezone:   timezone,
		},
		AlarmNames: alarmNames,
		StartDate:  startDate,
		ExpireDate: expireDate,
	})

	if err != nil {
		return fmt.Errorf("failed to create alarm mute rule: %w", err)
	}

	rule, err := client.GetAlarmMuteRule(name)
	if err != nil {
		return fmt.Errorf("failed to fetch alarm mute rule after creation: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CreateAlarmMuteRulePayloadType,
		[]any{alarmMuteRuleToMap(rule)},
	)
}

func (c *CreateAlarmMuteRule) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateAlarmMuteRule) HandleHook(_ core.ActionHookContext) error {
	return nil
}

func (c *CreateAlarmMuteRule) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *CreateAlarmMuteRule) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *CreateAlarmMuteRule) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
