package ec2

import (
	"io"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateInstance__Setup(t *testing.T) {
	component := &UpdateInstance{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       " ",
				"instance":     "i-abc123",
				"instanceType": "t3.medium",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing instance -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"instance":     "",
				"instanceType": "t3.medium",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance ID is required")
	})

	t.Run("no type and no security groups -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"instance": "i-abc123",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "at least one of instanceType or securityGroup must be set")
	})

	t.Run("stores metadata when config is valid", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeInstanceXML("i-abc123", "running")),
			},
		}
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"instance":     "i-abc123",
				"instanceType": "t3.medium",
			},
			HTTP:        httpCtx,
			Integration: updateIntegration(),
			Metadata:    metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(UpdateInstanceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "i-abc123", stored.InstanceID)
	})
}

func Test__UpdateInstance__Execute_SecurityGroupsOnly(t *testing.T) {
	component := &UpdateInstance{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			// DescribeInstances (check state)
			okResponse(describeInstanceXML("i-abc123", "running")),
			// ModifyInstanceAttribute (security groups)
			okResponse(modifyInstanceAttributeXML()),
			// DescribeInstances (after SG update)
			okResponse(describeInstanceXMLFull("i-abc123", "running", "t3.small")),
		},
	}
	execState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":         "us-east-1",
			"instance":       "i-abc123",
			"securityGroups": "sg-111",
		},
		HTTP:           httpCtx,
		Integration:    updateIntegration(),
		Metadata:       &contexts.MetadataContext{},
		Requests:       &contexts.RequestContext{},
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.True(t, execState.Passed)
	assert.Equal(t, UpdateInstancePayloadType, execState.Type)

	// Verify ModifyInstanceAttribute was called with GroupId params
	body, err := io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "ModifyInstanceAttribute")
	assert.Contains(t, string(body), "GroupId")
}

func Test__UpdateInstance__Execute_NoUpdatesConfigured(t *testing.T) {
	component := &UpdateInstance{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":   "us-east-1",
			"instance": "i-abc123",
		},
		HTTP:           &contexts.HTTPContext{},
		Integration:    updateIntegration(),
		Metadata:       &contexts.MetadataContext{},
		Requests:       &contexts.RequestContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
	})

	require.ErrorContains(t, err, "at least one of instanceType or securityGroup must be set")
}

func Test__UpdateInstance__Execute_SameInstanceTypeSkipsResize(t *testing.T) {
	component := &UpdateInstance{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			okResponse(describeInstanceXMLFull("i-abc123", "running", "t3.large")),
			okResponse(describeInstanceXMLFull("i-abc123", "running", "t3.large")),
		},
	}
	requests := &contexts.RequestContext{}
	execState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":       "us-east-1",
			"instance":     "i-abc123",
			"instanceType": "t3.large",
		},
		HTTP:           httpCtx,
		Integration:    updateIntegration(),
		Metadata:       &contexts.MetadataContext{},
		Requests:       requests,
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.True(t, execState.Passed)
	assert.Equal(t, UpdateInstancePayloadType, execState.Type)
	assert.Empty(t, requests.Action)
	assert.Len(t, httpCtx.Requests, 2)

	for _, req := range httpCtx.Requests {
		body, readErr := io.ReadAll(req.Body)
		require.NoError(t, readErr)
		assert.NotContains(t, string(body), "StopInstances")
		assert.NotContains(t, string(body), "ModifyInstanceAttribute")
	}
}

func Test__UpdateInstance__Execute_TypeChange_WasRunning(t *testing.T) {
	component := &UpdateInstance{}

	t.Run("with restartAfterResize true", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// DescribeInstances (check state = running)
				okResponse(describeInstanceXML("i-abc123", "running")),
				// StopInstances
				okResponse(stopInstancesXML("i-abc123")),
			},
		}
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"instanceType":       "t3.large",
				"restartAfterResize": true,
			},
			HTTP:           httpCtx,
			Integration:    updateIntegration(),
			Metadata:       metadata,
			Requests:       requests,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requests.Action)

		// Metadata should contain phase=stopping and wasRunning=true
		stored, ok := metadata.Get().(UpdateInstanceExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, updateInstancePhaseStopping, stored.Phase)
		assert.True(t, stored.WasRunning)
		assert.Equal(t, "t3.large", stored.NewInstanceType)
	})

	t.Run("defaults restartAfterResize to true when key is missing", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeInstanceXML("i-abc123", "running")),
				okResponse(stopInstancesXML("i-abc123")),
			},
		}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"instance":     "i-abc123",
				"instanceType": "t3.large",
			},
			HTTP:           httpCtx,
			Integration:    updateIntegration(),
			Metadata:       &contexts.MetadataContext{},
			Requests:       requests,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requests.Action)
	})
}

func Test__UpdateInstance__Execute_TypeChange_AlreadyStopped(t *testing.T) {
	component := &UpdateInstance{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			// DescribeInstances (check state = stopped)
			okResponse(describeInstanceXML("i-abc123", "stopped")),
			// ModifyInstanceAttribute
			okResponse(modifyInstanceAttributeXML()),
			// DescribeInstances after modify
			okResponse(describeInstanceXMLFull("i-abc123", "stopped", "t3.large")),
		},
	}
	execState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":             "us-east-1",
			"instance":           "i-abc123",
			"instanceType":       "t3.large",
			"restartAfterResize": false,
		},
		HTTP:           httpCtx,
		Integration:    updateIntegration(),
		Metadata:       &contexts.MetadataContext{},
		Requests:       &contexts.RequestContext{},
		ExecutionState: execState,
	})

	// Instance was already stopped and restart=false, so emit immediately
	require.NoError(t, err)
	assert.True(t, execState.Passed)
	assert.Equal(t, UpdateInstancePayloadType, execState.Type)
}

func Test__UpdateInstance__Poll_StoppedThenModifyAndStart(t *testing.T) {
	component := &UpdateInstance{}

	t.Run("with restartAfterResize true", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// DescribeInstances -> stopped
				okResponse(describeInstanceXML("i-abc123", "stopped")),
				// ModifyInstanceAttribute
				okResponse(modifyInstanceAttributeXML()),
				// StartInstances
				okResponse(startInstancesXML("i-abc123")),
			},
		}
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{
			Metadata: UpdateInstanceExecutionMetadata{
				InstanceID:      "i-abc123",
				NewInstanceType: "t3.large",
				WasRunning:      true,
				Phase:           updateInstancePhaseStopping,
			},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name:           "poll",
			Configuration:  map[string]any{"region": "us-east-1", "restartAfterResize": true},
			HTTP:           httpCtx,
			Integration:    updateIntegration(),
			Metadata:       metadata,
			Requests:       requests,
			ExecutionState: &contexts.ExecutionStateContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		// Moves to starting phase
		stored, ok := metadata.Get().(UpdateInstanceExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, updateInstancePhaseStarting, stored.Phase)
		assert.Equal(t, "poll", requests.Action)
	})

	t.Run("defaults restartAfterResize to true when key is missing", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeInstanceXML("i-abc123", "stopped")),
				okResponse(modifyInstanceAttributeXML()),
				okResponse(startInstancesXML("i-abc123")),
			},
		}
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{
			Metadata: UpdateInstanceExecutionMetadata{
				InstanceID:      "i-abc123",
				NewInstanceType: "t3.large",
				WasRunning:      true,
				Phase:           updateInstancePhaseStopping,
			},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name:           "poll",
			Configuration:  map[string]any{"region": "us-east-1"},
			HTTP:           httpCtx,
			Integration:    updateIntegration(),
			Metadata:       metadata,
			Requests:       requests,
			ExecutionState: &contexts.ExecutionStateContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(UpdateInstanceExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, updateInstancePhaseStarting, stored.Phase)
		assert.Equal(t, "poll", requests.Action)
	})
}

func Test__UpdateInstance__Poll_StartingUsesSeparateAttemptBudget(t *testing.T) {
	component := &UpdateInstance{}
	requests := &contexts.RequestContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name:          "poll",
		Configuration: map[string]any{"region": "us-east-1"},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeInstanceXML("i-abc123", "pending")),
			},
		},
		Integration: updateIntegration(),
		Metadata: &contexts.MetadataContext{
			Metadata: UpdateInstanceExecutionMetadata{
				InstanceID:        "i-abc123",
				NewInstanceType:   "t3.large",
				WasRunning:        true,
				Phase:             updateInstancePhaseStarting,
				StopPollAttempts:  maxInstancePollAttempts,
				StartPollAttempts: 1,
			},
		},
		Requests:       requests,
		ExecutionState: &contexts.ExecutionStateContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	assert.Equal(t, "poll", requests.Action)
}

func Test__UpdateInstance__Poll_RunningEmitsPayload(t *testing.T) {
	component := &UpdateInstance{}
	execState := &contexts.ExecutionStateContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name:          "poll",
		Configuration: map[string]any{"region": "us-east-1"},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeInstanceXMLFull("i-abc123", "running", "t3.large")),
			},
		},
		Integration: updateIntegration(),
		Metadata: &contexts.MetadataContext{
			Metadata: UpdateInstanceExecutionMetadata{
				InstanceID:      "i-abc123",
				NewInstanceType: "t3.large",
				WasRunning:      true,
				Phase:           updateInstancePhaseStarting,
			},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: execState,
		Logger:         logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	assert.True(t, execState.Passed)
	assert.Equal(t, UpdateInstancePayloadType, execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "t3.large", payload["instanceType"])
}

func Test__UpdateInstance__Poll_TerminatedDuringStop_Errors(t *testing.T) {
	component := &UpdateInstance{}

	for _, state := range []string{"terminated", "shutting-down"} {
		t.Run("state="+state, func(t *testing.T) {
			err := component.HandleHook(core.ActionHookContext{
				Name:          "poll",
				Configuration: map[string]any{"region": "us-east-1"},
				HTTP: &contexts.HTTPContext{
					Responses: []*http.Response{
						okResponse(describeInstanceXML("i-abc123", state)),
					},
				},
				Integration: updateIntegration(),
				Metadata: &contexts.MetadataContext{
					Metadata: UpdateInstanceExecutionMetadata{
						InstanceID: "i-abc123",
						Phase:      updateInstancePhaseStopping,
					},
				},
				Requests:       &contexts.RequestContext{},
				ExecutionState: &contexts.ExecutionStateContext{},
				Logger:         logrus.NewEntry(logrus.New()),
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), state)
		})
	}
}

func Test__UpdateInstance__Poll_NonRecoverableStateInStarting_Errors(t *testing.T) {
	component := &UpdateInstance{}

	for _, state := range []string{"terminated", "shutting-down", "stopped", "stopping"} {
		t.Run("state="+state, func(t *testing.T) {
			err := component.HandleHook(core.ActionHookContext{
				Name:          "poll",
				Configuration: map[string]any{"region": "us-east-1"},
				HTTP: &contexts.HTTPContext{
					Responses: []*http.Response{
						okResponse(describeInstanceXML("i-abc123", state)),
					},
				},
				Integration: updateIntegration(),
				Metadata: &contexts.MetadataContext{
					Metadata: UpdateInstanceExecutionMetadata{
						InstanceID: "i-abc123",
						Phase:      updateInstancePhaseStarting,
					},
				},
				Requests:       &contexts.RequestContext{},
				ExecutionState: &contexts.ExecutionStateContext{},
				Logger:         logrus.NewEntry(logrus.New()),
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), state)
		})
	}
}

func Test__UpdateInstance__Cancel(t *testing.T) {
	component := &UpdateInstance{}

	t.Run("does not restart when restartAfterResize is false", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		err := component.Cancel(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"instanceType":       "t3.large",
				"restartAfterResize": false,
			},
			HTTP:        httpCtx,
			Integration: updateIntegration(),
			Metadata: &contexts.MetadataContext{
				Metadata: UpdateInstanceExecutionMetadata{
					InstanceID: "i-abc123",
					WasRunning: true,
					Phase:      updateInstancePhaseStopping,
				},
			},
		})

		require.NoError(t, err)
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("restarts instance when stopping and restartAfterResize is true", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(startInstancesXML("i-abc123")),
			},
		}

		err := component.Cancel(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"instanceType":       "t3.large",
				"restartAfterResize": true,
			},
			HTTP:        httpCtx,
			Integration: updateIntegration(),
			Metadata: &contexts.MetadataContext{
				Metadata: UpdateInstanceExecutionMetadata{
					InstanceID: "i-abc123",
					WasRunning: true,
					Phase:      updateInstancePhaseStopping,
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)

		body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
		require.NoError(t, readErr)
		assert.Contains(t, string(body), "StartInstances")
	})

	t.Run("restarts instance when stopping and restartAfterResize defaults to true", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(startInstancesXML("i-abc123")),
			},
		}

		err := component.Cancel(core.ExecutionContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"instance":     "i-abc123",
				"instanceType": "t3.large",
			},
			HTTP:        httpCtx,
			Integration: updateIntegration(),
			Metadata: &contexts.MetadataContext{
				Metadata: UpdateInstanceExecutionMetadata{
					InstanceID: "i-abc123",
					WasRunning: true,
					Phase:      updateInstancePhaseStopping,
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)

		body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
		require.NoError(t, readErr)
		assert.Contains(t, string(body), "StartInstances")
	})

	t.Run("restarts from node metadata when instance is stopping", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeInstanceXML("i-abc123", "stopping")),
				okResponse(startInstancesXML("i-abc123")),
			},
		}

		err := component.Cancel(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"instanceType":       "t3.large",
				"restartAfterResize": true,
			},
			HTTP:        httpCtx,
			Integration: updateIntegration(),
			Metadata: &contexts.MetadataContext{
				Metadata: UpdateInstanceNodeMetadata{
					Region:     "us-east-1",
					InstanceID: "i-abc123",
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)

		body, readErr := io.ReadAll(httpCtx.Requests[1].Body)
		require.NoError(t, readErr)
		assert.Contains(t, string(body), "StartInstances")
	})

	t.Run("does not restart from node metadata when instance is already stopped", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeInstanceXML("i-abc123", "stopped")),
			},
		}

		err := component.Cancel(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"instanceType":       "t3.large",
				"restartAfterResize": true,
			},
			HTTP:        httpCtx,
			Integration: updateIntegration(),
			Metadata: &contexts.MetadataContext{
				Metadata: UpdateInstanceNodeMetadata{
					Region:     "us-east-1",
					InstanceID: "i-abc123",
				},
			},
		})

		require.NoError(t, err)
		assert.Len(t, httpCtx.Requests, 1)
	})

	t.Run("does not restart during starting phase", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		err := component.Cancel(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"instanceType":       "t3.large",
				"restartAfterResize": true,
			},
			HTTP:        httpCtx,
			Integration: updateIntegration(),
			Metadata: &contexts.MetadataContext{
				Metadata: UpdateInstanceExecutionMetadata{
					InstanceID: "i-abc123",
					WasRunning: true,
					Phase:      updateInstancePhaseStarting,
				},
			},
		})

		require.NoError(t, err)
		assert.Empty(t, httpCtx.Requests)
	})
}

// Helpers

func updateIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		CurrentSecrets: map[string]core.IntegrationSecret{
			"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
			"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
			"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
		},
	}
}

func modifyInstanceAttributeXML() string {
	return `<ModifyInstanceAttributeResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
		<requestId>req-modify</requestId>
		<return>true</return>
	</ModifyInstanceAttributeResponse>`
}

func describeInstanceXMLFull(instanceID, state, instanceType string) string {
	return `<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
		<reservationSet>
			<item>
				<instancesSet>
					<item>
						<instanceId>` + instanceID + `</instanceId>
						<instanceType>` + instanceType + `</instanceType>
						<instanceState><name>` + state + `</name></instanceState>
					</item>
				</instancesSet>
			</item>
		</reservationSet>
	</DescribeInstancesResponse>`
}
