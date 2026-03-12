package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// mockIntegrationContext implements core.IntegrationContext for testing.
type mockIntegrationContext struct {
	id     string
	config map[string]string
}

func (m *mockIntegrationContext) ID() uuid.UUID {
	id, _ := uuid.Parse(m.id)
	return id
}

func (m *mockIntegrationContext) GetConfig(name string) ([]byte, error) {
	if v, ok := m.config[name]; ok {
		return []byte(v), nil
	}
	return nil, fmt.Errorf("config %s not found", name)
}

func (m *mockIntegrationContext) GetMetadata() any                    { return nil }
func (m *mockIntegrationContext) SetMetadata(any)                     {}
func (m *mockIntegrationContext) Ready()                              {}
func (m *mockIntegrationContext) Error(string)                        {}
func (m *mockIntegrationContext) NewBrowserAction(core.BrowserAction) {}
func (m *mockIntegrationContext) RemoveBrowserAction()                {}
func (m *mockIntegrationContext) SetSecret(string, []byte) error      { return nil }
func (m *mockIntegrationContext) GetSecrets() ([]core.IntegrationSecret, error) {
	return nil, nil
}
func (m *mockIntegrationContext) RequestWebhook(any) error           { return nil }
func (m *mockIntegrationContext) Subscribe(any) (*uuid.UUID, error)  { return nil, nil }
func (m *mockIntegrationContext) ScheduleResync(time.Duration) error { return nil }
func (m *mockIntegrationContext) ScheduleActionCall(string, any, time.Duration) error {
	return nil
}
func (m *mockIntegrationContext) ListSubscriptions() ([]core.IntegrationSubscriptionContext, error) {
	return nil, nil
}
func (m *mockIntegrationContext) FindSubscription(func(core.IntegrationSubscriptionContext) bool) (core.IntegrationSubscriptionContext, error) {
	return nil, nil
}

func TestAzureIntegration_Name(t *testing.T) {
	integration := &AzureIntegration{}
	assert.Equal(t, "azure", integration.Name())
}

func TestAzureIntegration_Label(t *testing.T) {
	integration := &AzureIntegration{}
	assert.Equal(t, "Microsoft Azure", integration.Label())
}

func TestAzureIntegration_Icon(t *testing.T) {
	integration := &AzureIntegration{}
	assert.Equal(t, "azure", integration.Icon())
}

func TestAzureIntegration_Description(t *testing.T) {
	integration := &AzureIntegration{}
	description := integration.Description()
	assert.NotEmpty(t, description)
	assert.Contains(t, description, "Azure")
}

func TestAzureIntegration_Instructions(t *testing.T) {
	integration := &AzureIntegration{}
	instructions := integration.Instructions()
	assert.NotEmpty(t, instructions)
	assert.Contains(t, instructions, "Workload Identity Federation")
	assert.Contains(t, instructions, "App Registration")
	assert.Contains(t, instructions, "Tenant ID")
	assert.Contains(t, instructions, "Client ID")
	assert.Contains(t, instructions, "Subscription ID")
}

func TestAzureIntegration_Configuration(t *testing.T) {
	integration := &AzureIntegration{}
	fields := integration.Configuration()

	require.Len(t, fields, 3, "Should have exactly 3 configuration fields")

	tenantField := fields[0]
	assert.Equal(t, "tenantId", tenantField.Name)
	assert.Equal(t, "Tenant ID", tenantField.Label)
	assert.Equal(t, configuration.FieldTypeString, tenantField.Type)
	assert.True(t, tenantField.Required)
	assert.NotEmpty(t, tenantField.Description)

	clientField := fields[1]
	assert.Equal(t, "clientId", clientField.Name)
	assert.Equal(t, "Client ID", clientField.Label)
	assert.Equal(t, configuration.FieldTypeString, clientField.Type)
	assert.True(t, clientField.Required)
	assert.NotEmpty(t, clientField.Description)

	subscriptionField := fields[2]
	assert.Equal(t, "subscriptionId", subscriptionField.Name)
	assert.Equal(t, "Subscription ID", subscriptionField.Label)
	assert.Equal(t, configuration.FieldTypeString, subscriptionField.Type)
	assert.True(t, subscriptionField.Required)
	assert.NotEmpty(t, subscriptionField.Description)
}

func TestAzureIntegration_Components(t *testing.T) {
	integration := &AzureIntegration{}
	components := integration.Components()

	assert.NotNil(t, components)
	assert.IsType(t, []core.Component{}, components)
}

func TestAzureIntegration_Triggers(t *testing.T) {
	integration := &AzureIntegration{}
	triggers := integration.Triggers()

	assert.NotNil(t, triggers)
	assert.IsType(t, []core.Trigger{}, triggers)
}

func TestAzureIntegration_Actions(t *testing.T) {
	integration := &AzureIntegration{}
	actions := integration.Actions()

	assert.NotNil(t, actions)
	assert.IsType(t, []core.Action{}, actions)
}

func TestAzureIntegration_HandleAction(t *testing.T) {
	integration := &AzureIntegration{}

	ctx := core.IntegrationActionContext{
		Name: "unknown-action",
	}

	err := integration.HandleAction(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestAzureIntegration_ListResources(t *testing.T) {
	integration := &AzureIntegration{}

	logger := logrus.NewEntry(logrus.New())
	ctx := core.ListResourcesContext{
		Logger: logger,
	}

	tests := []struct {
		name         string
		resourceType string
		expectError  bool
	}{
		{
			name:         "resource group",
			resourceType: "resourceGroup",
			expectError:  false,
		},
		{
			name:         "virtual network",
			resourceType: "virtualNetwork",
			expectError:  false,
		},
		{
			name:         "subnet",
			resourceType: "subnet",
			expectError:  false,
		},
		{
			name:         "unsupported type",
			resourceType: "unsupported",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestAzureIntegration_Cleanup(t *testing.T) {
	integration := &AzureIntegration{}

	logger := logrus.NewEntry(logrus.New())
	ctx := core.IntegrationCleanupContext{
		Logger: logger,
	}

	err := integration.Cleanup(ctx)
	assert.NoError(t, err)
}

func TestAzureIntegration_HandleRequest_Unknown(t *testing.T) {
	integration := &AzureIntegration{}

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	logger := logrus.NewEntry(logrus.New())
	ctx := core.HTTPRequestContext{
		Request:  req,
		Response: rec,
		Logger:   logger,
	}

	integration.HandleRequest(ctx)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAzureIntegration_EnsureProvider_ReturnsCachedProvider(t *testing.T) {
	testID := "00000000-0000-0000-0000-000000000001"
	provider := &AzureProvider{}
	integration := &AzureIntegration{
		provider:      provider,
		integrationID: testID,
	}

	ctx := &mockIntegrationContext{id: testID}
	result, err := integration.ensureProvider(ctx)
	assert.NoError(t, err)
	assert.Equal(t, provider, result)
}

func TestAzureIntegration_EnsureProvider_FailsWithoutOIDC(t *testing.T) {
	integration := &AzureIntegration{}

	ctx := &mockIntegrationContext{id: "00000000-0000-0000-0000-000000000002"}
	result, err := integration.ensureProvider(ctx)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "OIDC provider not available")
}

func TestConfiguration_Struct(t *testing.T) {
	config := Configuration{
		TenantID:       "test-tenant-id",
		ClientID:       "test-client-id",
		SubscriptionID: "test-subscription-id",
	}

	assert.Equal(t, "test-tenant-id", config.TenantID)
	assert.Equal(t, "test-client-id", config.ClientID)
	assert.Equal(t, "test-subscription-id", config.SubscriptionID)
}

const testIntegrationID = "00000000-0000-0000-0000-000000000001"

func newTestIntegration(t *testing.T, handler http.HandlerFunc) (*AzureIntegration, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)

	client := &armClient{
		subscriptionID: "test-sub",
		baseURL:        server.URL,
		tokenFunc:      func(_ context.Context) (string, error) { return "mock-token", nil },
		httpClient:     &http.Client{},
		logger:         logrus.NewEntry(logrus.New()),
	}

	provider := &AzureProvider{
		subscriptionID: "test-sub",
		client:         client,
		logger:         logrus.NewEntry(logrus.New()),
	}

	integration := &AzureIntegration{
		provider:      provider,
		integrationID: testIntegrationID,
	}

	return integration, server
}

func newTestListCtx() core.ListResourcesContext {
	return core.ListResourcesContext{
		Logger: logrus.NewEntry(logrus.New()),
		Integration: &mockIntegrationContext{
			id: testIntegrationID,
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
	integration, server := newTestIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":     "my-rg",
			"id":       "/subscriptions/test-sub/resourceGroups/my-rg",
			"location": "eastus",
		})
	})
	defer server.Close()

	resources, err := integration.ListResourceGroupLocations(newTestListCtx(), "my-rg")
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, ResourceTypeResourceGroupLocation, resources[0].Type)
	assert.Equal(t, "eastus", resources[0].Name)
	assert.Equal(t, "eastus", resources[0].ID)
}

func TestListResourceGroupLocations_NotFound(t *testing.T) {
	integration, server := newTestIntegration(t, func(w http.ResponseWriter, r *http.Request) {
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

	resources, err := integration.ListResourceGroupLocations(newTestListCtx(), "my-rg")
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
