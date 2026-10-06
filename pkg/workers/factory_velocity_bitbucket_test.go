package workers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestFactoryVelocitySync_BitbucketFactory(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	provider := models.ProviderBitbucket
	integrationID := uuid.New().String()
	repository := "acme/app"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSProvider:      &provider,
		VCSIntegrationID: &integrationID,
		AppRepository:    &repository,
	}))

	worker := NewFactoryVelocitySyncWorker("", nil, nil)
	synced, err := worker.SyncFactory(t.Context(), factory.ID)
	require.NoError(t, err)
	assert.True(t, synced)

	stored, err := models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	assert.Equal(t, vcs.ErrNotSupported.Error(), stored.Error)
}
