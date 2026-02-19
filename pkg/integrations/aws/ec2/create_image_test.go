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

func Test__CreateImage__Setup(t *testing.T) {
	component := &CreateImage{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid"})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata:    &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{},
			Configuration: map[string]any{
				"region":     " ",
				"instanceId": "i-123",
				"name":       "my-image",
			},
		})
		require.ErrorContains(t, err, "region is required")
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
				"region":     "us-east-1",
				"instanceId": "i-123",
				"name":       "my-image",
			},
			Metadata:    metadata,
			Requests:    requests,
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integration.ActionRequests[0].ActionName)
		params := integration.ActionRequests[0].Parameters.(common.ProvisionRuleParameters)
		assert.Equal(t, "us-east-1", params.Region)
		assert.Equal(t, Source, params.Source)
		assert.Equal(t, DetailTypeAMIStateChange, params.DetailType)
		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, createImageInitialRuleAvailabilityWait, requests.Duration)
	})

	t.Run("rule exists -> subscribes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integration := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.ec2:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypeAMIStateChange},
						},
					},
				},
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"instanceId": "i-123",
				"name":       "my-image",
			},
			Metadata:    metadata,
			Requests:    &contexts.RequestContext{},
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.Subscriptions, 1)
		stored, ok := metadata.Get().(CreateImageNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__CreateImage__Execute(t *testing.T) {
	component := &CreateImage{}

	t.Run("create image -> persists waiting state", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<CreateImageResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<requestId>req-123</requestId>
							<imageId>ami-abc</imageId>
						</CreateImageResponse>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"instanceId":  "i-123",
				"name":        "my-image",
				"description": "test image",
				"noReboot":    true,
			},
			HTTP:           httpContext,
			Metadata:       metadata,
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
		require.Empty(t, execState.Payloads)
		assert.Equal(t, "ami-abc", execState.KVs[ec2ImageExecutionKVImageID])

		executionMetadata, ok := metadata.Get().(CreateImageExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, "ami-abc", executionMetadata.ImageID)
		assert.Equal(t, ImageStatePending, executionMetadata.State)
	})
}

func Test__CreateImage__HandleAction(t *testing.T) {
	component := &CreateImage{}

	t.Run("rule unavailable -> reschedules check", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:     "checkRuleAvailability",
			Logger:   logrus.NewEntry(logrus.New()),
			Requests: requests,
			Metadata: &contexts.MetadataContext{Metadata: CreateImageNodeMetadata{Region: "us-east-1"}},
			Integration: &contexts.IntegrationContext{
				Metadata: common.IntegrationMetadata{EventBridge: &common.EventBridgeMetadata{Rules: map[string]common.EventBridgeRuleMetadata{}}},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, createImageCheckRuleRetryInterval, requests.Duration)
	})

	t.Run("rule available -> subscribes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{Metadata: CreateImageNodeMetadata{Region: "us-east-1"}}
		integration := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.ec2:us-east-1": {
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
		stored, ok := metadata.Get().(CreateImageNodeMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__CreateImage__OnIntegrationMessage(t *testing.T) {
	component := &CreateImage{}

	newExecutionContext := func(imageID string) (*core.ExecutionContext, *contexts.ExecutionStateContext) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		execCtx := &core.ExecutionContext{
			ExecutionState: execState,
			Logger:         log.NewEntry(log.New()),
			Metadata: &contexts.MetadataContext{Metadata: CreateImageExecutionMetadata{
				ImageID: imageID,
				State:   ImageStatePending,
			}},
		}
		return execCtx, execState
	}

	t.Run("available event -> emits output", func(t *testing.T) {
		executionCtx, execState := newExecutionContext("ami-abc")
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: log.NewEntry(log.New()),
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
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
										<imageId>ami-abc</imageId>
										<imageState>available</imageState>
									</item>
								</imagesSet>
							</DescribeImagesResponse>
						`)),
					},
				},
			},
			Message: common.EventBridgeEvent{
				Region:     "us-east-1",
				Source:     Source,
				DetailType: DetailTypeAMIStateChange,
				Detail: map[string]any{
					"ImageId": "ami-abc",
					"State":   "available",
				},
			},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				if key != ec2ImageExecutionKVImageID {
					return nil, nil
				}
				if value != "ami-abc" {
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
		assert.Equal(t, "ami-abc", image.ImageID)
		assert.Equal(t, ImageStateAvailable, image.State)
	})

	t.Run("failed event -> fails execution", func(t *testing.T) {
		executionCtx, execState := newExecutionContext("ami-abc")
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: log.NewEntry(log.New()),
			Message: common.EventBridgeEvent{
				Source:     Source,
				DetailType: DetailTypeAMIStateChange,
				Detail: map[string]any{
					"ImageId":      "ami-abc",
					"State":        "failed",
					"ErrorMessage": "Insufficient permissions",
				},
			},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				if key == ec2ImageExecutionKVImageID && value == "ami-abc" {
					return executionCtx, nil
				}
				return nil, nil
			},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.False(t, execState.Passed)
		assert.Equal(t, models.CanvasNodeExecutionResultReasonError, execState.FailureReason)
		assert.Equal(t, "Insufficient permissions", execState.FailureMessage)
	})

	t.Run("pending event -> ignores", func(t *testing.T) {
		executionCtx, execState := newExecutionContext("ami-abc")
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: log.NewEntry(log.New()),
			Message: common.EventBridgeEvent{
				Source:     Source,
				DetailType: DetailTypeAMIStateChange,
				Detail: map[string]any{
					"ImageId": "ami-abc",
					"State":   "pending",
				},
			},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				if key == ec2ImageExecutionKVImageID && value == "ami-abc" {
					return executionCtx, nil
				}
				return nil, nil
			},
		})

		require.NoError(t, err)
		assert.Empty(t, execState.Payloads)
		assert.False(t, execState.Finished)
	})
}
