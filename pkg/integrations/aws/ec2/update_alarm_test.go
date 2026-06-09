package ec2

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

func Test__UpdateAlarm__Setup(t *testing.T) {
	component := &UpdateAlarm{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": " ",
				"alarm":  "HighCPU",
				"thresholdCondition": map[string]any{
					"threshold":          90.0,
					"comparisonOperator": "GreaterThanThreshold",
				},
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing alarm name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  " ",
				"thresholdCondition": map[string]any{
					"threshold":          90.0,
					"comparisonOperator": "GreaterThanThreshold",
				},
			},
		})
		require.ErrorContains(t, err, "alarm name is required")
	})

	t.Run("no update fields -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "HighCPU",
			},
		})
		require.ErrorContains(t, err, "at least one alarm property to update is required")
	})

	t.Run("threshold condition missing threshold -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "HighCPU",
				"thresholdCondition": map[string]any{
					"comparisonOperator": "GreaterThanThreshold",
				},
			},
		})
		require.ErrorContains(t, err, "threshold is required")
	})

	t.Run("valid configuration -> stores updated fields in metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "HighCPU",
				"thresholdCondition": map[string]any{
					"threshold":          90.0,
					"comparisonOperator": "GreaterThanThreshold",
				},
				"statistic": "Average",
			},
			Metadata: metadata,
		})
		require.NoError(t, err)

		stored, ok := metadata.Get().(UpdateAlarmNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, []string{"Threshold", "Statistic"}, stored.UpdatedFields)
	})
}

func Test__UpdateAlarm__Execute(t *testing.T) {
	component := &UpdateAlarm{}

	t.Run("updates alarm and emits alarm details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsXML)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsXML)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "HighCPU",
				"thresholdCondition": map[string]any{
					"threshold":          90.0,
					"comparisonOperator": "GreaterThanThreshold",
				},
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, UpdateAlarmPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "HighCPU", data["alarmName"])
		assert.Equal(t, float64(80), data["threshold"])
	})
}
