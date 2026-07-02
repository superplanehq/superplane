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

func Test__ManageInstancePower__Setup(t *testing.T) {
	component := &ManageInstancePower{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    " ",
				"instance":  "i-abc123",
				"operation": "start",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing instance id -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"instance":  "",
				"operation": "start",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance ID is required")
	})

	t.Run("invalid operation -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"instance":  "i-abc123",
				"operation": "nuke",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid operation")
	})

	t.Run("stores instance name in node metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(`
					<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<reservationSet>
							<item>
								<instancesSet>
									<item>
										<instanceId>i-abc123</instanceId>
										<instanceState><name>running</name></instanceState>
										<tagSet>
											<item><key>Name</key><value>my-server</value></item>
										</tagSet>
									</item>
								</instancesSet>
							</item>
						</reservationSet>
					</DescribeInstancesResponse>`),
			},
		}
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"instance":  "i-abc123",
				"operation": "stop",
			},
			HTTP:        httpContext,
			Integration: manageInstancePowerIntegration(),
			Metadata:    metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(ManageInstancePowerNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "my-server", stored.InstanceName)
	})
}

func Test__ManageInstancePower__Execute_Start(t *testing.T) {
	component := &ManageInstancePower{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{okResponse(startInstancesXML("i-abc123"))},
	}
	metadata := &contexts.MetadataContext{}
	requests := &contexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":    "us-east-1",
			"instance":  "i-abc123",
			"operation": "start",
		},
		HTTP:           httpContext,
		Metadata:       metadata,
		Requests:       requests,
		Integration:    manageInstancePowerIntegration(),
		ExecutionState: &contexts.ExecutionStateContext{},
	})

	require.NoError(t, err)
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Action=StartInstances")
	assert.Contains(t, string(body), "InstanceId.1=i-abc123")
	assert.Equal(t, "poll", requests.Action)
	assert.Equal(t, ManageInstancePowerExecutionMetadata{
		InstanceID: "i-abc123",
		Operation:  "start",
	}, metadata.Metadata)
}

func Test__ManageInstancePower__Execute_Stop(t *testing.T) {
	component := &ManageInstancePower{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{okResponse(stopInstancesXML("i-abc123"))},
	}
	metadata := &contexts.MetadataContext{}
	requests := &contexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":    "us-east-1",
			"instance":  "i-abc123",
			"operation": "stop",
		},
		HTTP:           httpContext,
		Metadata:       metadata,
		Requests:       requests,
		Integration:    manageInstancePowerIntegration(),
		ExecutionState: &contexts.ExecutionStateContext{},
	})

	require.NoError(t, err)
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Action=StopInstances")
	assert.NotContains(t, string(body), "Hibernate=true")
	assert.Equal(t, "poll", requests.Action)
	assert.Equal(t, ManageInstancePowerExecutionMetadata{
		InstanceID: "i-abc123",
		Operation:  "stop",
	}, metadata.Metadata)
}

func Test__ManageInstancePower__Execute_Hibernate(t *testing.T) {
	component := &ManageInstancePower{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{okResponse(stopInstancesXML("i-abc123"))},
	}
	metadata := &contexts.MetadataContext{}
	requests := &contexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":    "us-east-1",
			"instance":  "i-abc123",
			"operation": "hibernate",
		},
		HTTP:           httpContext,
		Metadata:       metadata,
		Requests:       requests,
		Integration:    manageInstancePowerIntegration(),
		ExecutionState: &contexts.ExecutionStateContext{},
	})

	require.NoError(t, err)
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Action=StopInstances")
	assert.Contains(t, string(body), "Hibernate=true")
	assert.Equal(t, "poll", requests.Action)
	assert.Equal(t, ManageInstancePowerExecutionMetadata{
		InstanceID: "i-abc123",
		Operation:  "hibernate",
	}, metadata.Metadata)
}

func Test__ManageInstancePower__Execute_Reboot(t *testing.T) {
	component := &ManageInstancePower{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			okResponse(rebootInstancesXML()),
			okResponse(describeInstanceXML("i-abc123", "running")),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":    "us-east-1",
			"instance":  "i-abc123",
			"operation": "reboot",
		},
		HTTP:           httpContext,
		Metadata:       &contexts.MetadataContext{},
		Requests:       &contexts.RequestContext{},
		Integration:    manageInstancePowerIntegration(),
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 2)
	rebootBody, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(rebootBody), "Action=RebootInstances")
	assert.True(t, executionState.Passed)
	assert.Equal(t, ManageInstancePowerRebootPayloadType, executionState.Type)
}

func Test__ManageInstancePower__PollStart_EmitsWhenRunning(t *testing.T) {
	component := &ManageInstancePower{}
	ctx := managePowerHookCtx("start", "running")
	executionState := ctx.ExecutionState.(*contexts.ExecutionStateContext)

	require.NoError(t, component.HandleHook(ctx))
	assert.True(t, executionState.Passed)
	assert.Equal(t, ManageInstancePowerStartPayloadType, executionState.Type)
}

func Test__ManageInstancePower__PollStart_ReschedulesWhenPending(t *testing.T) {
	component := &ManageInstancePower{}
	ctx := managePowerHookCtx("start", "pending")
	requests := ctx.Requests.(*contexts.RequestContext)

	require.NoError(t, component.HandleHook(ctx))
	assert.Equal(t, "poll", requests.Action)
}

func Test__ManageInstancePower__PollStart_FailsImmediatelyOnNonRecoverableState(t *testing.T) {
	for _, state := range []string{"terminated", "shutting-down", "stopped", "stopping"} {
		t.Run(state, func(t *testing.T) {
			component := &ManageInstancePower{}
			ctx := managePowerHookCtx("start", state)

			err := component.HandleHook(ctx)
			require.ErrorContains(t, err, "will not reach running without intervention")
		})
	}
}

func Test__ManageInstancePower__PollStop_EmitsWhenStopped(t *testing.T) {
	component := &ManageInstancePower{}
	ctx := managePowerHookCtx("stop", "stopped")
	executionState := ctx.ExecutionState.(*contexts.ExecutionStateContext)

	require.NoError(t, component.HandleHook(ctx))
	assert.True(t, executionState.Passed)
	assert.Equal(t, ManageInstancePowerStopPayloadType, executionState.Type)
}

func Test__ManageInstancePower__PollStop_ReschedulesWhenStopping(t *testing.T) {
	component := &ManageInstancePower{}
	ctx := managePowerHookCtx("stop", "stopping")
	requests := ctx.Requests.(*contexts.RequestContext)

	require.NoError(t, component.HandleHook(ctx))
	assert.Equal(t, "poll", requests.Action)
}

func Test__ManageInstancePower__PollStop_ErrorsWhenTerminated(t *testing.T) {
	component := &ManageInstancePower{}
	ctx := managePowerHookCtx("stop", "terminated")

	err := component.HandleHook(ctx)
	require.ErrorContains(t, err, "unexpectedly")
}

func Test__ManageInstancePower__PollHibernate_EmitsHibernatePayloadWhenStopped(t *testing.T) {
	component := &ManageInstancePower{}
	ctx := managePowerHookCtx("hibernate", "stopped")
	executionState := ctx.ExecutionState.(*contexts.ExecutionStateContext)

	require.NoError(t, component.HandleHook(ctx))
	assert.True(t, executionState.Passed)
	assert.Equal(t, ManageInstancePowerHibernatePayloadType, executionState.Type)
}

func Test__ManageInstancePower__Poll_NoopWhenAlreadyFinished(t *testing.T) {
	component := &ManageInstancePower{}
	httpContext := &contexts.HTTPContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name:          "poll",
		Configuration: map[string]any{"region": "us-east-1"},
		HTTP:          httpContext,
		Integration:   manageInstancePowerIntegration(),
		Metadata: &contexts.MetadataContext{
			Metadata: ManageInstancePowerExecutionMetadata{
				InstanceID: "i-abc123",
				Operation:  "start",
			},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: &contexts.ExecutionStateContext{Finished: true},
		Logger:         logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	assert.Empty(t, httpContext.Requests)
}

func Test__ManageInstancePower__PollStart_TimesOutAfterMaxAttempts(t *testing.T) {
	component := &ManageInstancePower{}
	ctx := core.ActionHookContext{
		Name:          "poll",
		Configuration: map[string]any{"region": "us-east-1"},
		HTTP:          &contexts.HTTPContext{Responses: []*http.Response{okResponse(describeInstanceXML("i-abc123", "pending"))}},
		Integration:   manageInstancePowerIntegration(),
		Metadata: &contexts.MetadataContext{
			Metadata: ManageInstancePowerExecutionMetadata{
				InstanceID:   "i-abc123",
				Operation:    "start",
				PollAttempts: maxInstancePollAttempts,
			},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := component.HandleHook(ctx)
	require.ErrorContains(t, err, "timed out")
}

func Test__ManageInstancePower__PollStop_TimesOutAfterMaxAttempts(t *testing.T) {
	component := &ManageInstancePower{}
	ctx := core.ActionHookContext{
		Name:          "poll",
		Configuration: map[string]any{"region": "us-east-1"},
		HTTP:          &contexts.HTTPContext{Responses: []*http.Response{okResponse(describeInstanceXML("i-abc123", "stopping"))}},
		Integration:   manageInstancePowerIntegration(),
		Metadata: &contexts.MetadataContext{
			Metadata: ManageInstancePowerExecutionMetadata{
				InstanceID:   "i-abc123",
				Operation:    "stop",
				PollAttempts: maxInstancePollAttempts,
			},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := component.HandleHook(ctx)
	require.ErrorContains(t, err, "timed out")
}

func manageInstancePowerIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		CurrentSecrets: map[string]core.IntegrationSecret{
			"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
			"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
			"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
		},
	}
}

func describeInstanceXML(instanceID, state string) string {
	return `
		<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<reservationSet>
				<item>
					<instancesSet>
						<item>
							<instanceId>` + instanceID + `</instanceId>
							<instanceState><name>` + state + `</name></instanceState>
						</item>
					</instancesSet>
				</item>
			</reservationSet>
		</DescribeInstancesResponse>`
}

func startInstancesXML(instanceID string) string {
	return `
		<StartInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<requestId>req-start</requestId>
			<instancesSet>
				<item>
					<instanceId>` + instanceID + `</instanceId>
					<currentState><name>pending</name></currentState>
				</item>
			</instancesSet>
		</StartInstancesResponse>`
}

func stopInstancesXML(instanceID string) string {
	return `
		<StopInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<requestId>req-stop</requestId>
			<instancesSet>
				<item>
					<instanceId>` + instanceID + `</instanceId>
					<currentState><name>stopping</name></currentState>
				</item>
			</instancesSet>
		</StopInstancesResponse>`
}

func rebootInstancesXML() string {
	return `<RebootInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><requestId>req-reboot</requestId><return>true</return></RebootInstancesResponse>`
}

func okResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func managePowerHookCtx(operation, state string, extraResponses ...*http.Response) core.ActionHookContext {
	responses := extraResponses
	if len(responses) == 0 {
		responses = []*http.Response{okResponse(describeInstanceXML("i-abc123", state))}
	}

	return core.ActionHookContext{
		Name:          "poll",
		Configuration: map[string]any{"region": "us-east-1"},
		HTTP:          &contexts.HTTPContext{Responses: responses},
		Integration:   manageInstancePowerIntegration(),
		Metadata: &contexts.MetadataContext{
			Metadata: ManageInstancePowerExecutionMetadata{
				InstanceID: "i-abc123",
				Operation:  operation,
			},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}
}
