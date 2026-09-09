package factories

import (
	"context"
	"errors"
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

func Test__ListFactoryRepositoryReviewBots(t *testing.T) {
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
		_, err := ListFactoryRepositoryReviewBots(ctx, deps, orgID, &pb.ListFactoryRepositoryReviewBotsRequest{
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

		response, err := ListFactoryRepositoryReviewBots(ctx, deps, orgID, &pb.ListFactoryRepositoryReviewBotsRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, "acme/api", response.GetRepository())
		assert.Empty(t, response.GetBots())
	})

	t.Run("serializes unique bots from GitHub", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		appRepo := "acme/api"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			VCSIntegrationID: &integrationID,
			AppRepository:    &appRepo,
		}))

		original := listRepositoryReviewBotsFromGitHub
		listRepositoryReviewBotsFromGitHub = func(context.Context, PRFeedbackDependencies, *gorm.DB, *models.Integration, string) ([]repositoryReviewBot, error) {
			return []repositoryReviewBot{
				{Login: "coderabbitai", DisplayName: "coderabbitai[bot]"},
			}, nil
		}
		t.Cleanup(func() { listRepositoryReviewBotsFromGitHub = original })

		response, err := ListFactoryRepositoryReviewBots(ctx, deps, orgID, &pb.ListFactoryRepositoryReviewBotsRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, response.GetBots(), 1)
		assert.Equal(t, "coderabbitai", response.GetBots()[0].GetLogin())
		assert.Equal(t, "coderabbitai[bot]", response.GetBots()[0].GetDisplayName())
	})

	t.Run("returns an empty catalog when GitHub listing fails", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		appRepo := "acme/api"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			VCSIntegrationID: &integrationID,
			AppRepository:    &appRepo,
		}))

		original := listRepositoryReviewBotsFromGitHub
		listRepositoryReviewBotsFromGitHub = func(context.Context, PRFeedbackDependencies, *gorm.DB, *models.Integration, string) ([]repositoryReviewBot, error) {
			return nil, errors.New("github unavailable")
		}
		t.Cleanup(func() { listRepositoryReviewBotsFromGitHub = original })

		response, err := ListFactoryRepositoryReviewBots(ctx, deps, orgID, &pb.ListFactoryRepositoryReviewBotsRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		assert.Empty(t, response.GetBots())
	})
}
