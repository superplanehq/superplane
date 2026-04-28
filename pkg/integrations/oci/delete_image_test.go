package oci

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

func Test__DeleteImage__Setup(t *testing.T) {
	component := &DeleteImage{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid", Metadata: &contexts.MetadataContext{}})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing image ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"imageId": " "}, Metadata: &contexts.MetadataContext{}})
		require.ErrorContains(t, err, "imageId is required")
	})

	t.Run("valid configuration stores node metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"imageId": "ocid1.image.oc1..example"}, Metadata: metadata})
		require.NoError(t, err)
		stored := metadata.Get().(imageNodeMetadata)
		assert.Equal(t, "ocid1.image.oc1..example", stored.ImageID)
	})
}

func Test__DeleteImage__Execute(t *testing.T) {
	component := &DeleteImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}},
	}

	metadata := &contexts.MetadataContext{}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"imageId": "ocid1.image.oc1..example"},
		HTTP:           httpContext,
		Metadata:       metadata,
		ExecutionState: execState,
		Integration:    testOCIIntegration(t),
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/20160918/images/ocid1.image.oc1..example")

	payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "ocid1.image.oc1..example", payload["imageId"])
	assert.Equal(t, imageStateDeleted, payload["state"])
	assert.NotEmpty(t, payload["deletedAt"])

	stored := metadata.Get().(imageExecutionMetadata)
	assert.Equal(t, "ocid1.image.oc1..example", stored.ImageID)
	assert.Equal(t, imageStateDeleted, stored.State)
	assert.NotEmpty(t, stored.StartedAt)
}
