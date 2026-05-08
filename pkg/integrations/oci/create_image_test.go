package oci

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateImage__Setup(t *testing.T) {
	component := &CreateImage{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid", Metadata: &contexts.MetadataContext{}})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing compartment -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"displayName": "image", "instance": "ocid1.instance.oc1..example"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "compartment is required")
	})

	t.Run("missing instance -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"compartment": "ocid1.compartment.oc1..example", "displayName": "image"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance is required")
	})

	t.Run("valid configuration stores node metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"compartment": "ocid1.compartment.oc1..example",
				"displayName": "image",
				"instance":    "ocid1.instance.oc1..example",
			},
			Metadata: metadata,
		})
		require.NoError(t, err)
		stored := metadata.Get().(imageNodeMetadata)
		assert.Equal(t, "ocid1.compartment.oc1..example", stored.CompartmentID)
		assert.Equal(t, "image", stored.DisplayName)
	})
}

func Test__CreateImage__ConfigurationUsesConditionalRequiredFields(t *testing.T) {
	fields := (&CreateImage{}).Configuration()
	foundTags := false

	for _, field := range fields {
		switch field.Name {
		case "sourceType":
			require.NotNil(t, field.TypeOptions)
			require.NotNil(t, field.TypeOptions.Resource)
			assert.Equal(t, ResourceTypeImageSource, field.TypeOptions.Resource.Type)
		case "instance", "sourceImageType", "sourceUri", "namespace", "bucket", "object":
			assert.False(t, field.Required, "%s should rely on requiredConditions", field.Name)
			assert.NotEmpty(t, field.RequiredConditions)
			switch field.Name {
			case "instance":
				require.NotNil(t, field.TypeOptions)
				require.NotNil(t, field.TypeOptions.Resource)
				assert.Equal(t, ResourceTypeInstance, field.TypeOptions.Resource.Type)
			case "sourceImageType":
				require.NotNil(t, field.TypeOptions)
				require.NotNil(t, field.TypeOptions.Resource)
				assert.Equal(t, ResourceTypeSourceImageType, field.TypeOptions.Resource.Type)
			case "namespace":
				require.NotNil(t, field.TypeOptions)
				require.NotNil(t, field.TypeOptions.Resource)
				assert.Equal(t, ResourceTypeObjectNamespace, field.TypeOptions.Resource.Type)
			case "bucket":
				require.NotNil(t, field.TypeOptions)
				require.NotNil(t, field.TypeOptions.Resource)
				assert.Equal(t, ResourceTypeObjectBucket, field.TypeOptions.Resource.Type)
			case "object":
				require.NotNil(t, field.TypeOptions)
				require.NotNil(t, field.TypeOptions.Resource)
				assert.Equal(t, ResourceTypeObject, field.TypeOptions.Resource.Type)
			}
		case "tags":
			foundTags = true
			assert.Equal(t, "list", field.Type)
			require.NotNil(t, field.TypeOptions)
			require.NotNil(t, field.TypeOptions.List)
		}
	}
	assert.True(t, foundTags)
}

func Test__CreateImage__ExecuteAndPoll(t *testing.T) {
	component := &CreateImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":"ocid1.image.oc1..created",
					"displayName":"image",
					"lifecycleState":"PROVISIONING",
					"compartmentId":"ocid1.compartment.oc1..example"
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":"ocid1.image.oc1..created",
					"displayName":"image",
					"lifecycleState":"AVAILABLE",
					"compartmentId":"ocid1.compartment.oc1..example",
					"operatingSystem":"Oracle Linux",
					"operatingSystemVersion":"8",
					"launchMode":"PARAVIRTUALIZED",
					"sizeInMBs":51200,
					"timeCreated":"2026-04-28T09:12:42.000Z",
					"createImageAllowed":true,
					"freeformTags":{"env":"test"}
				}`)),
			},
		},
	}

	metadata := &contexts.MetadataContext{}
	requests := &contexts.RequestContext{}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"compartment": "ocid1.compartment.oc1..example",
			"displayName": "image",
			"instance":    "ocid1.instance.oc1..source",
			"tags": []map[string]any{
				{"key": "env", "value": "test"},
			},
		},
		HTTP:           httpContext,
		Metadata:       metadata,
		Requests:       requests,
		ExecutionState: execState,
		Integration:    testOCIIntegration(t),
	})
	require.NoError(t, err)
	assert.Equal(t, "poll", requests.Action)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/20160918/images")

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"instanceId":"ocid1.instance.oc1..source"`)
	assert.Contains(t, string(body), `"freeformTags":{"env":"test"}`)

	err = component.HandleHook(core.ActionHookContext{
		Name:           "poll",
		HTTP:           httpContext,
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: execState,
		Integration:    testOCIIntegration(t),
	})
	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
	image := payload["image"].(map[string]any)
	assert.Equal(t, "ocid1.image.oc1..created", image["id"])
	assert.Equal(t, "AVAILABLE", image["lifecycleState"])
	assert.Equal(t, "Oracle Linux", image["operatingSystem"])
	assert.Equal(t, map[string]string{"env": "test"}, image["freeformTags"])

	stored := metadata.Get().(imageExecutionMetadata)
	assert.Equal(t, "ocid1.image.oc1..created", stored.ImageID)
	assert.Equal(t, "AVAILABLE", stored.State)
	assert.NotEmpty(t, stored.StartedAt)
}

func Test__CreateImage__ObjectStorageURIRequest(t *testing.T) {
	config, err := decodeCreateImageConfiguration(map[string]any{
		"sourceType":      createImageSourceObjectStorageURI,
		"compartment":     "ocid1.compartment.oc1..example",
		"displayName":     "imported",
		"sourceImageType": "QCOW2",
		"sourceUri":       "https://objectstorage.example.com/image.qcow2",
		"tags": []map[string]any{
			{"key": "purpose", "value": "import"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, validateCreateImageConfiguration(config))

	req := createImageRequest(config)
	require.NotNil(t, req.ImageSourceDetails)
	assert.Empty(t, req.InstanceID)
	assert.Equal(t, "objectStorageUri", req.ImageSourceDetails.SourceType)
	assert.Equal(t, "QCOW2", req.ImageSourceDetails.SourceImageType)
	assert.Equal(t, "https://objectstorage.example.com/image.qcow2", req.ImageSourceDetails.SourceURI)
	assert.Equal(t, map[string]string{"purpose": "import"}, req.FreeformTags)
}

func Test__CreateImage__InvalidTag(t *testing.T) {
	config, err := decodeCreateImageConfiguration(map[string]any{
		"compartment": "ocid1.compartment.oc1..example",
		"displayName": "image",
		"instance":    "ocid1.instance.oc1..source",
		"tags": []map[string]any{
			{"key": "env", "value": " "},
		},
	})
	require.NoError(t, err)
	require.ErrorContains(t, validateCreateImageConfiguration(config), "tag value is required")
}

func Test__CreateImage__LegacyInstanceIDConfiguration(t *testing.T) {
	config, err := decodeCreateImageConfiguration(map[string]any{
		"compartment": "ocid1.compartment.oc1..example",
		"displayName": "image",
		"instanceId":  "ocid1.instance.oc1..legacy",
	})
	require.NoError(t, err)
	require.NoError(t, validateCreateImageConfiguration(config))
	assert.Equal(t, "ocid1.instance.oc1..legacy", config.Instance)
	assert.Equal(t, "ocid1.instance.oc1..legacy", config.InstanceID)
}

func testOCIIntegration(t *testing.T) *contexts.IntegrationContext {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})

	return &contexts.IntegrationContext{Configuration: map[string]any{
		"tenancyOcid": "ocid1.tenancy.oc1..example",
		"userOcid":    "ocid1.user.oc1..example",
		"fingerprint": "11:22:33:44",
		"privateKey":  string(keyPEM),
		"region":      "eu-frankfurt-1",
	}}
}
