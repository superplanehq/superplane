package factories

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

// Repository settings are exported for reference; applying automations must
// never update them, even when the manifest includes equivalent defaults.
func TestFactoryConfigurationRepositoryRoundTrip(t *testing.T) {
	for _, absent := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchanged", true: "absent"}[absent], func(t *testing.T) {
			f := newConfigurationFixture(t)
			db := database.DB(t.Context())
			for _, repository := range []struct {
				name, access string
				enabled      bool
			}{
				{"acme/docs", "read", true}, {"acme/tools", "write", true}, {"acme/disabled", "read", false},
			} {
				_, err := f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", repository.enabled, models.FactoryAgentResourceConfig{Repository: repository.name, Ref: "release", Access: repository.access})
				require.NoError(t, err)
			}
			before, err := f.factory.ListContextRepos(db)
			require.NoError(t, err)
			config := f.factory.OnboardingConfigValue()
			bundle := f.export(t)
			manifest, diagnostic := parseFactoryConfigurationBundle(bundle)
			require.Nil(t, diagnostic)
			require.Equal(t, config.AppRepository, manifest.Repositories.Primary.Repository)
			require.Equal(t, config.DefaultBranch, manifest.Repositories.Primary.Ref)
			require.Equal(t, config.BacklogRepository, manifest.Repositories.BacklogRepository)
			require.ElementsMatch(t, []factoryConfigurationContextRepository{
				{Repository: "acme/docs", Ref: "release", Access: "read", Enabled: boolPointer(true), Provider: models.ProviderGitHub},
				{Repository: "acme/tools", Ref: "release", Access: "write", Enabled: boolPointer(true), Provider: models.ProviderGitHub},
				{Repository: "acme/disabled", Ref: "release", Access: "read", Enabled: boolPointer(false), Provider: models.ProviderGitHub},
			}, manifest.Repositories.Additional)
			editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
				if absent {
					manifest.Repositories = nil
				} else {
					slices.Reverse(manifest.Repositories.Additional)
					manifest.Repositories.Binding = factoryConfigurationBinding{IntegrationID: config.VCSIntegrationID}
					manifest.Repositories.Primary.Repository = strings.ToUpper(config.AppRepository)
					for i := range manifest.Repositories.Additional {
						entry := &manifest.Repositories.Additional[i]
						entry.Provider = ""
						if entry.enabled() {
							entry.Enabled = nil
						}
					}
				}
				manifest.Stages[1].Name = "Updated implementation"
			})
			plan := f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			require.Len(t, plan.Changes, 1)
			checkWrite := func(tx *gorm.DB) {
				if tx.Statement.Table == "factories" || tx.Statement.Table == "factory_agent_resources" || tx.Statement.Table == "vcs_provider_integration_repositories" {
					t.Errorf("apply wrote repository table %s", tx.Statement.Table)
				}
			}
			require.NoError(t, db.Callback().Create().Before("gorm:create").Register("reject_repository_write", checkWrite))
			require.NoError(t, db.Callback().Update().Before("gorm:update").Register("reject_repository_write", checkWrite))
			require.NoError(t, db.Callback().Delete().Before("gorm:delete").Register("reject_repository_write", checkWrite))
			defer db.Callback().Create().Remove("reject_repository_write")
			defer db.Callback().Update().Remove("reject_repository_write")
			defer db.Callback().Delete().Remove("reject_repository_write")
			result, err := f.apply(bundle, plan.BaseStateToken)
			require.NoError(t, err)
			require.True(t, result.Changed)
			after, err := f.factory.ListContextRepos(db)
			require.NoError(t, err)
			require.Equal(t, before, after)
			factory, err := models.FindFactory(db, f.r.Organization.ID, f.factory.ID)
			require.NoError(t, err)
			require.Equal(t, config, factory.OnboardingConfigValue())
		})
	}
}

func TestFactoryConfigurationRejectsRepositoryEditsBeforePublication(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	_, err := f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{Repository: "acme/docs", Ref: "main", Access: "read"})
	require.NoError(t, err)
	for _, test := range []struct {
		name string
		edit func(*factoryConfigurationRepositories)
	}{
		{"primary", func(r *factoryConfigurationRepositories) { r.Primary.Repository = "acme/other" }},
		{"branch", func(r *factoryConfigurationRepositories) { r.Primary.Ref = "next" }},
		{"omitted branch", func(r *factoryConfigurationRepositories) { r.Primary.Ref = "" }},
		{"backlog", func(r *factoryConfigurationRepositories) { r.BacklogRepository = "acme/other" }},
		{"connection", func(r *factoryConfigurationRepositories) {
			r.Binding = factoryConfigurationBinding{IntegrationID: createReadyOnboardingIntegration(t, f.r.Organization.ID, models.ProviderGitHub)}
		}},
		{"addition", func(r *factoryConfigurationRepositories) {
			r.Additional = append(r.Additional, factoryConfigurationContextRepository{Repository: "acme/new", Access: "read"})
		}},
		{"removal", func(r *factoryConfigurationRepositories) { r.Additional = nil }},
		{"additional branch", func(r *factoryConfigurationRepositories) { r.Additional[0].Ref = "next" }},
		{"access", func(r *factoryConfigurationRepositories) { r.Additional[0].Access = "write" }},
		{"enabled", func(r *factoryConfigurationRepositories) { r.Additional[0].Enabled = boolPointer(false) }},
		{"provider", func(r *factoryConfigurationRepositories) { r.Additional[0].Provider = models.ProviderBitbucket }},
	} {
		t.Run(test.name, func(t *testing.T) {
			bundle := f.export(t)
			editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) {
				test.edit(m.Repositories)
				m.Stages[0].Automations = append(m.Stages[0].Automations, factoryConfigurationAutomation{ID: "issues", Name: "Rejected intake", Uses: "@superplane/intake/github-issues"})
			})
			before := configurationResourceCounts(t)
			revisions := configurationRevisionCount(t, f.canvas)
			plan := f.plan(t, bundle)
			require.Len(t, plan.Diagnostics, 1)
			require.Contains(t, plan.Diagnostics[0].Field, "repositories")
			require.Contains(t, plan.Diagnostics[0].Message, "workspace settings")
			require.Empty(t, plan.BaseStateToken)
			_, err := f.apply(bundle, "invalid")
			require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
			require.Equal(t, before, configurationResourceCounts(t))
			require.Equal(t, revisions, configurationRevisionCount(t, f.canvas))
		})
	}
}

func TestFactoryConfigurationRejectsRepositoryChangesAfterPlan(t *testing.T) {
	for _, absent := range []bool{false, true} {
		t.Run(map[bool]string{false: "included", true: "absent"}[absent], func(t *testing.T) {
			f := newConfigurationFixture(t)
			db := database.DB(t.Context())
			resource, err := f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{Repository: "acme/docs", Ref: "main", Access: "read"})
			require.NoError(t, err)
			bundle := f.export(t)
			editConfigurationManifest(t, bundle, func(m *factoryConfigurationManifest) {
				if absent {
					m.Repositories = nil
				}
				m.Stages[1].Name = "Updated implementation"
			})
			plan := f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			updated := resource.Config.Data()
			updated.Ref = "saved-in-ui"
			require.NoError(t, resource.Update(db, nil, nil, &updated))
			before := configurationResourceCounts(t)
			_, err = f.apply(bundle, plan.BaseStateToken)
			if absent {
				require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
			} else {
				require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
			}
			require.Equal(t, before, configurationResourceCounts(t))
		})
	}
}

func TestFactoryConfigurationResolvesNamedIntakeConnection(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	name := integrationName(t, f.r.Organization.ID, f.factory.OnboardingConfigValue().VCSIntegrationID)
	bundle.Files[0].Content = strings.Replace(bundle.Files[0].Content, "name: Backlog\n    automations: []", `name: Backlog
    automations:
      - id: issues
        name: Named connection intake
        uses: "@superplane/intake/github-issues"
        binding:
          integration_name: "`+name+`"`, 1)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	plan = f.plan(t, f.export(t))
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationRemovesIntakeConnectionPin(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	previous := f.factory.OnboardingConfigValue()
	connection := previous.VCSIntegrationID
	bundle := f.export(t)
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		manifest.Stages[0].Automations = append(manifest.Stages[0].Automations, factoryConfigurationAutomation{
			ID: "issues", Name: "Unpinned issues", Uses: "@superplane/intake/github-issues",
			Binding: &factoryConfigurationBinding{IntegrationID: connection},
		})
	})
	applyConfigurationBundle(t, f, bundle)
	canvas := configurationCanvasByName(t, f, "Unpinned issues")
	bundle = f.export(t)
	configurationUpdateCatalogAutomation(t, bundle, "issues", func(automation *factoryConfigurationAutomation) {
		automation.Binding = nil
	})
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes, "removing a saved connection pin must change ownership")
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	owner, err := models.FindFactoryConfigurationAutomationByCanvas(db, canvas.ID)
	require.NoError(t, err)
	require.NotContains(t, owner.Binding, "integration_id")
	plan = f.plan(t, f.export(t))
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes, "exporting the selected connection must not introduce a pin")
	newConnection := createReadyOnboardingIntegration(t, f.factory.OrganizationID, models.ProviderGitHub)
	require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &newConnection}))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return reconcileFactoryRepository(f.ctx, tx, testIntakeDependencies(f.r), f.factory, f.r.User,
			connection, newConnection, previous.AppRepository, previous.BacklogRepository, previous.DefaultBranch, previous.AppRepository, false)
	}))
	trigger := findIntakeNode(resetTestLiveVersion(t, canvas).Nodes, intakeTriggerNodeID)
	require.NotNil(t, trigger)
	require.Equal(t, newConnection, *trigger.IntegrationID, "an unpinned intake must follow the workspace connection")
}

func boolPointer(value bool) *bool { return &value }

func TestFactoryConfigurationRejectsPerAutomationRepositorySelection(t *testing.T) {
	for _, templateID := range []string{lineImplementationTemplateID, "risk-score", scheduledIntakeTemplateID, "pr-closure"} {
		t.Run(templateID, func(t *testing.T) {
			f := configurationCatalogFixture(t)
			automation := configurationCatalogEntry(t, f, templateID)
			automation.Binding = &factoryConfigurationBinding{ResourceID: "acme/backlog"}
			bundle := configurationCatalogBundle(t, f, automation, templateID)
			before := configurationResourceCounts(t)
			plan := f.plan(t, bundle)
			require.Len(t, plan.Diagnostics, 1)
			require.Contains(t, plan.Diagnostics[0].Field, ".binding.resource_id")
			require.Empty(t, plan.BaseStateToken)
			_, err := f.apply(bundle, "invalid")
			require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
			require.Equal(t, before, configurationResourceCounts(t))
		})
	}
}
