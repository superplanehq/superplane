package organizations

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

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
