package railway

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Railway__TriggerDeploy__Setup(t *testing.T) {
	action := &TriggerDeploy{}

	t.Run("success", func(t *testing.T) {
		err := action.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project":     "p-1",
				"service":     "s-1",
				"environment": "e-1",
			},
		})
		require.NoError(t, err)
	})

	t.Run("missing project", func(t *testing.T) {
		err := action.Setup(core.SetupContext{
			Configuration: map[string]any{
				"service":     "s-1",
				"environment": "e-1",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})
}

func Test__Railway__TriggerDeploy__Execute(t *testing.T) {
	action := &TriggerDeploy{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"serviceInstanceDeployV2":"deploy-123"}}`)),
			},
		},
	}

	intCtx := &contexts.IntegrationContext{NewSetupFlow: true}
	_ = intCtx.SetSecret("apiToken", []byte("test-token"))

	execCtx := core.ExecutionContext{
		HTTP: httpCtx,
		Configuration: map[string]any{
			"project":     "p-1",
			"service":     "s-1",
			"environment": "e-1",
		},
		Integration:    intCtx,
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
		Requests:       &contexts.RequestContext{},
	}

	err := action.Execute(execCtx)
	require.NoError(t, err)

	// Check metadata is set to QUEUED
	metadata := TriggerDeployExecutionMetadata{}
	err = mapstructure.Decode(execCtx.Metadata.Get(), &metadata)
	require.NoError(t, err)
	require.NotNil(t, metadata.Deploy)
	assert.Equal(t, "deploy-123", metadata.Deploy.ID)
	assert.Equal(t, "QUEUED", metadata.Deploy.Status)

	// Check poll hook is scheduled
	reqs := execCtx.Requests.(*contexts.RequestContext)
	assert.Equal(t, "poll", reqs.Action)
	assert.Equal(t, 15*time.Second, reqs.Duration)
}

func Test__Railway__TriggerDeploy__Poll(t *testing.T) {
	action := &TriggerDeploy{}

	t.Run("continues polling when active", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"deployment":{"id":"deploy-123","status":"BUILDING","createdAt":"","updatedAt":""}}}`)),
				},
			},
		}

		intCtx := &contexts.IntegrationContext{NewSetupFlow: true}
		_ = intCtx.SetSecret("apiToken", []byte("test-token"))

		metaCtx := &contexts.MetadataContext{
			Metadata: TriggerDeployExecutionMetadata{
				Deploy: &TriggerDeployMetadata{
					ID:          "deploy-123",
					Status:      "QUEUED",
					ProjectID:   "p-1",
					ServiceID:   "s-1",
					Environment: "e-1",
				},
			},
		}

		reqsCtx := &contexts.RequestContext{}

		err := action.poll(core.ActionHookContext{
			HTTP:           httpCtx,
			Integration:    intCtx,
			Metadata:       metaCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       reqsCtx,
		})
		require.NoError(t, err)

		// Check metadata updated to BUILDING
		metadata := TriggerDeployExecutionMetadata{}
		err = mapstructure.Decode(metaCtx.Get(), &metadata)
		require.NoError(t, err)
		assert.Equal(t, "BUILDING", metadata.Deploy.Status)

		// Check next poll is scheduled
		assert.Equal(t, "poll", reqsCtx.Action)
		assert.Equal(t, 15*time.Second, reqsCtx.Duration)
	})

	t.Run("emits success on SUCCESS status", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"deployment":{"id":"deploy-123","status":"SUCCESS","createdAt":"","updatedAt":""}}}`)),
				},
			},
		}

		intCtx := &contexts.IntegrationContext{NewSetupFlow: true}
		_ = intCtx.SetSecret("apiToken", []byte("test-token"))

		metaCtx := &contexts.MetadataContext{
			Metadata: TriggerDeployExecutionMetadata{
				Deploy: &TriggerDeployMetadata{
					ID:          "deploy-123",
					Status:      "BUILDING",
					ProjectID:   "p-1",
					ServiceID:   "s-1",
					Environment: "e-1",
				},
			},
		}

		stateCtx := &contexts.ExecutionStateContext{}
		reqsCtx := &contexts.RequestContext{}

		err := action.poll(core.ActionHookContext{
			HTTP:           httpCtx,
			Integration:    intCtx,
			Metadata:       metaCtx,
			ExecutionState: stateCtx,
			Requests:       reqsCtx,
		})
		require.NoError(t, err)

		// Check emitted event
		assert.Equal(t, "success", stateCtx.Channel)
		assert.Equal(t, "railway.deploy.finished", stateCtx.Type)
		require.Len(t, stateCtx.Payloads, 1)

		// No further polls scheduled
		assert.Equal(t, "", reqsCtx.Action)
	})

	t.Run("emits failed on FAILED status", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"deployment":{"id":"deploy-123","status":"FAILED","createdAt":"","updatedAt":""}}}`)),
				},
			},
		}

		intCtx := &contexts.IntegrationContext{NewSetupFlow: true}
		_ = intCtx.SetSecret("apiToken", []byte("test-token"))

		metaCtx := &contexts.MetadataContext{
			Metadata: TriggerDeployExecutionMetadata{
				Deploy: &TriggerDeployMetadata{
					ID:          "deploy-123",
					Status:      "BUILDING",
					ProjectID:   "p-1",
					ServiceID:   "s-1",
					Environment: "e-1",
				},
			},
		}

		stateCtx := &contexts.ExecutionStateContext{}
		reqsCtx := &contexts.RequestContext{}

		err := action.poll(core.ActionHookContext{
			HTTP:           httpCtx,
			Integration:    intCtx,
			Metadata:       metaCtx,
			ExecutionState: stateCtx,
			Requests:       reqsCtx,
		})
		require.NoError(t, err)

		// Check emitted event
		assert.Equal(t, "failed", stateCtx.Channel)
		assert.Equal(t, "railway.deploy.finished", stateCtx.Type)
		require.Len(t, stateCtx.Payloads, 1)

		// No further polls scheduled
		assert.Equal(t, "", reqsCtx.Action)
	})
}
