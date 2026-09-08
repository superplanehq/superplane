package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__ListFactoryRepositoryStatusChecks(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		GitProvider:    r.GitProvider,
		WebhookBaseURL: "http://localhost:8000",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	t.Run("fails when the factory has no repository", func(t *testing.T) {
		factory := newFactory(t)
		_, err := ListFactoryRepositoryStatusChecks(ctx, deps, orgID, &pb.ListFactoryRepositoryStatusChecksRequest{
			FactoryId: factory.ID.String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("returns an empty catalog when GitHub is not connected", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/api"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))

		response, err := ListFactoryRepositoryStatusChecks(ctx, deps, orgID, &pb.ListFactoryRepositoryStatusChecksRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, "acme/api", response.GetRepository())
		assert.Empty(t, response.GetChecks())
	})

	t.Run("serializes required and observed checks from GitHub", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		appRepo := "acme/api"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			VCSIntegrationID: &integrationID,
			AppRepository:    &appRepo,
		}))

		original := listRepositoryStatusChecksFromGitHub
		listRepositoryStatusChecksFromGitHub = func(context.Context, PRFeedbackDependencies, *gorm.DB, *models.Integration, string) ([]repositoryStatusCheck, error) {
			return mergeRepositoryStatusChecks(
				[]string{"lint"},
				[]observedRepositoryStatusCheck{
					{Name: "unit", DetailsURL: "https://acme.semaphoreci.com/jobs/1"},
				},
			), nil
		}
		t.Cleanup(func() { listRepositoryStatusChecksFromGitHub = original })

		response, err := ListFactoryRepositoryStatusChecks(ctx, deps, orgID, &pb.ListFactoryRepositoryStatusChecksRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, response.GetChecks(), 2)
		assert.Equal(t, "lint", response.GetChecks()[0].GetName())
		assert.True(t, response.GetChecks()[0].GetRequired())
		assert.Equal(t, "unit", response.GetChecks()[1].GetName())
		assert.Equal(t, "semaphore", response.GetChecks()[1].GetSuggestedIntegration())
	})
}
