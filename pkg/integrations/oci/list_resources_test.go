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

func Test__OCI__ListResources__StaticDropdownResources(t *testing.T) {
	integration := &OCI{}

	tests := []struct {
		name         string
		resourceType string
		ids          []string
	}{
		{
			name:         "boot volume performance",
			resourceType: ResourceTypeBootVolumeVPU,
			ids:          []string{"0", "10", "20", "30"},
		},
		{
			name:         "image source",
			resourceType: ResourceTypeImageSource,
			ids:          []string{createImageSourceInstance, createImageSourceObjectStorageURI, createImageSourceObjectStorageObject},
		},
		{
			name:         "source image type",
			resourceType: ResourceTypeSourceImageType,
			ids:          []string{"QCOW2", "VMDK", "OCI"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resources, err := integration.ListResources(tc.resourceType, core.ListResourcesContext{})
			require.NoError(t, err)
			require.Len(t, resources, len(tc.ids))

			for i, id := range tc.ids {
				assert.Equal(t, tc.resourceType, resources[i].Type)
				assert.Equal(t, id, resources[i].ID)
				assert.NotEmpty(t, resources[i].Name)
			}
		})
	}
}

func Test__OCI__ListResources__ImageOperatingSystems(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`[
				{"id":"ocid1.image.oc1..one","displayName":"ubuntu","lifecycleState":"AVAILABLE","operatingSystem":"Canonical Ubuntu"},
				{"id":"ocid1.image.oc1..two","displayName":"oracle","lifecycleState":"AVAILABLE","operatingSystem":"Oracle Linux"},
				{"id":"ocid1.image.oc1..three","displayName":"old","lifecycleState":"DELETED","operatingSystem":"Windows"},
				{"id":"ocid1.image.oc1..four","displayName":"ubuntu-2","lifecycleState":"AVAILABLE","operatingSystem":"Canonical Ubuntu"}
			]`)),
		}},
	}

	resources, err := (&OCI{}).ListResources(ResourceTypeImageOS, core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: testOCIIntegration(t),
	})
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "Canonical Ubuntu", resources[0].ID)
	assert.Equal(t, "Oracle Linux", resources[1].ID)
	assert.Equal(t, ResourceTypeImageOS, resources[0].Type)
}

func Test__OCI__ListResources__CustomImagesPaginates(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Header:     http.Header{"opc-next-page": []string{"page-2"}},
				Body: io.NopCloser(strings.NewReader(`[
					{"id":"ocid1.image.oc1..platform","displayName":"Canonical-Ubuntu","lifecycleState":"AVAILABLE"}
				]`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"id":"ocid1.image.oc1..custom","displayName":"golden-image","lifecycleState":"AVAILABLE","compartmentId":"ocid1.tenancy.oc1..example"}
				]`)),
			},
		},
	}

	resources, err := (&OCI{}).ListResources(ResourceTypeCustomImage, core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: testOCIIntegration(t),
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, ResourceTypeCustomImage, resources[0].Type)
	assert.Equal(t, "golden-image", resources[0].Name)
	assert.Equal(t, "ocid1.image.oc1..custom", resources[0].ID)
	require.Len(t, httpContext.Requests, 2)
	assert.Empty(t, httpContext.Requests[0].URL.Query().Get("page"))
	assert.Equal(t, "page-2", httpContext.Requests[1].URL.Query().Get("page"))
}

func Test__OCI__ListResources__ObjectStorageNamespace(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`"namespace-a"`)),
		}},
	}

	resources, err := (&OCI{}).ListResources(ResourceTypeObjectNamespace, core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: testOCIIntegration(t),
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, ResourceTypeObjectNamespace, resources[0].Type)
	assert.Equal(t, "namespace-a", resources[0].ID)
	assert.Equal(t, "namespace-a", resources[0].Name)
}

func Test__OCI__ListResources__ObjectStorageBuckets(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`[
				{"name":"bucket-a","compartmentId":"ocid1.compartment.oc1..example","namespace":"namespace-a"},
				{"name":"bucket-b","compartmentId":"ocid1.compartment.oc1..example","namespace":"namespace-a"}
			]`)),
		}},
	}

	resources, err := (&OCI{}).ListResources(ResourceTypeObjectBucket, core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: testOCIIntegration(t),
		Parameters: map[string]string{
			"namespaceName": "namespace-a",
			"compartmentId": "ocid1.compartment.oc1..example",
		},
	})
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, ResourceTypeObjectBucket, resources[0].Type)
	assert.Equal(t, "bucket-a", resources[0].ID)
	assert.Equal(t, "bucket-b", resources[1].ID)
	require.Len(t, httpContext.Requests, 1)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/n/namespace-a/b")
	assert.Equal(t, "ocid1.compartment.oc1..example", httpContext.Requests[0].URL.Query().Get("compartmentId"))
}

func Test__OCI__ListResources__ObjectStorageObjects(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"objects": [
					{"name":"image-a.qcow2"},
					{"name":"folder/image-b.qcow2"}
				]
			}`)),
		}},
	}

	resources, err := (&OCI{}).ListResources(ResourceTypeObject, core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: testOCIIntegration(t),
		Parameters: map[string]string{
			"namespaceName": "namespace-a",
			"bucketName":    "bucket-a",
		},
	})
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, ResourceTypeObject, resources[0].Type)
	assert.Equal(t, "image-a.qcow2", resources[0].ID)
	assert.Equal(t, "folder/image-b.qcow2", resources[1].ID)
	require.Len(t, httpContext.Requests, 1)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/n/namespace-a/b/bucket-a/o")
}

func Test__OCI__ListResources__Instances(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`[
				{"id":"ocid1.instance.oc1..running","displayName":"running-instance","lifecycleState":"RUNNING"},
				{"id":"ocid1.instance.oc1..stopped","displayName":"stopped-instance","lifecycleState":"STOPPED"},
				{"id":"ocid1.instance.oc1..terminated","displayName":"terminated-instance","lifecycleState":"TERMINATED"}
			]`)),
		}},
	}

	resources, err := (&OCI{}).ListResources(ResourceTypeInstance, core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: testOCIIntegration(t),
		Parameters: map[string]string{
			"compartmentId": "ocid1.compartment.oc1..example",
		},
	})
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, ResourceTypeInstance, resources[0].Type)
	assert.Equal(t, "running-instance", resources[0].Name)
	assert.Equal(t, "ocid1.instance.oc1..running", resources[0].ID)
	assert.Equal(t, "stopped-instance", resources[1].Name)
	require.Len(t, httpContext.Requests, 1)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/20160918/instances")
	assert.Equal(t, "ocid1.compartment.oc1..example", httpContext.Requests[0].URL.Query().Get("compartmentId"))
}
