package factories

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/bitbucketapp"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestSelectBitbucketRepositoryUsesTheForgeInstallation(t *testing.T) {
	t.Setenv(config.EnvBitbucketForgeAppID, "ari:cloud:ecosystem::app/example")
	t.Setenv(config.EnvBitbucketForgeInstallURL, "https://developer.atlassian.com/console/install/example")

	r := support.Setup(t)
	db := database.DB(t.Context())
	const accountID = "11111111-1111-1111-1111-111111111111"
	const repositoryID = "22222222-2222-2222-2222-222222222222"
	installationID := uuid.NewString()
	now := time.Now()
	require.NoError(t, models.SaveAccountLinkedAccount(db, models.NewAccountLinkedAccount(
		r.Account.ID,
		models.ProviderBitbucket,
		accountID,
		"ada-bb",
		"Ada",
		"",
	)))
	require.NoError(t, db.Create(&models.BitbucketForgeInstallation{
		InstallationID: installationID,
		WorkspaceUUID:  "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		WorkspaceSlug:  "acme",
		SystemToken:    []byte("ciphertext"),
		TokenExpiresAt: now.Add(2 * time.Hour),
		LastDeliveryAt: now,
		InstalledAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)
	bitbucketapp.SetSystemTokenSource(func(id string) (string, time.Time, error) {
		if id != installationID {
			return "", time.Time{}, bitbucketapp.ErrReconnect
		}
		return "system-token", now.Add(2 * time.Hour), nil
	})
	t.Cleanup(func() { bitbucketapp.SetSystemTokenSource(nil) })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "Bearer system-token", request.Header.Get("Authorization"))
		switch request.URL.Path {
		case "/workspaces/acme/permissions/repositories":
			assert.Equal(t, `user.uuid="{11111111-1111-1111-1111-111111111111}"`, request.URL.Query().Get("q"))
			_, _ = w.Write([]byte(`{"values":[{"type":"repository_permission","permission":"write","repository":{"uuid":"{22222222-2222-2222-2222-222222222222}","full_name":"acme/api"}}]}`))
		case "/repositories/acme":
			_, _ = w.Write([]byte(`{"values":[{"uuid":"{22222222-2222-2222-2222-222222222222}","full_name":"acme/api","is_private":true,"mainbranch":{"name":"develop"}}]}`))
		default:
			http.NotFound(w, request)
		}
	}))
	t.Cleanup(server.Close)
	restore := bitbucket.UseDirectory(bitbucket.Directory{BaseURL: server.URL, HTTP: server.Client()})
	t.Cleanup(restore)

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	response, err := SelectFactoryVCSProviderRepository(ctx, r.Organization.ID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{
		Id:         factory.ID.String(),
		Provider:   models.ProviderBitbucket,
		Repository: "acme/api",
	})
	require.NoError(t, err)
	assert.Equal(t, models.ProviderBitbucket, response.Factory.Onboarding.VcsProvider)
	assert.Equal(t, "acme/api", response.Factory.Onboarding.AppRepository)
	assert.Equal(t, repositoryID, response.Factory.Onboarding.AppRepositoryExternalId)
	assert.Equal(t, int64(0), response.Factory.Onboarding.AppRepositoryId)
	assert.Equal(t, "develop", response.Factory.Onboarding.DefaultBranch)
	assert.NotEmpty(t, response.Factory.Onboarding.VcsIntegrationId)
}

func TestUpdateFactoryRepositoryBitbucketDoesNotRequireGitHub(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, models.ProviderBitbucket)
	require.NoError(t, db.Model(&models.Integration{}).
		Where("id = ?", integrationID).
		Update("metadata", datatypes.NewJSONType(map[string]any{
			"authType": "workspaceAccessToken",
			"workspace": map[string]any{
				"uuid": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				"name": "acme",
				"slug": "acme",
			},
		})).Error)
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	provider := models.ProviderBitbucket
	repository := "acme/api"
	backlog := "acme/api"
	branch := "main"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSProvider:       &provider,
		VCSIntegrationID:  &integrationID,
		AppRepository:     &repository,
		BacklogRepository: &backlog,
		DefaultBranch:     &branch,
	}))

	ctx := withRepositoryBranchResolver(
		authentication.SetUserIdInMetadata(context.Background(), r.User.String()),
		repositoryBranchResolver{bitbucketHead: func(_, branch string) (string, error) {
			if branch == "develop" {
				return "abc123", nil
			}
			return "", nil
		}},
	)
	updated, err := UpdateFactoryRepository(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryRepositoryRequest{
		Id:            factory.ID.String(),
		Repository:    "acme/web",
		DefaultBranch: "develop",
	})
	require.NoError(t, err)
	assert.Equal(t, "acme/web", updated.Factory.Onboarding.AppRepository)
	assert.Equal(t, "develop", updated.Factory.Onboarding.DefaultBranch)
	assert.Equal(t, models.ProviderBitbucket, updated.Factory.Onboarding.VcsProvider)

	_, err = UpdateFactoryRepository(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryRepositoryRequest{
		Id:            factory.ID.String(),
		Repository:    "other/web",
		DefaultBranch: "develop",
	})
	require.ErrorContains(t, err, "not accessible")
}

func TestUpdateFactoryRepositoryBitbucketKeepsRequestedCatalogBranch(t *testing.T) {
	t.Setenv(config.EnvBitbucketForgeAppID, "ari:cloud:ecosystem::app/example")
	t.Setenv(config.EnvBitbucketForgeInstallURL, "https://developer.atlassian.com/console/install/example")

	r := support.Setup(t)
	db := database.DB(t.Context())
	const accountID = "11111111-1111-1111-1111-111111111111"
	installationID := uuid.NewString()
	now := time.Now()
	require.NoError(t, models.SaveAccountLinkedAccount(db, models.NewAccountLinkedAccount(
		r.Account.ID, models.ProviderBitbucket, accountID, "ada-bb", "Ada", "",
	)))
	require.NoError(t, db.Create(&models.BitbucketForgeInstallation{
		InstallationID: installationID,
		WorkspaceUUID:  "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		WorkspaceSlug:  "acme",
		SystemToken:    []byte("ciphertext"),
		TokenExpiresAt: now.Add(2 * time.Hour),
		LastDeliveryAt: now,
		InstalledAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)
	bitbucketapp.SetSystemTokenSource(func(id string) (string, time.Time, error) {
		if id != installationID {
			return "", time.Time{}, bitbucketapp.ErrReconnect
		}
		return "system-token", now.Add(2 * time.Hour), nil
	})
	t.Cleanup(func() { bitbucketapp.SetSystemTokenSource(nil) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/workspaces/acme/permissions/repositories":
			_, _ = w.Write([]byte(`{"values":[{"type":"repository_permission","permission":"write","repository":{"uuid":"{22222222-2222-2222-2222-222222222222}","full_name":"acme/api"}}]}`))
		case "/repositories/acme":
			_, _ = w.Write([]byte(`{"values":[{"uuid":"{22222222-2222-2222-2222-222222222222}","full_name":"acme/api","is_private":true,"mainbranch":{"name":"main"}}]}`))
		default:
			http.NotFound(w, request)
		}
	}))
	t.Cleanup(server.Close)
	restore := bitbucket.UseDirectory(bitbucket.Directory{BaseURL: server.URL, HTTP: server.Client()})
	t.Cleanup(restore)

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	selected, err := SelectFactoryVCSProviderRepository(ctx, r.Organization.ID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{
		Id: factory.ID.String(), Provider: models.ProviderBitbucket, Repository: "acme/api",
	})
	require.NoError(t, err)
	require.Equal(t, "main", selected.Factory.Onboarding.DefaultBranch)

	ctx = withRepositoryBranchResolver(ctx, repositoryBranchResolver{
		bitbucketHead: func(repository, branch string) (string, error) {
			if repository == "acme/api" && branch == "develop" {
				return "abc123", nil
			}
			return "", nil
		},
	})
	updated, err := UpdateFactoryRepository(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryRepositoryRequest{
		Id: factory.ID.String(), Repository: "acme/api", DefaultBranch: "develop",
	})
	require.NoError(t, err)
	assert.Equal(t, "acme/api", updated.Factory.Onboarding.AppRepository)
	assert.Equal(t, "develop", updated.Factory.Onboarding.DefaultBranch)
	assert.Equal(t, selected.Factory.Onboarding.VcsIntegrationId, updated.Factory.Onboarding.VcsIntegrationId)
}

func TestUpdateFactoryRepositoryBitbucketRejectsUnknownBranch(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := tokenBitbucketFactory(t, r, "main")
	ctx := withRepositoryBranchResolver(
		authentication.SetUserIdInMetadata(context.Background(), r.User.String()),
		repositoryBranchResolver{bitbucketHead: func(_, _ string) (string, error) { return "", nil }},
	)
	_, err := UpdateFactoryRepository(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryRepositoryRequest{
		Id: factory.ID.String(), Repository: "acme/api", DefaultBranch: "missing",
	})
	require.ErrorContains(t, err, "branch does not exist")
	reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	assert.Equal(t, "main", reloaded.OnboardingConfigValue().DefaultBranch)
}

func TestUpdateFactoryRepositoryBitbucketBranchSaveKeepsStartedTasks(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := tokenBitbucketFactory(t, r, "main")
	started, err := factory.CreateWorkOrder(db, "Started", "", &r.User, nil, nil)
	require.NoError(t, err)
	ctx := withRepositoryBranchResolver(
		authentication.SetUserIdInMetadata(context.Background(), r.User.String()),
		repositoryBranchResolver{bitbucketHead: func(_, branch string) (string, error) {
			if branch == "develop" {
				return "abc123", nil
			}
			return "", nil
		}},
	)
	updated, err := UpdateFactoryRepository(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryRepositoryRequest{
		Id: factory.ID.String(), Repository: "acme/api", DefaultBranch: "develop",
	})
	require.NoError(t, err)
	assert.Equal(t, "acme/api", updated.Factory.Onboarding.AppRepository)
	assert.Equal(t, "develop", updated.Factory.Onboarding.DefaultBranch)

	var kept models.FactoryWorkOrder
	require.NoError(t, db.First(&kept, "id = ?", started.ID).Error)
	require.NotNil(t, kept.DefaultBranch)
	assert.Equal(t, "main", *kept.DefaultBranch)
	reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	created, err := reloaded.CreateWorkOrder(db, "Next", "", &r.User, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, created.DefaultBranch)
	assert.Equal(t, "develop", *created.DefaultBranch)
	assert.Equal(t, "acme/api", *created.Repository)
}

func tokenBitbucketFactory(t *testing.T, r *support.ResourceRegistry, branch string) *models.Factory {
	t.Helper()
	db := database.DB(t.Context())
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, models.ProviderBitbucket)
	require.NoError(t, db.Model(&models.Integration{}).Where("id = ?", integrationID).Update("metadata", datatypes.NewJSONType(map[string]any{
		"authType":  "workspaceAccessToken",
		"workspace": map[string]any{"uuid": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "name": "acme", "slug": "acme"},
	})).Error)
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	provider := models.ProviderBitbucket
	repository := "acme/api"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSProvider: &provider, VCSIntegrationID: &integrationID,
		AppRepository: &repository, BacklogRepository: &repository, DefaultBranch: &branch,
	}))
	reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	return reloaded
}
