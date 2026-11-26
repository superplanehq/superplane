package factories

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func organizationKeyConfigurationFixture(t *testing.T) configurationFixture {
	t.Helper()
	f := newConfigurationFixture(t)
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-test")
	t.Setenv(config.EnvGitHubAppPrivateKey, "test-private-key")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "test-webhook-secret")
	db := database.DB(t.Context())
	const installation = int64(101)
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{Provider: models.ProviderGitHub, InstallationID: installation, AccountLogin: "acme"}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, installation, []models.VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/app", DefaultBranch: "main"},
		{RepositoryID: 202, FullName: "acme/backlog", DefaultBranch: "main"},
		{RepositoryID: 203, FullName: "acme/docs", DefaultBranch: "main"},
		{RepositoryID: 204, FullName: "acme/ungranted", DefaultBranch: "main"},
	}))
	id := uuid.MustParse(f.factory.OnboardingConfigValue().VCSIntegrationID)
	require.NoError(t, db.Create(&models.VCSProviderIntegrationBinding{IntegrationID: id, OrganizationID: f.factory.OrganizationID, Provider: models.ProviderGitHub, InstallationID: installation}).Error)
	for _, repository := range []int64{201, 202, 203} {
		require.NoError(t, models.GrantVCSProviderBindingRepository(db, id, models.ProviderGitHub, repository))
	}
	primary, backlog := int64(201), int64(202)
	require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepositoryID: &primary, BacklogRepositoryID: &backlog}))
	key, err := models.CreateAPIKey(db, f.factory.OrganizationID, "workspace automation", nil, f.r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, f.r.AuthService.AssignRole(key.ID.String(), models.RoleOrgAdmin, f.factory.OrganizationID.String(), models.DomainTypeOrganization))
	f.ctx = authentication.SetUserIdInMetadata(f.ctx, key.ID.String())
	f.ctx = withRepositoryBranchResolver(f.ctx, repositoryBranchResolver{gitHubBranches: func(string) ([]string, error) { return []string{"main", "develop"}, nil }})
	return f
}

func TestFactoryConfigurationOrganizationKeyRoundTripAndApply(t *testing.T) {
	f := organizationKeyConfigurationFixture(t)
	bundle := f.export(t)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		manifest.Stages[1].Automations[0].With["agents"].(map[string]any)[implementationAgentNodeID].(map[string]any)["additional_instructions"] = "Follow the runbook."
	})
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	live := resetTestLiveVersion(t, f.canvas)
	require.Equal(t, "Follow the runbook.", findIntakeNode(live.Nodes, implementationAgentNodeID).Configuration["additionalInstructions"])
}

func TestFactoryConfigurationOrganizationKeySelectsGrantedRepositories(t *testing.T) {
	f := organizationKeyConfigurationFixture(t)
	bundle := f.export(t)
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		manifest.Repositories.Primary.Repository = "acme/backlog"
		manifest.Repositories.BacklogRepository = "acme/app"
		manifest.Repositories.Additional = []factoryConfigurationContextRepository{{Repository: "acme/docs", Access: "read"}}
	})
	before, err := models.ListVCSProviderBindingRepositories(database.DB(t.Context()), uuid.MustParse(f.factory.OnboardingConfigValue().VCSIntegrationID))
	require.NoError(t, err)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	after, err := models.ListVCSProviderBindingRepositories(database.DB(t.Context()), uuid.MustParse(f.factory.OnboardingConfigValue().VCSIntegrationID))
	require.NoError(t, err)
	require.Equal(t, before, after, "plan must not change grants")
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	saved, err := models.FindFactory(database.DB(t.Context()), f.factory.OrganizationID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, "acme/backlog", saved.OnboardingConfigValue().AppRepository)
	require.Equal(t, "acme/app", saved.OnboardingConfigValue().BacklogRepository)
	resources, err := saved.ListContextRepos(database.DB(t.Context()))
	require.NoError(t, err)
	require.Len(t, resources, 1)
	require.Equal(t, "acme/docs", resources[0].Config.Data().Repository)
}

func TestFactoryConfigurationOrganizationKeyRejectsUnavailableGrants(t *testing.T) {
	for _, scenario := range []string{"ungranted primary", "ungranted backlog", "ungranted additional", "creator access", "other organization", "other connection", "removed repository", "suspended installation", "inactive connection", "deleted connection"} {
		t.Run(scenario, func(t *testing.T) {
			f := organizationKeyConfigurationFixture(t)
			db := database.DB(t.Context())
			id := uuid.MustParse(f.factory.OnboardingConfigValue().VCSIntegrationID)
			bundle := f.export(t)
			field := "repositories.primary.repository"
			switch scenario {
			case "ungranted primary", "creator access":
				if scenario == "creator access" {
					require.NoError(t, models.SaveAccountLinkedAccount(db, models.NewAccountLinkedAccount(f.r.Account.ID, models.ProviderGitHub, "42", "octocat", "Octocat", "")))
					require.NoError(t, models.ReplaceVCSProviderRepositoryCollaborators(db, models.ProviderGitHub, 204, []models.VCSProviderRepositoryCollaborator{{ProviderUserID: 42, ProviderLogin: "octocat"}}))
				}
				editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) { m.Repositories.Primary.Repository = "acme/ungranted" })
			case "ungranted backlog":
				field = "repositories.backlog_repository"
				editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) { m.Repositories.BacklogRepository = "acme/ungranted" })
			case "ungranted additional":
				field = "repositories.additional[0].repository"
				editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) {
					m.Repositories.Additional = []factoryConfigurationContextRepository{{Repository: "acme/ungranted", Access: "read"}}
				})
			case "other organization", "other connection":
				require.NoError(t, db.Where("integration_id = ? AND repository_id = ?", id, 201).Delete(&models.VCSProviderIntegrationRepository{}).Error)
				org := f.factory.OrganizationID
				if scenario == "other organization" {
					org = support.CreateOrganization(t, f.r, f.r.User).ID
				}
				connection := uuid.MustParse(createReadyOnboardingIntegration(t, org, models.ProviderGitHub))
				require.NoError(t, db.Create(&models.VCSProviderIntegrationBinding{IntegrationID: connection, OrganizationID: org, Provider: models.ProviderGitHub, InstallationID: 101}).Error)
				require.NoError(t, models.GrantVCSProviderBindingRepository(db, connection, models.ProviderGitHub, 201))
			case "removed repository":
				require.NoError(t, db.Where("provider = ? AND repository_id = ?", models.ProviderGitHub, 201).Delete(&models.VCSProviderRepository{}).Error)
			case "suspended installation":
				require.NoError(t, db.Model(&models.VCSProviderInstallation{}).Where("provider = ? AND installation_id = ?", models.ProviderGitHub, 101).Update("suspended_at", time.Now()).Error)
			case "inactive connection":
				require.NoError(t, db.Model(&models.Integration{}).Where("id = ?", id).Update("state", models.IntegrationStateError).Error)
			case "deleted connection":
				require.NoError(t, db.Delete(&models.Integration{ID: id}).Error)
			}
			plan := f.plan(t, bundle)
			require.Len(t, plan.Diagnostics, 1)
			if scenario != "inactive connection" && scenario != "deleted connection" {
				require.Equal(t, field, plan.Diagnostics[0].Field)
				require.Equal(t, organizationKeyRepositoryDenied, plan.Diagnostics[0].Message)
			}
			before := configurationResourceCounts(t)
			_, err := f.apply(bundle, "unused")
			require.Error(t, err)
			require.Equal(t, before, configurationResourceCounts(t))
		})
	}
}

func TestFactoryConfigurationOrganizationKeyRechecksRevokedGrants(t *testing.T) {
	f := organizationKeyConfigurationFixture(t)
	bundle := f.export(t)
	editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) {
		m.Repositories.Primary.Repository = "acme/backlog"
		m.Repositories.BacklogRepository = "acme/app"
	})
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	db := database.DB(t.Context())
	id := uuid.MustParse(f.factory.OnboardingConfigValue().VCSIntegrationID)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := models.LockVCSProviderBindingRepositories(tx, id); err != nil {
			return err
		}
		return tx.Where("integration_id = ? AND repository_id = ?", id, 202).Delete(&models.VCSProviderIntegrationRepository{}).Error
	}))
	before := configurationResourceCounts(t)
	versions := configurationRevisionCount(t, f.canvas)
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.ErrorContains(t, err, organizationKeyRepositoryDenied)
	saved, err := models.FindFactory(db, f.factory.OrganizationID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, f.factory.OnboardingConfigValue(), saved.OnboardingConfigValue())
	require.Equal(t, before, configurationResourceCounts(t))
	require.Equal(t, versions, configurationRevisionCount(t, f.canvas))
}

func TestFactoryConfigurationRepositoryLookupFailureIsInternal(t *testing.T) {
	f := organizationKeyConfigurationFixture(t)
	db := database.DB(t.Context())
	bundle := f.export(t)
	const callback = "organization_key_repository_lookup_failure"
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "repository" {
			tx.AddError(errors.New("repository lookup unavailable"))
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callback) })
	response, err := PlanFactoryConfiguration(f.ctx, testIntakeDependencies(f.r), f.factory.OrganizationID.String(), &pb.PlanFactoryConfigurationRequest{FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(), Bundle: bundle})
	require.Nil(t, response)
	require.Equal(t, codes.Internal, grpcerrors.Code(err))
}

func TestOrganizationKeyDirectRepositoryAPIs(t *testing.T) {
	for _, api := range []string{"select", "update", "context", "update context", "branch", "onboarding"} {
		for _, granted := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/granted=%t", api, granted), func(t *testing.T) {
				f := organizationKeyConfigurationFixture(t)
				repository, id := "acme/backlog", int64(202)
				if !granted {
					repository, id = "acme/ungranted", 204
				}
				deps := testIntakeDependencies(f.r)
				var err error
				switch api {
				case "select":
					_, err = SelectFactoryVCSProviderRepository(f.ctx, deps, f.factory.OrganizationID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{Id: f.factory.ID.String(), Provider: models.ProviderGitHub, RepositoryId: id})
				case "update":
					_, err = UpdateFactoryRepository(f.ctx, deps, f.factory.OrganizationID.String(), &pb.UpdateFactoryRepositoryRequest{Id: f.factory.ID.String(), Repository: repository, DefaultBranch: "develop"})
				case "context":
					_, err = CreateFactoryAgentResource(f.ctx, deps, f.factory.OrganizationID.String(), &pb.CreateFactoryAgentResourceRequest{FactoryId: f.factory.ID.String(), Kind: pb.FactoryAgentResource_KIND_CONTEXT_REPO, Repository: repository, Access: "read", Enabled: true})
				case "update context":
					resource, createErr := f.factory.CreateAgentResource(database.DB(t.Context()), models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{Repository: "acme/docs", Access: "read"})
					require.NoError(t, createErr)
					_, err = UpdateFactoryAgentResource(f.ctx, deps, f.factory.OrganizationID.String(), &pb.UpdateFactoryAgentResourceRequest{FactoryId: f.factory.ID.String(), ResourceId: resource.ID.String(), Repository: &repository})
				case "branch":
					branch := "develop"
					patch := models.FactoryOnboardingPatch{AppRepository: &repository, DefaultBranch: &branch}
					_, err = verifiedOnboardingDefaultBranch(f.ctx, deps, database.DB(t.Context()), f.factory.OrganizationID, f.factory.OrganizationID.String(), f.factory, patch)
				case "onboarding":
					_, err = UpdateFactoryOnboarding(f.ctx, deps, f.factory.OrganizationID.String(), &pb.UpdateFactoryOnboardingRequest{Id: f.factory.ID.String(), AppRepository: &repository})
				}
				if granted {
					require.NoError(t, err)
					return
				}
				require.Error(t, err)
				require.ErrorContains(t, err, organizationKeyRepositoryDenied)
			})
		}
	}
}

func TestFactoryConfigurationPersonalTokenRetainsGitHubAccessChecks(t *testing.T) {
	f := organizationKeyConfigurationFixture(t)
	f.ctx = authentication.SetUserIdInMetadata(f.ctx, f.r.User.String())
	db := database.DB(t.Context())
	bundle := f.export(t)
	editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) { m.Repositories.Primary.Repository = "acme/ungranted" })
	// A personal token with no linked GitHub account gets a distinct precondition failure.
	_, err := PlanFactoryConfiguration(f.ctx, testIntakeDependencies(f.r), f.factory.OrganizationID.String(), &pb.PlanFactoryConfigurationRequest{FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(), Bundle: bundle})
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.NoError(t, models.SaveAccountLinkedAccount(db, models.NewAccountLinkedAccount(f.r.Account.ID, models.ProviderGitHub, "42", "octocat", "Octocat", "")))
	plan := f.plan(t, bundle)
	require.Len(t, plan.Diagnostics, 1, "an installation grant does not replace the owner's GitHub access")
	for _, id := range []int64{201, 202, 204} {
		require.NoError(t, models.ReplaceVCSProviderRepositoryCollaborators(db, models.ProviderGitHub, id, []models.VCSProviderRepositoryCollaborator{{ProviderUserID: 42, ProviderLogin: "octocat"}}))
	}
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	grants, err := models.ListVCSProviderBindingRepositories(db, uuid.MustParse(f.factory.OnboardingConfigValue().VCSIntegrationID))
	require.NoError(t, err)
	for _, grant := range grants {
		require.NotEqual(t, int64(204), grant.RepositoryID, "plan must not grant the new repository")
	}
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	saved, err := models.FindFactory(db, f.factory.OrganizationID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, "acme/ungranted", saved.OnboardingConfigValue().AppRepository)
}

func TestOrganizationKeyRepositoryMutationsWaitForGrantRevocation(t *testing.T) {
	for _, mutation := range []string{"apply", "select", "update", "context"} {
		t.Run(mutation, func(t *testing.T) {
			f := organizationKeyConfigurationFixture(t)
			db := database.DB(t.Context())
			bundle := f.export(t)
			editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) { m.Repositories.Primary.Repository = "acme/backlog" })
			plan := f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			before := configurationResourceCounts(t)
			removal := db.Begin()
			require.NoError(t, removal.Error)
			defer removal.Rollback()
			id := uuid.MustParse(f.factory.OnboardingConfigValue().VCSIntegrationID)
			require.NoError(t, models.LockVCSProviderBindingRepositories(removal, id))
			require.NoError(t, removal.Where("integration_id = ? AND repository_id = ?", id, 202).Delete(&models.VCSProviderIntegrationRepository{}).Error)
			done := make(chan error, 1)
			go func() {
				deps := testIntakeDependencies(f.r)
				var err error
				switch mutation {
				case "apply":
					_, err = f.apply(bundle, plan.BaseStateToken)
				case "select":
					_, err = SelectFactoryVCSProviderRepository(f.ctx, deps, f.factory.OrganizationID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{Id: f.factory.ID.String(), Provider: models.ProviderGitHub, RepositoryId: 202})
				case "update":
					_, err = UpdateFactoryRepository(f.ctx, deps, f.factory.OrganizationID.String(), &pb.UpdateFactoryRepositoryRequest{Id: f.factory.ID.String(), Repository: "acme/backlog", DefaultBranch: "main"})
				case "context":
					_, err = CreateFactoryAgentResource(f.ctx, deps, f.factory.OrganizationID.String(), &pb.CreateFactoryAgentResourceRequest{FactoryId: f.factory.ID.String(), Kind: pb.FactoryAgentResource_KIND_CONTEXT_REPO, Repository: "acme/backlog", Access: "read", Enabled: true})
				}
				done <- err
			}()
			// Observe the actual database waiter, so a slow goroutine cannot pass this check.
			require.Eventually(t, func() bool {
				var waiting bool
				err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND NOT granted AND objid::bigint = (hashtext(?)::bigint & 4294967295))", "vcs-provider-binding-repositories:"+id.String()).Scan(&waiting).Error
				return err == nil && waiting
			}, 2*time.Second, 10*time.Millisecond)
			require.NoError(t, removal.Commit().Error)
			select {
			case err := <-done:
				require.Error(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("repository mutation did not finish after revocation")
			}
			saved, err := models.FindFactory(db, f.factory.OrganizationID, f.factory.ID)
			require.NoError(t, err)
			require.Equal(t, f.factory.OnboardingConfigValue(), saved.OnboardingConfigValue())
			require.Equal(t, before, configurationResourceCounts(t))
		})
	}
}

func TestOrganizationKeyContextRepositoryUsesCurrentConnectionAfterLock(t *testing.T) {
	f := organizationKeyConfigurationFixture(t)
	db := database.DB(t.Context())
	previous := uuid.MustParse(f.factory.OnboardingConfigValue().VCSIntegrationID)
	next := uuid.MustParse(createReadyOnboardingIntegration(t, f.factory.OrganizationID, models.ProviderGitHub))
	require.NoError(t, db.Create(&models.VCSProviderIntegrationBinding{IntegrationID: next, OrganizationID: f.factory.OrganizationID, Provider: models.ProviderGitHub, InstallationID: 101}).Error)
	blocker := db.Begin()
	require.NoError(t, blocker.Error)
	defer blocker.Rollback()
	require.NoError(t, models.LockOrganizationVCSProviderBindingRepositories(blocker, f.factory.OrganizationID))
	done := make(chan error, 1)
	go func() {
		_, err := CreateFactoryAgentResource(f.ctx, testIntakeDependencies(f.r), f.factory.OrganizationID.String(), &pb.CreateFactoryAgentResourceRequest{FactoryId: f.factory.ID.String(), Kind: pb.FactoryAgentResource_KIND_CONTEXT_REPO, Repository: "acme/docs", Access: "read", Enabled: true})
		done <- err
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND NOT granted AND objid::bigint IN ((hashtext(?)::bigint & 4294967295), (hashtext(?)::bigint & 4294967295)))", "vcs-provider-binding-repositories:"+previous.String(), "vcs-provider-binding-repositories:"+next.String()).Scan(&waiting).Error
		return err == nil && waiting
	}, 2*time.Second, 10*time.Millisecond)
	nextID := next.String()
	require.NoError(t, f.factory.UpdateOnboarding(blocker, models.FactoryOnboardingPatch{VCSIntegrationID: &nextID}))
	require.NoError(t, blocker.Commit().Error)
	select {
	case err := <-done:
		require.ErrorContains(t, err, organizationKeyRepositoryDenied)
	case <-time.After(5 * time.Second):
		t.Fatal("repository mutation did not finish after connection change")
	}
	resources, err := f.factory.ListContextRepos(db)
	require.NoError(t, err)
	require.Empty(t, resources)
}

func TestOrganizationKeyResolvesGrantedConnectionWithoutExpandingRetainedAccess(t *testing.T) {
	for _, scenario := range []struct {
		name, api               string
		backlogGranted, context bool
		allowed                 bool
	}{
		{"select resolves granted connection", "select", true, false, true},
		{"select rejects retained backlog on another connection", "select", false, false, false},
		{"update resolves granted connection", "update", false, false, true},
		{"select rejects retained context on another connection", "select", true, true, false},
		{"update rejects retained context on another connection", "update", false, true, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := organizationKeyConfigurationFixture(t)
			db := database.DB(t.Context())
			next := uuid.MustParse(createReadyOnboardingIntegration(t, f.factory.OrganizationID, models.ProviderGitHub))
			require.NoError(t, db.Create(&models.VCSProviderIntegrationBinding{IntegrationID: next, OrganizationID: f.factory.OrganizationID, Provider: models.ProviderGitHub, InstallationID: 101}).Error)
			require.NoError(t, models.GrantVCSProviderBindingRepository(db, next, models.ProviderGitHub, 204))
			if scenario.backlogGranted {
				require.NoError(t, models.GrantVCSProviderBindingRepository(db, next, models.ProviderGitHub, 202))
			}
			if scenario.context {
				_, err := f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{Repository: "acme/docs", Access: "write"})
				require.NoError(t, err)
			}
			var err error
			if scenario.api == "select" {
				_, err = SelectFactoryVCSProviderRepository(f.ctx, testIntakeDependencies(f.r), f.factory.OrganizationID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{Id: f.factory.ID.String(), Provider: models.ProviderGitHub, RepositoryId: 204})
			} else {
				_, err = UpdateFactoryRepository(f.ctx, testIntakeDependencies(f.r), f.factory.OrganizationID.String(), &pb.UpdateFactoryRepositoryRequest{Id: f.factory.ID.String(), Repository: "acme/ungranted", DefaultBranch: "main"})
			}
			saved, loadErr := models.FindFactory(db, f.factory.OrganizationID, f.factory.ID)
			require.NoError(t, loadErr)
			if scenario.allowed {
				require.NoError(t, err)
				require.Equal(t, next.String(), saved.OnboardingConfigValue().VCSIntegrationID)
				return
			}
			require.ErrorContains(t, err, organizationKeyRepositoryDenied)
			require.Equal(t, f.factory.OnboardingConfigValue(), saved.OnboardingConfigValue())
			grants, loadErr := models.ListVCSProviderBindingRepositories(db, next)
			require.NoError(t, loadErr)
			for _, grant := range grants {
				require.NotEqual(t, int64(203), grant.RepositoryID)
			}
		})
	}
}

func TestFactoryConfigurationBitbucketRepositoryDenialIsDiagnostic(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	id := uuid.MustParse(f.factory.OnboardingConfigValue().VCSIntegrationID)
	provider := models.ProviderBitbucket
	require.NoError(t, db.Model(&models.Integration{}).Where("id = ?", id).Update("app_name", provider).Error)
	configureConfigurationCatalogIntegration(t, f, id.String(), provider)
	integration, err := models.FindIntegrationInTransaction(db, f.factory.OrganizationID, id)
	require.NoError(t, err)
	metadata := integration.Metadata.Data()
	// A connection without its synced workspace metadata cannot authorize repositories.
	delete(metadata, "workspace")
	require.NoError(t, db.Model(integration).Update("metadata", metadata).Error)
	require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSProvider: &provider}))
	plan := f.plan(t, f.export(t))
	require.Len(t, plan.Diagnostics, 1)
	require.Equal(t, "repositories.primary.repository", plan.Diagnostics[0].Field)
	require.Empty(t, plan.BaseStateToken)
}
