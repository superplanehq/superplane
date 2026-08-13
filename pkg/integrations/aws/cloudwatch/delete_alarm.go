package cloudwatch

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type DeleteAlarm struct{}

type DeleteAlarmConfiguration struct {
	Region    string `json:"region" mapstructure:"region"`
	AlarmName string `json:"alarm" mapstructure:"alarm"`
}

type DeleteAlarmNodeMetadata struct {
	Region    string `json:"region" mapstructure:"region"`
	AlarmName string `json:"alarmName" mapstructure:"alarmName"`
}

func (c *DeleteAlarm) Name() string {
	return "aws.cloudwatch.deleteAlarm"
}

func (c *DeleteAlarm) Label() string {
	return "CloudWatch • Delete Alarm"
}

func (c *DeleteAlarm) Description() string {
	return "Delete a CloudWatch metric alarm"
}

func (c *DeleteAlarm) Documentation() string {
	return `The Delete Alarm component removes a CloudWatch metric alarm.

The alarm is read before it is deleted, so the emitted payload is a snapshot of what was removed —
for a single-metric alarm, enough to recreate it with Create Alarm if the deletion needs to be undone.

## Use Cases

- **Cleanup**: Remove temporary alarms raised by automation workflows
- **Decommissioning**: Delete monitoring for resources being retired
- **Rollback**: Undo alarm creation steps in a failed provisioning run

## Configuration

- **Region**: AWS region where the alarm lives
- **Alarm**: CloudWatch alarm to delete (` + "`cloudwatch.alarm`" + ` resource picker)

## Output

Emits the alarm as it was immediately before deletion on the default output channel, with the same
fields as Get Alarm plus ` + "`deleted`" + `. The console URL is omitted, since it no longer resolves.

Deleting an alarm that does not exist fails. This component deletes metric alarms; composite alarms
are not handled. An alarm driven by metric math, anomaly detection or a Metrics Insights query is
deleted, but its metric definition is not captured in the snapshot, so Create Alarm cannot restore it.
`
}

func (c *DeleteAlarm) Icon() string {
	return "aws"
}

func (c *DeleteAlarm) Color() string {
	return "gray"
}

func (c *DeleteAlarm) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteAlarm) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		alarmField("CloudWatch alarm to delete"),
	}
}

func (c *DeleteAlarm) Setup(ctx core.SetupContext) error {
	config := DeleteAlarmConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	alarmName, err := requireAlarmName(config.AlarmName)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(DeleteAlarmNodeMetadata{
		Region:    region,
		AlarmName: alarmName,
	})
}

func (c *DeleteAlarm) Execute(ctx core.ExecutionContext) error {
	config := DeleteAlarmConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	alarmName, err := requireAlarmName(config.AlarmName)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	// Snapshot the alarm first: DeleteAlarms returns nothing, and a deletion that
	// cannot be described is a deletion nothing downstream can act on.
	client := NewClient(ctx.HTTP, creds, region)
	alarm, err := client.DescribeAlarm(alarmName)
	if err != nil {
		return fmt.Errorf("failed to describe alarm: %w", err)
	}

	if err := client.DeleteAlarms(alarmName); err != nil {
		return fmt.Errorf("failed to delete alarm: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeleteAlarmPayloadType,
		[]any{deletedAlarmToMap(alarm)},
	)
}

// deletedAlarmToMap describes the alarm that was removed. The console URL is
// dropped because it no longer resolves once the alarm is gone.
func deletedAlarmToMap(alarm *MetricAlarm) map[string]any {
	payload := alarmToMap(alarm)
	delete(payload, "consoleUrl")
	payload["deleted"] = true

	return payload
}

func (c *DeleteAlarm) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteAlarm) HandleHook(_ core.ActionHookContext) error {
	return nil
}

func (c *DeleteAlarm) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteAlarm) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *DeleteAlarm) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *DeleteAlarm) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
