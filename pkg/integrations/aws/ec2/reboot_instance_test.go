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

func Test__RebootInstance__Setup(t *testing.T) {
	component := &RebootInstance{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid"})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"region":     " ",
			"instanceId": "i-123",
		}})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing instance ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"region": "us-east-1",
		}})
		require.ErrorContains(t, err, "instance ID is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"region":     "us-east-1",
			"instanceId": "i-1234567890abcdef0",
		}})
		require.NoError(t, err)
	})
}

func Test__RebootInstance__Execute(t *testing.T) {
	component := &RebootInstance{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"instanceId": "i-1234567890abcdef0",
			},
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})
		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits reboot confirmation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<RebootInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<requestId>req-123</requestId>
							<return>true</return>
						</RebootInstancesResponse>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"instanceId": "i-1234567890abcdef0",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"]
		output, ok := payload.(map[string]any)
		require.True(t, ok)
		reboot, ok := output["instanceReboot"].(*RebootInstanceOutput)
		require.True(t, ok)
		assert.Equal(t, "i-1234567890abcdef0", reboot.InstanceID)
		assert.Equal(t, "us-east-1", reboot.Region)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://ec2.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
		bodyBytes, readErr := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, readErr)
		requestBody := string(bodyBytes)
		assert.Contains(t, requestBody, "Action=RebootInstances")
		assert.Contains(t, requestBody, "InstanceId.1=i-1234567890abcdef0")
	})
}
