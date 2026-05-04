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

func Test__UpdateImage__Setup(t *testing.T) {
	component := &UpdateImage{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid", Metadata: &contexts.MetadataContext{}})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing display name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"image": "ocid1.image.oc1..example", "displayName": " "},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "displayName is required")
	})

	t.Run("valid configuration stores node metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"image": "ocid1.image.oc1..example", "displayName": "renamed"},
			Metadata:      metadata,
		})
		require.NoError(t, err)
		stored := metadata.Get().(imageNodeMetadata)
		assert.Equal(t, "ocid1.image.oc1..example", stored.ImageID)
		assert.Equal(t, "renamed", stored.DisplayName)
	})
}

func Test__UpdateImage__ConfigurationUsesCustomImages(t *testing.T) {
	fields := (&UpdateImage{}).Configuration()
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	assert.Equal(t, ResourceTypeCustomImage, fields[0].TypeOptions.Resource.Type)
}

func Test__UpdateImage__Execute(t *testing.T) {
	component := &UpdateImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":"ocid1.image.oc1..example",
					"displayName":"original",
					"lifecycleState":"AVAILABLE",
					"compartmentId":"ocid1.compartment.oc1..example"
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":"ocid1.image.oc1..example",
					"displayName":"renamed",
					"lifecycleState":"AVAILABLE",
					"compartmentId":"ocid1.compartment.oc1..example",
					"operatingSystem":"Oracle Linux",
					"timeCreated":"2026-04-28T09:12:42.000Z",
					"createImageAllowed":true
				}`)),
			},
		},
	}

	metadata := &contexts.MetadataContext{}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"image": "ocid1.image.oc1..example", "displayName": "renamed"},
		HTTP:           httpContext,
		Metadata:       metadata,
		ExecutionState: execState,
		Integration:    testOCIIntegration(t),
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	assert.Equal(t, http.MethodPut, httpContext.Requests[1].Method)
	body, err := io.ReadAll(httpContext.Requests[1].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"displayName":"renamed"`)

	payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
	image := payload["image"].(map[string]any)
	assert.Equal(t, "renamed", image["displayName"])

	stored := metadata.Get().(imageExecutionMetadata)
	assert.Equal(t, "renamed", stored.DisplayName)
	assert.NotEmpty(t, stored.StartedAt)
}

func Test__UpdateImage__Execute__PlatformImage(t *testing.T) {
	component := &UpdateImage{}
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
		Configuration:  map[string]any{"image": "ocid1.image.oc1..platform", "displayName": "renamed"},
		HTTP:           httpContext,
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Integration:    testOCIIntegration(t),
	})
	require.ErrorContains(t, err, "only custom images can be updated or deleted")
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
}
