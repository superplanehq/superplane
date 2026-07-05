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

func Test__RunInstance__Setup(t *testing.T) {
	component := &RunInstance{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid"})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"region":       " ",
			"imageId":      "ami-123",
			"instanceType": "t3.micro",
		}})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing image ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"region":       "us-east-1",
			"instanceType": "t3.micro",
		}})
		require.ErrorContains(t, err, "image ID is required")
	})

	t.Run("missing instance type -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"region":  "us-east-1",
			"imageId": "ami-123",
		}})
		require.ErrorContains(t, err, "instance type is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"region":       "us-east-1",
			"imageId":      "ami-123",
			"instanceType": "t3.micro",
		}})
		require.NoError(t, err)
	})
}

func Test__RunInstance__Execute(t *testing.T) {
	component := &RunInstance{}

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
				"region":       "us-east-1",
				"imageId":      "ami-123",
				"instanceType": "t3.micro",
			},
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})
		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits instance detail", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<RunInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<requestId>req-123</requestId>
							<instancesSet>
								<item>
									<instanceId>i-1234567890abcdef0</instanceId>
									<instanceType>t3.micro</instanceType>
									<instanceState>
										<name>pending</name>
									</instanceState>
									<tagSet/>
								</item>
							</instancesSet>
						</RunInstancesResponse>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":          "us-east-1",
				"imageId":         "ami-123",
				"instanceType":    "t3.micro",
				"securityGroupId": "sg-abc123",
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
		instance, ok := output["instance"].(*RunInstanceOutput)
		require.True(t, ok)
		assert.Equal(t, "i-1234567890abcdef0", instance.InstanceID)
		assert.Equal(t, "t3.micro", instance.InstanceType)
		assert.Equal(t, "pending", instance.State)
		assert.Equal(t, "us-east-1", instance.Region)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://ec2.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
		bodyBytes, readErr := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, readErr)
		requestBody := string(bodyBytes)
		assert.Contains(t, requestBody, "Action=RunInstances")
		assert.Contains(t, requestBody, "ImageId=ami-123")
		assert.Contains(t, requestBody, "InstanceType=t3.micro")
		assert.Contains(t, requestBody, "SecurityGroupId.1=sg-abc123")
	})
}
