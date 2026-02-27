package azure

import (
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
	"github.com/superplanehq/superplane/pkg/oidc"
)

// mockOIDCProvider implements oidc.Provider for testing.
type mockOIDCProvider struct{}

func (m *mockOIDCProvider) Sign(subject string, duration time.Duration, audience string, additionalClaims map[string]any) (string, error) {
	return "mock-jwt-token", nil
}

func (m *mockOIDCProvider) PublicJWKs() []oidc.PublicJWK {
	return nil
}

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

func TestAzureIntegration_SetOIDCProvider(t *testing.T) {
	integration := &AzureIntegration{}

	assert.Nil(t, integration.oidcProvider)

	mockOIDC := &mockOIDCProvider{}
	integration.SetOIDCProvider(mockOIDC)

	assert.NotNil(t, integration.oidcProvider)
	assert.Equal(t, mockOIDC, integration.oidcProvider)
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
