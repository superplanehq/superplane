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

func Test__CopyImage__Setup(t *testing.T) {
	component := &CopyImage{}

	t.Run("missing source region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":        "us-west-2",
				"sourceImageId": "ami-123",
				"name":          "my-copy",
			},
		})

		require.ErrorContains(t, err, "source region is required")
	})
}

func Test__CopyImage__Execute(t *testing.T) {
	component := &CopyImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<CopyImageResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-copy</requestId>
						<imageId>ami-copy-123</imageId>
					</CopyImageResponse>
				`)),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":        "us-west-2",
			"sourceRegion":  "us-east-1",
			"sourceImageId": "ami-source-123",
			"name":          "my-copy",
			"description":   "copy for west region",
		},
		HTTP:           httpContext,
		ExecutionState: execState,
		Integration:    integrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	assert.Equal(t, "aws.ec2.image.copied", execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(*CopyImageOutput)
	require.True(t, ok)
	assert.Equal(t, "ami-copy-123", output.ImageID)
	assert.Equal(t, "ami-source-123", output.SourceImageID)
	assert.Equal(t, "us-east-1", output.SourceRegion)
	assert.Equal(t, "us-west-2", output.Region)
	assert.Equal(t, ImageStatePending, output.State)

	require.Len(t, httpContext.Requests, 1)
	requestBody := requestBodyString(t, httpContext.Requests[0])
	assert.Contains(t, requestBody, "Action=CopyImage")
	assert.Contains(t, requestBody, "SourceImageId=ami-source-123")
	assert.Contains(t, requestBody, "SourceRegion=us-east-1")
	assert.Contains(t, requestBody, "Name=my-copy")
}

func Test__DeregisterImage__Execute(t *testing.T) {
	component := &DeregisterImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DeregisterImageResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-deregister</requestId>
						<return>true</return>
					</DeregisterImageResponse>
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
		Integration:    integrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	assert.Equal(t, "aws.ec2.image.deregistered", execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(DeregisterImageOutput)
	require.True(t, ok)
	assert.Equal(t, "ami-123", output.ImageID)
	assert.True(t, output.Deregistered)

	require.Len(t, httpContext.Requests, 1)
	requestBody := requestBodyString(t, httpContext.Requests[0])
	assert.Contains(t, requestBody, "Action=DeregisterImage")
	assert.Contains(t, requestBody, "ImageId=ami-123")
}

func Test__EnableImage__Execute(t *testing.T) {
	component := &EnableImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<EnableImageResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-enable</requestId>
						<return>true</return>
					</EnableImageResponse>
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
		Integration:    integrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	assert.Equal(t, "aws.ec2.image.enabled", execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(EnableImageOutput)
	require.True(t, ok)
	assert.Equal(t, "ami-123", output.ImageID)
	assert.True(t, output.Enabled)
}

func Test__DisableImage__Execute(t *testing.T) {
	component := &DisableImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DisableImageResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-disable</requestId>
						<return>true</return>
					</DisableImageResponse>
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
		Integration:    integrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	assert.Equal(t, "aws.ec2.image.disabled", execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(DisableImageOutput)
	require.True(t, ok)
	assert.Equal(t, "ami-123", output.ImageID)
	assert.True(t, output.Disabled)
}

func Test__EnableImageDeprecation__Setup(t *testing.T) {
	component := &EnableImageDeprecation{}

	t.Run("invalid deprecateAt -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"imageId":     "ami-123",
				"deprecateAt": "not-a-time",
			},
		})

		require.ErrorContains(t, err, "deprecateAt must be a valid RFC3339 timestamp")
	})
}

func Test__EnableImageDeprecation__Execute(t *testing.T) {
	component := &EnableImageDeprecation{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<EnableImageDeprecationResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-enable-deprecation</requestId>
						<return>true</return>
					</EnableImageDeprecationResponse>
				`)),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":      "us-east-1",
			"imageId":     "ami-123",
			"deprecateAt": "2026-03-01T00:00:00Z",
		},
		HTTP:           httpContext,
		ExecutionState: execState,
		Integration:    integrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	assert.Equal(t, "aws.ec2.image.deprecation.enabled", execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(EnableImageDeprecationResult)
	require.True(t, ok)
	assert.Equal(t, "ami-123", output.ImageID)
	assert.Equal(t, "2026-03-01T00:00:00Z", output.DeprecateAt)
	assert.True(t, output.DeprecationEnabled)

	require.Len(t, httpContext.Requests, 1)
	requestBody := requestBodyString(t, httpContext.Requests[0])
	assert.Contains(t, requestBody, "Action=EnableImageDeprecation")
	assert.Contains(t, requestBody, "ImageId=ami-123")
	assert.Contains(t, requestBody, "DeprecateAt=2026-03-01T00%3A00%3A00Z")
}

func Test__DisableImageDeprecation__Execute(t *testing.T) {
	component := &DisableImageDeprecation{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DisableImageDeprecationResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-disable-deprecation</requestId>
						<return>true</return>
					</DisableImageDeprecationResponse>
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
		Integration:    integrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	assert.Equal(t, "aws.ec2.image.deprecation.disabled", execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(DisableImageDeprecationOutput)
	require.True(t, ok)
	assert.Equal(t, "ami-123", output.ImageID)
	assert.False(t, output.DeprecationEnabled)
}

func integrationWithCredentials() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Secrets: map[string]core.IntegrationSecret{
			"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
			"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
			"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
		},
	}
}

func requestBodyString(t *testing.T, request *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	return string(body)
}
