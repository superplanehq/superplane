package ec2

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteInstance__Setup(t *testing.T) {
	component := &DeleteInstance{}

	t.Run("missing instance id -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"instance": "",
			},
		})

		require.ErrorContains(t, err, "instance ID is required")
	})

	t.Run("stores instance name in node metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<reservationSet>
								<item>
									<instancesSet>
										<item>
											<instanceId>i-abc123</instanceId>
											<instanceState><name>running</name></instanceState>
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
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"instance": "i-abc123",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
			Metadata: metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(DeleteInstanceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "my-instance", stored.InstanceName)
	})
}

func Test__DeleteInstance__Execute(t *testing.T) {
	component := &DeleteInstance{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<TerminateInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-123</requestId>
						<instancesSet>
							<item>
								<instanceId>i-abc123</instanceId>
								<currentState><name>shutting-down</name></currentState>
							</item>
						</instancesSet>
					</TerminateInstancesResponse>
				`)),
			},
		},
	}
	metadata := &contexts.MetadataContext{}
	requests := &contexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":   "us-east-1",
			"instance": "i-abc123",
		},
		HTTP:     httpContext,
		Metadata: metadata,
		Requests: requests,
		Integration: &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Action=TerminateInstances")
	assert.Contains(t, string(body), "InstanceId.1=i-abc123")
	assert.Equal(t, "poll", requests.Action)
	assert.Equal(t, DeleteInstanceExecutionMetadata{InstanceID: "i-abc123"}, metadata.Metadata)
}

func Test__DeleteInstance__PollEmitsWhenTerminated(t *testing.T) {
	component := &DeleteInstance{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<reservationSet>
							<item>
								<instancesSet>
									<item>
										<instanceId>i-abc123</instanceId>
										<instanceState><name>terminated</name></instanceState>
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

	err := component.HandleHook(core.ActionHookContext{
		Name: "poll",
		Configuration: map[string]any{
			"region": "us-east-1",
		},
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		},
		Metadata: &contexts.MetadataContext{
			Metadata: DeleteInstanceExecutionMetadata{InstanceID: "i-abc123"},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, DeleteInstancePayloadType, executionState.Type)
}

func Test__DeleteInstance__Execute_InstanceAlreadyGone(t *testing.T) {
	component := &DeleteInstance{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Body: io.NopCloser(strings.NewReader(`
					<Response>
						<Errors>
							<Error>
								<Code>InvalidInstanceID.NotFound</Code>
								<Message>The instance ID 'i-abc123' does not exist</Message>
							</Error>
						</Errors>
					</Response>
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
		Metadata:       &contexts.MetadataContext{},
		Requests:       &contexts.RequestContext{},
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
	assert.Equal(t, DeleteInstancePayloadType, executionState.Type)
}

func Test__DeleteInstance__PollTreatsNotFoundAsTerminated(t *testing.T) {
	component := &DeleteInstance{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Body: io.NopCloser(strings.NewReader(`
					<Response>
						<Errors>
							<Error>
								<Code>InvalidInstanceID.NotFound</Code>
								<Message>The instance ID 'i-abc123' does not exist</Message>
							</Error>
						</Errors>
					</Response>
				`)),
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name: "poll",
		Configuration: map[string]any{
			"region": "us-east-1",
		},
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		},
		Metadata: &contexts.MetadataContext{
			Metadata: DeleteInstanceExecutionMetadata{InstanceID: "i-abc123"},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, DeleteInstancePayloadType, executionState.Type)
}
