package workers

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestSystemSkillSyncAcrossOrganizations(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	org2, err := models.CreateOrganizationInTransaction(db, support.RandomName("org"), "")
	require.NoError(t, err)
	var unchanged []*models.FactoryAgentResource
	for _, org := range []*models.Organization{r.Organization, org2} {
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureWorkspaceSkills))
		factory, err := models.CreateFactory(db, org.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		// More than one scan page must update, across separate organizations.
		for i := 0; i < 51; i++ {
			resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, fmt.Sprintf("refine-%d", i), true, models.FactoryAgentResourceConfig{
				Markdown: "# Original supplied instructions", SystemSkill: &models.FactorySystemSkill{TemplateID: "refine-task", Version: 1},
			})
			require.NoError(t, err)
			unchanged = append(unchanged, resource)
		}
	}
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	custom, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "custom", true, models.FactoryAgentResourceConfig{Markdown: "# Supplied instructions", SystemSkill: &models.FactorySystemSkill{TemplateID: "refine-task", Version: 1}})
	require.NoError(t, err)
	edited := custom.Config.Data()
	edited.Markdown = "# User edits"
	require.NoError(t, custom.Update(db, nil, nil, &edited))
	unknown, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "unknown", true, models.FactoryAgentResourceConfig{Markdown: "# Unknown", SystemSkill: &models.FactorySystemSkill{TemplateID: "unknown", Version: 1}})
	require.NoError(t, err)
	ordinary, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "ordinary", true, models.FactoryAgentResourceConfig{Markdown: "# User skill"})
	require.NoError(t, err)
	worker := NewSystemSkillSyncWorker()
	count, err := worker.Process(t.Context(), db)
	require.NoError(t, err)
	require.Equal(t, 102, count)
	for _, resource := range unchanged {
		stored := &models.FactoryAgentResource{}
		err := db.First(stored, "id = ?", resource.ID).Error
		require.NoError(t, err)
		require.Equal(t, resource.Name, stored.Name)
		require.True(t, stored.Enabled)
		require.Contains(t, stored.Config.Data().Markdown, "You refine draft tasks")
		require.Contains(t, stored.Config.Data().Markdown, "name: "+resource.Name)
		require.Equal(t, stored.Config.Data().SkillContentHash(), stored.Config.Data().SystemSkill.ContentHash)
	}
	for _, resource := range []*models.FactoryAgentResource{custom, unknown, ordinary} {
		stored := &models.FactoryAgentResource{}
		err := db.First(stored, "id = ?", resource.ID).Error
		require.NoError(t, err)
		require.Equal(t, resource.Config, stored.Config)
		require.Equal(t, resource.UpdatedAt, stored.UpdatedAt)
	}
	count, err = worker.Process(t.Context(), db)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSystemSkillSyncSkipsInactiveWorkspaces(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	for _, tc := range []struct {
		name                                         string
		enabled, deletedFactory, deletedOrganization bool
	}{
		{"skills disabled", false, false, false},
		{"workspace deleted", true, true, false},
		{"organization deleted", true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org, err := models.CreateOrganizationInTransaction(db, support.RandomName("org"), "")
			require.NoError(t, err)
			if tc.enabled {
				require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureWorkspaceSkills))
			}
			factory, err := models.CreateFactory(db, org.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "refine", true, models.FactoryAgentResourceConfig{
				Markdown: "# Original", SystemSkill: &models.FactorySystemSkill{TemplateID: "refine-task", Version: 1},
			})
			require.NoError(t, err)
			if tc.deletedFactory {
				require.NoError(t, db.Model(factory).Update("deleted_at", time.Now()).Error)
			}
			if tc.deletedOrganization {
				require.NoError(t, db.Model(org).Update("deleted_at", time.Now()).Error)
			}
			count, err := NewSystemSkillSyncWorker().Process(t.Context(), db)
			require.NoError(t, err)
			require.Zero(t, count)
			var stored models.FactoryAgentResource
			require.NoError(t, db.First(&stored, "id = ?", resource.ID).Error)
			require.Equal(t, resource.Config, stored.Config)
		})
	}
}
