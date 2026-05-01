package organizations

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/protobuf/types/known/structpb"
)

const setupFlowIntegrationAppName = "setupflowtest"

func registerDevelopmentSetupFlowIntegration(t *testing.T, r *support.ResourceRegistry, provider core.IntegrationSetupProvider) {
	t.Helper()
	r.Registry.AppEnv = "development"
	r.Registry.Integrations[setupFlowIntegrationAppName] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.SetupProviders[setupFlowIntegrationAppName] = provider
}

func createSetupFlowIntegration(ctx context.Context, t *testing.T, r *support.ResourceRegistry, installationName string) string {
	t.Helper()
	appConfig, err := structpb.NewStruct(map[string]any{})
	require.NoError(t, err)
	resp, err := CreateIntegration(ctx, r.Registry, nil, "http://localhost", "http://localhost", r.Organization.ID.String(), setupFlowIntegrationAppName, installationName, appConfig)
	require.NoError(t, err)
	require.NotNil(t, resp.Integration)
	return resp.Integration.Metadata.Id
}

func seedIntegrationProperty(t *testing.T, integrationID uuid.UUID, def core.IntegrationPropertyDefinition) {
	t.Helper()
	var integration models.Integration
	require.NoError(t, database.Conn().Where("id = ?", integrationID).First(&integration).Error)
	integration.Properties = append(integration.Properties, def)
	require.NoError(t, database.Conn().Save(&integration).Error)
}

func seedIntegrationSecret(t *testing.T, r *support.ResourceRegistry, integrationID uuid.UUID, name, plaintext string) {
	t.Helper()
	enc, err := r.Encryptor.Encrypt(context.Background(), []byte(plaintext), []byte(integrationID.String()))
	require.NoError(t, err)
	now := time.Now()
	secret := models.IntegrationSecret{
		OrganizationID: r.Organization.ID,
		InstallationID: integrationID,
		Name:           name,
		Value:          enc,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Editable:       true,
	}
	require.NoError(t, database.Conn().Create(&secret).Error)
}
