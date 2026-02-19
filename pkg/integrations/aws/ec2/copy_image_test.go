package ec2

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CopyImage__Setup(t *testing.T) {
	component := &CopyImage{}

	t.Run("missing source region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata:    &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{},
			Configuration: map[string]any{
				"region":        "us-west-2",
				"sourceImageId": "ami-123",
				"name":          "my-copy",
			},
		})

		require.ErrorContains(t, err, "source region is required")
	})

	t.Run("rule missing -> schedules provisioning", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		integration := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{},
				},
			},
		}

		err := component.Setup(core.SetupContext{
			Logger: log.NewEntry(log.New()),
			Configuration: map[string]any{
				"region":        "us-west-2",
				"sourceRegion":  "us-east-1",
				"sourceImageId": "ami-source-123",
				"name":          "my-copy",
			},
			Metadata:    metadata,
			Requests:    requests,
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integration.ActionRequests[0].ActionName)
		params := integration.ActionRequests[0].Parameters.(common.ProvisionRuleParameters)
		assert.Equal(t, "us-west-2", params.Region)
		assert.Equal(t, Source, params.Source)
		assert.Equal(t, DetailTypeAMIStateChange, params.DetailType)
		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, copyImageInitialRuleAvailabilityTimeout, requests.Duration)

		stored, ok := metadata.Get().(CopyImageNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-west-2", stored.Region)
		assert.Empty(t, stored.SubscriptionID)
	})

	t.Run("rule exists -> subscribes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integration := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.ec2:us-west-2": {
							Source:      Source,
							DetailTypes: []string{DetailTypeAMIStateChange},
						},
					},
				},
			},
		}

		err := component.Setup(core.SetupContext{
			Logger: log.NewEntry(log.New()),
			Configuration: map[string]any{
				"region":        "us-west-2",
				"sourceRegion":  "us-east-1",
				"sourceImageId": "ami-source-123",
				"name":          "my-copy",
			},
			Metadata:    metadata,
			Requests:    &contexts.RequestContext{},
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.Subscriptions, 1)
		stored, ok := metadata.Get().(CopyImageNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-west-2", stored.Region)
		assert.NotEmpty(t, stored.SubscriptionID)
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
	metadata := &contexts.MetadataContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":        "us-west-2",
			"sourceRegion":  "us-east-1",
			"sourceImageId": "ami-source-123",
			"name":          "my-copy",
			"description":   "copy for west region",
		},
		HTTP:           httpContext,
		Metadata:       metadata,
		ExecutionState: execState,
		Integration:    testIntegrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Empty(t, execState.Payloads)
	assert.Equal(t, "ami-copy-123", execState.KVs[ec2CopyImageExecutionKVImageID])

	executionMetadata, ok := metadata.Get().(CopyImageExecutionMetadata)
	require.True(t, ok)
	assert.Equal(t, "ami-copy-123", executionMetadata.ImageID)
	assert.Equal(t, "ami-source-123", executionMetadata.SourceImageID)
	assert.Equal(t, "us-east-1", executionMetadata.SourceRegion)
	assert.Equal(t, ImageStatePending, executionMetadata.State)

	require.Len(t, httpContext.Requests, 1)
	requestBody := testRequestBodyString(t, httpContext.Requests[0])
	assert.Contains(t, requestBody, "Action=CopyImage")
	assert.Contains(t, requestBody, "SourceImageId=ami-source-123")
	assert.Contains(t, requestBody, "SourceRegion=us-east-1")
	assert.Contains(t, requestBody, "Name=my-copy")
}

func Test__CopyImage__HandleAction(t *testing.T) {
	component := &CopyImage{}

	t.Run("rule unavailable -> reschedules check", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:     "checkRuleAvailability",
			Logger:   logrus.NewEntry(logrus.New()),
			Requests: requests,
			Metadata: &contexts.MetadataContext{Metadata: CopyImageNodeMetadata{Region: "us-west-2"}},
			Integration: &contexts.IntegrationContext{
				Metadata: common.IntegrationMetadata{
					EventBridge: &common.EventBridgeMetadata{
						Rules: map[string]common.EventBridgeRuleMetadata{},
					},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, copyImageCheckRuleRetryInterval, requests.Duration)
	})

	t.Run("rule available -> subscribes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{Metadata: CopyImageNodeMetadata{Region: "us-west-2"}}
		integration := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.ec2:us-west-2": {
							Source:      Source,
							DetailTypes: []string{DetailTypeAMIStateChange},
						},
					},
				},
			},
		}
		err := component.HandleAction(core.ActionContext{
			Name:        "checkRuleAvailability",
			Logger:      logrus.NewEntry(logrus.New()),
			Requests:    &contexts.RequestContext{},
			Metadata:    metadata,
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.Subscriptions, 1)
		stored, ok := metadata.Get().(CopyImageNodeMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__CopyImage__OnIntegrationMessage(t *testing.T) {
	component := &CopyImage{}

	newExecutionContext := func(imageID string) (*core.ExecutionContext, *contexts.ExecutionStateContext) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		execCtx := &core.ExecutionContext{
			ExecutionState: execState,
			Logger:         log.NewEntry(log.New()),
			Metadata: &contexts.MetadataContext{Metadata: CopyImageExecutionMetadata{
				ImageID:       imageID,
				SourceImageID: "ami-source-123",
				SourceRegion:  "us-east-1",
				State:         ImageStatePending,
			}},
		}
		return execCtx, execState
	}

	t.Run("available event -> emits output", func(t *testing.T) {
		executionCtx, execState := newExecutionContext("ami-copy-123")
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: log.NewEntry(log.New()),
			Integration: &contexts.IntegrationContext{
				Secrets: testIntegrationWithCredentials().Secrets,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`
							<DescribeImagesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
								<requestId>req-123</requestId>
								<imagesSet>
									<item>
										<imageId>ami-copy-123</imageId>
										<name>my-copy</name>
										<imageState>available</imageState>
									</item>
								</imagesSet>
							</DescribeImagesResponse>
						`)),
					},
				},
			},
			Message: common.EventBridgeEvent{
				Region:     "us-west-2",
				Source:     Source,
				DetailType: DetailTypeAMIStateChange,
				Detail: map[string]any{
					"ImageId": "ami-copy-123",
					"State":   ImageStateAvailable,
				},
			},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				if key != ec2CopyImageExecutionKVImageID {
					return nil, nil
				}
				if value != "ami-copy-123" {
					return nil, nil
				}
				return executionCtx, nil
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"]
		output, ok := payload.(map[string]any)
		require.True(t, ok)
		image, ok := output["image"].(*Image)
		require.True(t, ok)
		assert.Equal(t, "ami-copy-123", image.ImageID)
		assert.Equal(t, ImageStateAvailable, image.State)
	})

	t.Run("failed event -> fails execution", func(t *testing.T) {
		executionCtx, execState := newExecutionContext("ami-copy-123")
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: log.NewEntry(log.New()),
			Message: common.EventBridgeEvent{
				Source:     Source,
				DetailType: DetailTypeAMIStateChange,
				Detail: map[string]any{
					"ImageId":      "ami-copy-123",
					"State":        ImageStateFailed,
					"ErrorMessage": "copy failed",
				},
			},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				if key == ec2CopyImageExecutionKVImageID && value == "ami-copy-123" {
					return executionCtx, nil
				}
				return nil, nil
			},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.False(t, execState.Passed)
		assert.Equal(t, models.CanvasNodeExecutionResultReasonError, execState.FailureReason)
		assert.Equal(t, "copy failed", execState.FailureMessage)
	})

	t.Run("pending event -> ignores", func(t *testing.T) {
		executionCtx, execState := newExecutionContext("ami-copy-123")
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: log.NewEntry(log.New()),
			Message: common.EventBridgeEvent{
				Source:     Source,
				DetailType: DetailTypeAMIStateChange,
				Detail: map[string]any{
					"ImageId": "ami-copy-123",
					"State":   ImageStatePending,
				},
			},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return executionCtx, nil
			},
		})

		require.NoError(t, err)
		assert.Empty(t, execState.Payloads)
		assert.False(t, execState.Finished)
	})
}
