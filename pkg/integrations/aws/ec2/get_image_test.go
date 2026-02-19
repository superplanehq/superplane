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

func Test__GetImage__Setup(t *testing.T) {
	component := &GetImage{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid"})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"region":  " ",
			"imageId": "ami-123",
		}})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing image ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"region": "us-east-1",
		}})
		require.ErrorContains(t, err, "image ID is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"region":  "us-east-1",
			"imageId": "ami-123",
		}})
		require.NoError(t, err)
	})
}

func Test__GetImage__Execute(t *testing.T) {
	component := &GetImage{}

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
				"region":  "us-east-1",
				"imageId": "ami-123",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits image detail", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DescribeImagesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<requestId>req-123</requestId>
							<imagesSet>
								<item>
									<imageId>ami-123</imageId>
									<name>my-ami</name>
									<description>test image</description>
									<imageState>available</imageState>
									<creationDate>2026-02-18T12:00:00.000Z</creationDate>
									<ownerId>123456789012</ownerId>
									<architecture>x86_64</architecture>
									<imageType>machine</imageType>
									<rootDeviceType>ebs</rootDeviceType>
									<rootDeviceName>/dev/xvda</rootDeviceName>
									<virtualizationType>hvm</virtualizationType>
									<hypervisor>xen</hypervisor>
								</item>
							</imagesSet>
						</DescribeImagesResponse>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"imageId": "ami-123",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
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
		image, ok := output["image"].(*Image)
		require.True(t, ok)
		assert.Equal(t, "ami-123", image.ImageID)
		assert.Equal(t, "my-ami", image.Name)
		assert.Equal(t, "available", image.State)
		assert.Equal(t, "us-east-1", image.Region)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://ec2.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
		bodyBytes, readErr := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, readErr)
		requestBody := string(bodyBytes)
		assert.Contains(t, requestBody, "Action=DescribeImages")
		assert.Contains(t, requestBody, "ImageId.1=ami-123")
	})
}
