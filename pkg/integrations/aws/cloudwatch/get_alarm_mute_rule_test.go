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

func validGetAlarmMuteRuleConfig() map[string]any {
	return map[string]any{
		"region": "us-east-1",
		"name":   "DailyMaintenanceWindow",
	}
}

func Test__GetAlarmMuteRule__Setup(t *testing.T) {
	component := &GetAlarmMuteRule{}

	t.Run("missing region -> error", func(t *testing.T) {
		configuration := validGetAlarmMuteRuleConfig()
		configuration["region"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing name -> error", func(t *testing.T) {
		configuration := validGetAlarmMuteRuleConfig()
		configuration["name"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: validGetAlarmMuteRuleConfig(),
			Metadata:      metadata,
		})
		require.NoError(t, err)

		nodeMetadata, ok := metadata.Metadata.(GetAlarmMuteRuleNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", nodeMetadata.Region)
		assert.Equal(t, "DailyMaintenanceWindow", nodeMetadata.Name)
	})
}

func Test__GetAlarmMuteRule__Execute(t *testing.T) {
	component := &GetAlarmMuteRule{}

	t.Run("fetches and emits the rule details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(getAlarmMuteRuleXML))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:  validGetAlarmMuteRuleConfig(),
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    awsIntegrationContext(),
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, GetAlarmMuteRulePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "DailyMaintenanceWindow", data["name"])
		assert.Equal(t, "arn:aws:cloudwatch:us-east-1:123456789012:alarm-mute-rule:DailyMaintenanceWindow", data["alarmMuteRuleArn"])
		assert.Equal(t, "cron(0 2 * * *)", data["scheduleExpression"])
		assert.Equal(t, "PT2H", data["scheduleDuration"])
		assert.Equal(t, "UTC", data["scheduleTimezone"])
		assert.Equal(t, []string{"WebServerCPUAlarm", "DatabaseConnectionAlarm"}, data["alarmNames"])
		assert.Equal(t, "SCHEDULED", data["status"])
		assert.Equal(t, "RECURRING", data["muteType"])
	})

	t.Run("requests the rule by name", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(getAlarmMuteRuleXML))},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  validGetAlarmMuteRuleConfig(),
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})
		require.NoError(t, err)

		body := requestBody(t, httpContext, 0)
		assert.Contains(t, body, "Action=GetAlarmMuteRule")
		assert.Contains(t, body, "AlarmMuteRuleName=DailyMaintenanceWindow")
	})

	t.Run("rule not found -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
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
			Configuration:  validGetAlarmMuteRuleConfig(),
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to get alarm mute rule")
		require.ErrorContains(t, err, "rule not found")
	})
}
