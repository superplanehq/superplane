package azure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestAzureIntegration_HandleAction(t *testing.T) {
	integration := &AzureIntegration{}

	err := integration.HandleAction(core.IntegrationActionContext{Name: "unknown-action"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestAzureIntegration_ListResources(t *testing.T) {
	integration := &AzureIntegration{}

	logger := logrus.NewEntry(logrus.New())
	ctx := core.ListResourcesContext{Logger: logger}

	tests := []struct {
		resourceType string
		expectError  bool
	}{
		{"resourceGroup", false},
		{"virtualNetwork", false},
		{"subnet", false},
		{"unsupported", true},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			resources, err := integration.ListResources(tt.resourceType, ctx)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported resource type")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resources)
			}
		})
	}
}

func TestAzureIntegration_HandleRequest_Unknown(t *testing.T) {
	integration := &AzureIntegration{}

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	integration.HandleRequest(core.HTTPRequestContext{
		Request:  req,
		Response: rec,
		Logger:   logrus.NewEntry(logrus.New()),
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestNewProvider_FailsWithoutAccessToken(t *testing.T) {
	ctx := &contexts.IntegrationContext{
		IntegrationID: "00000000-0000-0000-0000-000000000002",
		Configuration: map[string]any{
			"tenantId":       "test-tenant",
			"clientId":       "test-client",
			"subscriptionId": "test-sub",
		},
		// No secrets set — simulates integration that has not yet synced.
	}
	result, err := newProvider(ctx)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Azure access token not found")
}

const testIntegrationID = "00000000-0000-0000-0000-000000000001"

func newTestProvider(t *testing.T, handler http.HandlerFunc) (*AzureProvider, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)

	client := &armClient{
		subscriptionID: "test-sub",
		baseURL:        server.URL,
		tokenFunc:      func(_ context.Context) (string, error) { return "mock-token", nil },
		httpClient:     &http.Client{},
		logger:         logrus.NewEntry(logrus.New()),
	}

	return &AzureProvider{
		subscriptionID: "test-sub",
		client:         client,
		logger:         logrus.NewEntry(logrus.New()),
	}, server
}

func newTestListCtx() core.ListResourcesContext {
	return core.ListResourcesContext{
		Logger: logrus.NewEntry(logrus.New()),
		Integration: &contexts.IntegrationContext{
			IntegrationID: testIntegrationID,
		},
	}
}

func TestListResourceGroupLocations_EmptyResourceGroup(t *testing.T) {
	integration := &AzureIntegration{}
	resources, err := integration.ListResourceGroupLocations(newTestListCtx(), "")
	assert.NoError(t, err)
	assert.Empty(t, resources)
}

func TestListResourceGroupLocations_Success(t *testing.T) {
	provider, server := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":     "my-rg",
			"id":       "/subscriptions/test-sub/resourceGroups/my-rg",
			"location": "eastus",
		})
	})
	defer server.Close()

	resources, err := listResourceGroupLocations(newTestListCtx(), provider, "my-rg")
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, ResourceTypeResourceGroupLocation, resources[0].Type)
	assert.Equal(t, "eastus", resources[0].Name)
	assert.Equal(t, "eastus", resources[0].ID)
}

func TestListResourceGroupLocations_NotFound(t *testing.T) {
	provider, server := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "ResourceGroupNotFound",
				"message": "Resource group 'my-rg' could not be found.",
			},
		})
	})
	defer server.Close()

	resources, err := listResourceGroupLocations(newTestListCtx(), provider, "my-rg")
	assert.NoError(t, err)
	assert.Empty(t, resources)
}

func TestCreateVMComponent_LocationField(t *testing.T) {
	component := &CreateVMComponent{}
	fields := component.Configuration()

	for i := range fields {
		assert.NotEqual(t, "location", fields[i].Name, "location field must not be in configuration")
	}

	var sizeField *configuration.Field
	for i := range fields {
		if fields[i].Name == "size" {
			sizeField = &fields[i]
			break
		}
	}

	require.NotNil(t, sizeField, "size field must exist")
	require.NotNil(t, sizeField.TypeOptions)
	require.NotNil(t, sizeField.TypeOptions.Resource)
	assert.Equal(t, ResourceTypeVMSizeDropdown, sizeField.TypeOptions.Resource.Type)
	require.Len(t, sizeField.TypeOptions.Resource.Parameters, 1)
	assert.Equal(t, "resourceGroup", sizeField.TypeOptions.Resource.Parameters[0].Name)
	assert.Equal(t, "resourceGroup", sizeField.TypeOptions.Resource.Parameters[0].ValueFrom.Field)
}
