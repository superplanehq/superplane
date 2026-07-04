package ec2

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ReleaseElasticIP__Setup(t *testing.T) {
	component := &ReleaseElasticIP{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       " ",
				"allocationId": "eipalloc-abc123",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing allocation ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"allocationId": " ",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "allocation ID is required")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"allocationId": "eipalloc-abc123",
			},
			Metadata: metadata,
		})
		require.NoError(t, err)

		stored, ok := metadata.Get().(ReleaseElasticIPNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "eipalloc-abc123", stored.AllocationID)
	})
}

func Test__ReleaseElasticIP__Execute(t *testing.T) {
	component := &ReleaseElasticIP{}

	t.Run("release address -> emits allocation ID and region", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{okResponse(releaseAddressXML())},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"allocationId": "eipalloc-abc123",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    elasticIPIntegration(),
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, ReleaseElasticIPPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "eipalloc-abc123", data["allocationId"])
		assert.Equal(t, "us-east-1", data["region"])

		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Action=ReleaseAddress")
		assert.Contains(t, string(body), "AllocationId=eipalloc-abc123")
	})
}
