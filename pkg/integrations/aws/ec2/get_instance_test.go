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

func Test__GetInstance__Setup(t *testing.T) {
	component := &GetInstance{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   " ",
				"instance": "i-abc123",
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing instance -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"instance": " ",
			},
		})
		require.ErrorContains(t, err, "instance ID is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"instance": "i-abc123",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__GetInstance__Execute(t *testing.T) {
	component := &GetInstance{}

	t.Run("describe instance -> emits instance details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<requestId>req-123</requestId>
							<reservationSet>
								<item>
									<instancesSet>
										<item>
											<instanceId>i-abc123</instanceId>
											<instanceType>t3.micro</instanceType>
											<imageId>ami-xyz</imageId>
											<instanceState><name>running</name></instanceState>
											<privateIpAddress>10.0.0.5</privateIpAddress>
											<ipAddress>52.1.2.3</ipAddress>
											<dnsName>ec2-52-1-2-3.compute-1.amazonaws.com</dnsName>
											<privateDnsName>ip-10-0-0-5.ec2.internal</privateDnsName>
											<subnetId>subnet-abc</subnetId>
											<vpcId>vpc-abc</vpcId>
											<tagSet>
												<item><key>Name</key><value>my-instance</value></item>
											</tagSet>
										</item>
									</instancesSet>
								</item>
							</reservationSet>
						</DescribeInstancesResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"instance": "i-abc123",
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
		assert.Equal(t, GetInstancePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "i-abc123", data["instanceId"])
		assert.Equal(t, "running", data["state"])
		assert.Equal(t, "t3.micro", data["instanceType"])
		assert.Equal(t, "my-instance", data["name"])
		assert.Equal(t, "52.1.2.3", data["publicIpAddress"])
	})
}
