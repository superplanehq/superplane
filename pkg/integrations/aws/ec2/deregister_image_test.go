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

func Test__DeregisterImage__ExecuteWithoutDeletingSnapshot(t *testing.T) {
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
		Integration:    testIntegrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	assert.Equal(t, "aws.ec2.image.deregistered", execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(map[string]any)
	require.True(t, ok)
	image, ok := output["image"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ami-123", image["imageId"])
	assert.Equal(t, "req-deregister", output["requestId"])

	require.Len(t, httpContext.Requests, 1)
	requestBody := testRequestBodyString(t, httpContext.Requests[0])
	assert.Contains(t, requestBody, "Action=DeregisterImage")
	assert.Contains(t, requestBody, "ImageId=ami-123")
	assert.NotContains(t, requestBody, "Action=DeleteSnapshot")
}

func Test__DeregisterImage__ExecuteDeletingSnapshot(t *testing.T) {
	component := &DeregisterImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DescribeImagesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-describe</requestId>
						<imagesSet>
							<item>
								<imageId>ami-123</imageId>
								<blockDeviceMapping>
									<item>
										<deviceName>/dev/xvda</deviceName>
										<ebs>
											<snapshotId>snap-123</snapshotId>
										</ebs>
									</item>
								</blockDeviceMapping>
							</item>
						</imagesSet>
					</DescribeImagesResponse>
				`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DeregisterImageResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-deregister</requestId>
						<return>true</return>
					</DeregisterImageResponse>
				`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DeleteSnapshotResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-delete-snapshot</requestId>
						<return>true</return>
					</DeleteSnapshotResponse>
				`)),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":          "us-east-1",
			"imageId":         "ami-123",
			"deleteSnapshots": true,
		},
		HTTP:           httpContext,
		ExecutionState: execState,
		Integration:    testIntegrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	assert.Equal(t, "aws.ec2.image.deregistered", execState.Type)
	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(map[string]any)
	require.True(t, ok)
	image, ok := output["image"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ami-123", image["imageId"])
	assert.Equal(t, "req-deregister", output["requestId"])
	deletedSnapshots, ok := output["deletedSnapshots"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"snap-123"}, deletedSnapshots)

	require.Len(t, httpContext.Requests, 3)

	describeRequestBody := testRequestBodyString(t, httpContext.Requests[0])
	assert.Contains(t, describeRequestBody, "Action=DescribeImages")
	assert.Contains(t, describeRequestBody, "ImageId.1=ami-123")

	deregisterRequestBody := testRequestBodyString(t, httpContext.Requests[1])
	assert.Contains(t, deregisterRequestBody, "Action=DeregisterImage")
	assert.Contains(t, deregisterRequestBody, "ImageId=ami-123")

	deleteSnapshotRequestBody := testRequestBodyString(t, httpContext.Requests[2])
	assert.Contains(t, deleteSnapshotRequestBody, "Action=DeleteSnapshot")
	assert.Contains(t, deleteSnapshotRequestBody, "SnapshotId=snap-123")
}
