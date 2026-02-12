package azure

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

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

	// Check tenantId field
	tenantField := fields[0]
	assert.Equal(t, "tenantId", tenantField.Name)
	assert.Equal(t, "Tenant ID", tenantField.Label)
	assert.Equal(t, configuration.FieldTypeString, tenantField.Type)
	assert.True(t, tenantField.Required)
	assert.NotEmpty(t, tenantField.Description)

	// Check clientId field
	clientField := fields[1]
	assert.Equal(t, "clientId", clientField.Name)
	assert.Equal(t, "Client ID", clientField.Label)
	assert.Equal(t, configuration.FieldTypeString, clientField.Type)
	assert.True(t, clientField.Required)
	assert.NotEmpty(t, clientField.Description)

	// Check subscriptionId field
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

	// Currently returns empty list
	assert.NotNil(t, actions)
	assert.IsType(t, []core.Action{}, actions)
}

func TestAzureIntegration_HandleAction(t *testing.T) {
	integration := &AzureIntegration{}

	// Create mock context
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

func TestAzureIntegration_GetProvider(t *testing.T) {
	integration := &AzureIntegration{}

	// Initially nil
	assert.Nil(t, integration.GetProvider())

	// Set a mock provider
	provider := &AzureProvider{}
	integration.provider = provider

	assert.NotNil(t, integration.GetProvider())
	assert.Equal(t, provider, integration.GetProvider())
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

func TestMetadata_Struct(t *testing.T) {
	// Test that Metadata struct exists and can be instantiated
	metadata := Metadata{}
	assert.NotNil(t, metadata)
}

// Note: Testing Sync() would require:
// 1. Mock core.SyncContext
// 2. Valid OIDC token file setup
// 3. Mock Azure API responses
// These should be tested in integration tests, not unit tests

// Integration test example (commented out, requires real setup):
/*
func TestAzureIntegration_Sync_Integration(t *testing.T) {
	t.Skip("Requires real Azure credentials and OIDC token")

	// Setup OIDC token file
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "token")
	err := os.WriteFile(tokenFile, []byte("mock-token"), 0600)
	require.NoError(t, err)
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", tokenFile)

	integration := &AzureIntegration{}
	logger := logrus.NewEntry(logrus.New())

	// Mock SyncContext
	ctx := core.SyncContext{
		Logger: logger,
		Configuration: map[string]any{
			"tenantId":       "test-tenant-id",
			"clientId":       "test-client-id",
			"subscriptionId": "test-subscription-id",
		},
		Integration: mockIntegrationContext{},
	}

	err = integration.Sync(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, integration.GetProvider())
}
*/
