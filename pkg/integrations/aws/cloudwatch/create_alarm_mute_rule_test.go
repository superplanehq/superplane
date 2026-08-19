package cloudwatch

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const getAlarmMuteRuleXML = `
<GetAlarmMuteRuleResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <GetAlarmMuteRuleResult>
    <Name>DailyMaintenanceWindow</Name>
    <AlarmMuteRuleArn>arn:aws:cloudwatch:us-east-1:123456789012:alarm-mute-rule:DailyMaintenanceWindow</AlarmMuteRuleArn>
    <Description>Mute alarms during daily maintenance</Description>
    <Rule>
      <Schedule>
        <Expression>cron(0 2 * * *)</Expression>
        <Duration>PT2H</Duration>
        <Timezone>UTC</Timezone>
      </Schedule>
    </Rule>
    <MuteTargets>
      <AlarmNames>
        <member>WebServerCPUAlarm</member>
        <member>DatabaseConnectionAlarm</member>
      </AlarmNames>
    </MuteTargets>
    <Status>SCHEDULED</Status>
    <LastUpdatedTimestamp>2026-01-15T10:30:00Z</LastUpdatedTimestamp>
    <MuteType>RECURRING</MuteType>
  </GetAlarmMuteRuleResult>
</GetAlarmMuteRuleResponse>`

func validCreateAlarmMuteRuleConfig() map[string]any {
	return map[string]any{
		"region":             "us-east-1",
		"name":               "DailyMaintenanceWindow",
		"alarmNames":         []string{"WebServerCPUAlarm", "DatabaseConnectionAlarm"},
		"scheduleExpression": "cron(0 2 * * *)",
		"duration":           "PT2H",
	}
}

func Test__CreateAlarmMuteRule__Setup(t *testing.T) {
	component := &CreateAlarmMuteRule{}

	t.Run("missing region -> error", func(t *testing.T) {
		configuration := validCreateAlarmMuteRuleConfig()
		configuration["region"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing name -> error", func(t *testing.T) {
		configuration := validCreateAlarmMuteRuleConfig()
		configuration["name"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing alarm names -> error", func(t *testing.T) {
		configuration := validCreateAlarmMuteRuleConfig()
		configuration["alarmNames"] = []string{}
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "at least one alarm is required")
	})

	t.Run("missing schedule expression -> error", func(t *testing.T) {
		configuration := validCreateAlarmMuteRuleConfig()
		configuration["scheduleExpression"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "schedule expression is required")
	})

	t.Run("schedule expression missing cron/at prefix -> error", func(t *testing.T) {
		configuration := validCreateAlarmMuteRuleConfig()
		configuration["scheduleExpression"] = "0 2 * * *"
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, `must be a recurring "cron(...)" or one-time "at(...)" expression`)
	})

	t.Run("missing duration -> error", func(t *testing.T) {
		configuration := validCreateAlarmMuteRuleConfig()
		configuration["duration"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "duration is required")
	})

	t.Run("invalid start date -> error", func(t *testing.T) {
		configuration := validCreateAlarmMuteRuleConfig()
		configuration["startDate"] = "not-a-date"
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "invalid start date")
	})

	t.Run("invalid expire date -> error", func(t *testing.T) {
		configuration := validCreateAlarmMuteRuleConfig()
		configuration["expireDate"] = "not-a-date"
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "invalid expire date")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: validCreateAlarmMuteRuleConfig(),
			Metadata:      metadata,
		})
		require.NoError(t, err)

		nodeMetadata, ok := metadata.Metadata.(CreateAlarmMuteRuleNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", nodeMetadata.Region)
		assert.Equal(t, "DailyMaintenanceWindow", nodeMetadata.Name)
	})
}

func Test__CreateAlarmMuteRule__Execute(t *testing.T) {
	component := &CreateAlarmMuteRule{}

	t.Run("creates the rule and emits its fetched details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// PutAlarmMuteRule returns an empty body on success.
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(getAlarmMuteRuleXML))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:  validCreateAlarmMuteRuleConfig(),
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    awsIntegrationContext(),
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, CreateAlarmMuteRulePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "DailyMaintenanceWindow", data["name"])
		assert.Equal(t, "cron(0 2 * * *)", data["scheduleExpression"])
		assert.Equal(t, "PT2H", data["scheduleDuration"])
		assert.Equal(t, "SCHEDULED", data["status"])
		assert.Equal(t, []string{"WebServerCPUAlarm", "DatabaseConnectionAlarm"}, data["alarmNames"])
	})

	t.Run("sends the schedule and targets to CloudWatch", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(getAlarmMuteRuleXML))},
			},
		}

		configuration := validCreateAlarmMuteRuleConfig()
		configuration["description"] = "Mute alarms during daily maintenance"
		configuration["timezone"] = "America/New_York"
		err := component.Execute(core.ExecutionContext{
			Configuration:  configuration,
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})
		require.NoError(t, err)

		body := requestBody(t, httpContext, 0)
		assert.Contains(t, body, "Action=PutAlarmMuteRule")
		assert.Contains(t, body, "Name=DailyMaintenanceWindow")
		assert.Contains(t, body, "Description=Mute+alarms+during+daily+maintenance")
		assert.Contains(t, body, "Rule.Schedule.Expression=cron%280+2+%2A+%2A+%2A%29")
		assert.Contains(t, body, "Rule.Schedule.Duration=PT2H")
		assert.Contains(t, body, "Rule.Schedule.Timezone=America%2FNew_York")
		assert.Contains(t, body, "MuteTargets.AlarmNames.member.1=WebServerCPUAlarm")
		assert.Contains(t, body, "MuteTargets.AlarmNames.member.2=DatabaseConnectionAlarm")
	})

	t.Run("rule creation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body: io.NopCloser(strings.NewReader(`
<ErrorResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <Error>
    <Code>ValidationError</Code>
    <Message>Invalid schedule expression</Message>
  </Error>
</ErrorResponse>`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  validCreateAlarmMuteRuleConfig(),
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to create alarm mute rule")
		require.ErrorContains(t, err, "Invalid schedule expression")
	})

	t.Run("fetch after creation fails -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))},
				{
					StatusCode: http.StatusNotFound,
					Body: io.NopCloser(strings.NewReader(`
<ErrorResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <Error>
    <Code>ResourceNotFoundException</Code>
    <Message>rule not found</Message>
  </Error>
</ErrorResponse>`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  validCreateAlarmMuteRuleConfig(),
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to fetch alarm mute rule after creation")
	})
}
