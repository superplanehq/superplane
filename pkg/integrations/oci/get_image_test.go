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

func Test__GetImage__Setup(t *testing.T) {
	component := &GetImage{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid", Metadata: &contexts.MetadataContext{}})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing image ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"image": " "}, Metadata: &contexts.MetadataContext{}})
		require.ErrorContains(t, err, "image is required")
	})

	t.Run("valid configuration stores node metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"image": "ocid1.image.oc1..example"}, Metadata: metadata})
		require.NoError(t, err)
		stored := metadata.Get().(imageNodeMetadata)
		assert.Equal(t, "ocid1.image.oc1..example", stored.ImageID)
	})
}

func Test__GetImage__ConfigurationUsesCustomImages(t *testing.T) {
	fields := (&GetImage{}).Configuration()
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	assert.Equal(t, ResourceTypeCustomImage, fields[0].TypeOptions.Resource.Type)
}

func Test__GetImage__Execute(t *testing.T) {
	component := &GetImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id":"ocid1.image.oc1..example",
				"displayName":"image",
				"lifecycleState":"AVAILABLE",
				"compartmentId":"ocid1.compartment.oc1..example",
				"operatingSystem":"Oracle Linux",
				"operatingSystemVersion":"8",
				"launchMode":"PARAVIRTUALIZED",
				"sizeInMBs":51200,
				"timeCreated":"2026-04-28T09:12:42.000Z",
				"createImageAllowed":true
			}`)),
		}},
	}

	metadata := &contexts.MetadataContext{}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"image": "ocid1.image.oc1..example"},
		HTTP:           httpContext,
		Metadata:       metadata,
		ExecutionState: execState,
		Integration:    testOCIIntegration(t),
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/20160918/images/ocid1.image.oc1..example")

	require.Len(t, execState.Payloads, 1)
	payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
	image := payload["image"].(map[string]any)
	assert.Equal(t, "ocid1.image.oc1..example", image["id"])
	assert.Equal(t, "image", image["displayName"])
	assert.Equal(t, "AVAILABLE", image["lifecycleState"])

	stored := metadata.Get().(imageExecutionMetadata)
	assert.Equal(t, "ocid1.image.oc1..example", stored.ImageID)
	assert.Equal(t, "AVAILABLE", stored.State)
	assert.NotEmpty(t, stored.StartedAt)
}

func Test__GetImage__Execute__PlatformImage(t *testing.T) {
	component := &GetImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id":"ocid1.image.oc1..platform",
				"displayName":"Canonical-Ubuntu",
				"lifecycleState":"AVAILABLE",
				"operatingSystem":"Canonical Ubuntu"
			}`)),
		}},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"image": "ocid1.image.oc1..platform"},
		HTTP:           httpContext,
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Integration:    testOCIIntegration(t),
	})
	require.ErrorContains(t, err, "only custom images can be retrieved")
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
}
