package factories

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	bitbucketintegration "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// The request has read the resource when it starts waiting for the workspace lock.
func TestContextRepositoryUpdateReloadsAfterLock(t *testing.T) {
	for _, scenario := range []string{"partial update", "deleted", "enable current repository", "deny current repository", "deny partial update"} {
		t.Run(scenario, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			requestContext := t.Context()
			denied := scenario == "deny current repository" || scenario == "deny partial update"
			if denied {
				f := organizationKeyConfigurationFixture(t)
				r, factory, requestContext = f.r, f.factory, f.ctx
			}
			resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", false, models.FactoryAgentResourceConfig{Repository: "acme/old", Ref: "old"})
			require.NoError(t, err)
			if denied {
				config := resource.Config.Data()
				config.Repository = "acme/app"
				require.NoError(t, resource.Update(db, nil, nil, &config))
			}
			locked := db.Begin()
			defer locked.Rollback()
			require.NoError(t, factory.LockForUpdate(locked))
			waiting := make(chan struct{}, 1)
			type requestKey struct{}
			const callback = "test:waiting_resource_update"
			require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
				if tx.Statement.Context.Value(requestKey{}) == true && tx.Statement.Table == "factories" {
					if _, ok := tx.Statement.Clauses["FOR"]; ok {
						select {
						case waiting <- struct{}{}:
						default:
						}
					}
				}
			}))
			defer db.Callback().Query().Remove(callback)
			enabled := true
			ref := "requested"
			req := &pb.UpdateFactoryAgentResourceRequest{FactoryId: factory.ID.String(), ResourceId: resource.ID.String(), Ref: &ref}
			if scenario == "enable current repository" || scenario == "deny current repository" {
				req.Ref = nil
				req.Enabled = &enabled
			}
			done := make(chan error, 1)
			go func() {
				_, err := UpdateFactoryAgentResource(context.WithValue(requestContext, requestKey{}, true), IntakeDependencies{}, r.Organization.ID.String(), req)
				done <- err
			}()
			select {
			case <-waiting:
			case <-time.After(5 * time.Second):
				t.Fatal("request did not wait for workspace lock")
			}
			if scenario == "deleted" {
				require.NoError(t, resource.Delete(locked))
			} else {
				config := resource.Config.Data()
				config.Repository = "acme/current"
				if denied {
					config.Repository = "acme/ungranted"
				}
				config.Ref = "current"
				require.NoError(t, resource.Update(locked, nil, nil, &config))
			}
			require.NoError(t, locked.Commit().Error)
			select {
			case err := <-done:
				if scenario == "deleted" {
					require.Equal(t, codes.NotFound, grpcerrors.Code(err))
					return
				}
				if denied {
					require.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
					saved, err := factory.FindAgentResource(db, resource.ID)
					require.NoError(t, err)
					require.False(t, saved.Enabled)
					require.Equal(t, "current", saved.Config.Data().Ref)
					return
				}
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("update did not finish")
			}
			saved, err := factory.FindAgentResource(db, resource.ID)
			require.NoError(t, err)
			require.Equal(t, "acme/current", saved.Config.Data().Repository)
			if req.Ref != nil {
				require.Equal(t, ref, saved.Config.Data().Ref)
			} else {
				require.True(t, saved.Enabled)
				require.Equal(t, "current", saved.Config.Data().Ref)
			}
		})
	}
}

func TestFactoryConfigurationProviderChecksReleaseLocks(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) { m.Repositories.Primary.Ref = "next" })
	f.ctx = withRepositoryBranchResolver(f.ctx, repositoryBranchResolver{gitHubBranches: func(string) ([]string, error) { return []string{"next"}, nil }})
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	f.ctx = withRepositoryBranchResolver(f.ctx, repositoryBranchResolver{gitHubBranches: func(string) ([]string, error) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
			if err := models.LockOrganizationVCSProviderBindingRepositories(tx, f.r.Organization.ID); err != nil {
				return err
			}
			current := *f.factory
			return current.LockForUpdate(tx)
		})
		require.NoError(t, err, "provider validation must release workspace locks")
		return []string{"next"}, nil
	}})
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
}

func TestFactoryConfigurationRechecksGrantsUnderMutationLocks(t *testing.T) {
	f := organizationKeyConfigurationFixture(t)
	db := database.DB(t.Context())
	bundle := f.export(t)
	before, err := models.FindFactory(db, f.factory.OrganizationID, f.factory.ID)
	require.NoError(t, err)
	editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) {
		m.Repositories.Primary.Ref = "develop"
		m.Stages[0].Name = "Must not be saved"
	})
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	const callback = "test:revoke_repository_before_mutation"
	revoked := false
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "factories" || revoked {
			return
		}
		if _, locking := tx.Statement.Clauses["FOR"]; !locking {
			return
		}
		revoked = true
		// This is a committed revocation after preparation. It must be visible to
		// authorization performed while apply holds the mutation locks.
		require.NoError(t, db.Where("integration_id = ? AND provider = ? AND repository_id = ?", f.factory.OnboardingConfigValue().VCSIntegrationID, models.ProviderGitHub, 201).Delete(&models.VCSProviderIntegrationRepository{}).Error)
	}))
	defer db.Callback().Query().Remove(callback)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.Error(t, err)
	require.True(t, revoked)
	saved, err := models.FindFactory(db, f.factory.OrganizationID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, before.OnboardingConfig, saved.OnboardingConfig)
	line, err := f.factory.FindLine(db, f.line.ID)
	require.NoError(t, err)
	require.Equal(t, f.line.StagesValue(), line.StagesValue(), "failed authorization must roll back all YAML changes")
}

func TestFactoryConfigurationRejectsChangedProviderInputs(t *testing.T) {
	for _, changedInput := range []string{"configuration", "credentials"} {
		t.Run(changedInput, func(t *testing.T) {
			f := newConfigurationFixture(t)
			db := database.DB(t.Context())
			bundle := f.export(t)
			editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) { m.Repositories.Primary.Ref = "next" })
			f.ctx = withRepositoryBranchResolver(f.ctx, repositoryBranchResolver{gitHubBranches: func(string) ([]string, error) { return []string{"next"}, nil }})
			plan := f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			changed := false
			f.ctx = withRepositoryBranchResolver(f.ctx, repositoryBranchResolver{gitHubBranches: func(string) ([]string, error) {
				if !changed {
					changed = true
					integration, err := models.FindIntegrationInTransaction(db, f.factory.OrganizationID, uuid.MustParse(f.factory.OnboardingConfigValue().VCSIntegrationID))
					require.NoError(t, err)
					if changedInput == "credentials" {
						secrets, err := models.ListIntegrationSecretsInTransaction(db, integration.ID)
						require.NoError(t, err)
						require.NotEmpty(t, secrets)
						require.NoError(t, db.Model(&secrets[0]).Update("value", []byte("rotated-token")).Error)
					} else {
						config := integration.Configuration.Data()
						config["providerValidationRevision"] = "changed"
						require.NoError(t, db.Model(integration).Update("configuration", datatypes.NewJSONType(config)).Error)
					}
				}
				return []string{"next"}, nil
			}})
			_, err := f.apply(bundle, plan.BaseStateToken)
			require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err), "%v", err)
			require.True(t, changed)
			saved, err := models.FindFactory(db, f.factory.OrganizationID, f.factory.ID)
			require.NoError(t, err)
			require.Equal(t, "main", saved.OnboardingConfigValue().DefaultBranch)
		})
	}
}

func TestSkillUpdateDoesNotLockRepositoryBindings(t *testing.T) {
	f := organizationKeyConfigurationFixture(t)
	enableWorkspaceSkills(t, f.r.Organization.ID)
	db := database.DB(t.Context())
	skill, err := f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "notes", true, models.FactoryAgentResourceConfig{Source: models.FactoryAgentResourceSourceInline, Markdown: "Use the runbook."})
	require.NoError(t, err)
	locked := db.Begin()
	defer locked.Rollback()
	require.NoError(t, models.LockOrganizationVCSProviderBindingRepositories(locked, f.factory.OrganizationID))
	ctx, cancel := context.WithTimeout(f.ctx, time.Second)
	defer cancel()
	name := "runbook"
	response, err := UpdateFactoryAgentResource(ctx, IntakeDependencies{}, f.factory.OrganizationID.String(), &pb.UpdateFactoryAgentResourceRequest{FactoryId: f.factory.ID.String(), ResourceId: skill.ID.String(), Name: &name})
	require.NoError(t, err)
	require.Equal(t, name, response.Resource.Name)
}

type repositoryValidationHTTP func(*http.Request) (*http.Response, error)

func (f repositoryValidationHTTP) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestBitbucketContextRepositoryValidationReleasesWorkspace(t *testing.T) {
	for _, operation := range []string{"create", "enable"} {
		t.Run(operation, func(t *testing.T) {
			r := support.Setup(t)
			factory := bitbucketFactory(t, r)
			db := database.DB(t.Context())
			integrationID := uuid.NewString()
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID}))
			original := newFactoryBitbucketAPI
			defer func() { newFactoryBitbucketAPI = original }()
			calls := 0
			newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucketintegration.Client, error) {
				return &bitbucketintegration.Client{AuthType: bitbucketintegration.AuthTypeWorkspaceAccessToken, Token: "token", HTTP: repositoryValidationHTTP(func(*http.Request) (*http.Response, error) {
					calls++
					ctx, cancel := context.WithTimeout(t.Context(), time.Second)
					defer cancel()
					require.NoError(t, database.DB(ctx).Transaction(func(tx *gorm.DB) error { current := *factory; return current.LockForUpdate(tx) }), "provider HTTP must release workspace locks")
					return okBitbucketResponse(`{"full_name":"acme/docs"}`), nil
				})}, nil
			}
			if operation == "create" {
				_, err := CreateFactoryAgentResource(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAgentResourceRequest{FactoryId: factory.ID.String(), Kind: pb.FactoryAgentResource_KIND_CONTEXT_REPO, Repository: "acme/docs", VcsProvider: models.ProviderBitbucket, Access: "write", Enabled: true})
				require.NoError(t, err)
			} else {
				resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", false, models.FactoryAgentResourceConfig{Repository: "acme/docs", VCSProvider: models.ProviderBitbucket, Access: "write"})
				require.NoError(t, err)
				enabled := true
				_, err = UpdateFactoryAgentResource(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryAgentResourceRequest{FactoryId: factory.ID.String(), ResourceId: resource.ID.String(), Enabled: &enabled})
				require.NoError(t, err)
			}
			require.Equal(t, 1, calls, "the mutation must consume the provider receipt")
		})
	}
}
