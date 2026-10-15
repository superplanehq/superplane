package models_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestSystemSkillAutomaticUpdatePreservesEditsAndNewerVersions(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	create := func(name string) *models.FactoryAgentResource {
		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, name, true, models.FactoryAgentResourceConfig{Markdown: "# Original", SystemSkill: &models.FactorySystemSkill{TemplateID: "refine-task", Version: 1}})
		require.NoError(t, err)
		return resource
	}
	load := func(resource *models.FactoryAgentResource) *models.FactoryAgentResource {
		var stored models.FactoryAgentResource
		require.NoError(t, db.First(&stored, "id = ?", resource.ID).Error)
		return &stored
	}
	t.Run("advance baseline and skip unchanged or older templates", func(t *testing.T) {
		resource := create("refine")
		updated, err := resource.UpdateSystemSkillIfUnmodified(db, 2, "# Current")
		require.NoError(t, err)
		require.True(t, updated)
		stored := load(resource)
		require.EqualValues(t, 2, stored.Config.Data().SystemSkill.Version)
		require.Equal(t, "# Current", stored.Config.Data().Markdown)
		require.Equal(t, stored.Config.Data().SkillContentHash(), stored.Config.Data().SystemSkill.ContentHash)
		for _, version := range []int32{2, 1} {
			updated, err = stored.UpdateSystemSkillIfUnmodified(db, version, "# Current")
			require.NoError(t, err)
			require.False(t, updated)
			require.Equal(t, stored.UpdatedAt, load(resource).UpdatedAt)
		}
	})
	t.Run("preserve a saved customization", func(t *testing.T) {
		resource := create("custom")
		config := resource.Config.Data()
		config.Markdown = "# My edits"
		require.NoError(t, resource.Update(db, nil, nil, &config))
		updated, err := resource.UpdateSystemSkillIfUnmodified(db, 2, "# Current")
		require.NoError(t, err)
		require.False(t, updated)
		require.Equal(t, "# My edits", load(resource).Config.Data().Markdown)
	})
	t.Run("an edit after the scan wins", func(t *testing.T) {
		resource := create("concurrent")
		snapshot := *resource
		config := resource.Config.Data()
		config.Markdown = "# Edited while syncing"
		require.NoError(t, resource.Update(db, nil, nil, &config))
		updated, err := snapshot.UpdateSystemSkillIfUnmodified(db, 2, "# Current")
		require.NoError(t, err)
		require.False(t, updated)
		require.Equal(t, "# Edited while syncing", load(resource).Config.Data().Markdown)
	})
	t.Run("an older worker cannot replace a newer baseline", func(t *testing.T) {
		resource := create("rolling")
		oldSnapshot := *resource
		updated, err := resource.UpdateSystemSkillIfUnmodified(db, 3, "# Newest")
		require.NoError(t, err)
		require.True(t, updated)
		updated, err = oldSnapshot.UpdateSystemSkillIfUnmodified(db, 2, "# Older release")
		require.NoError(t, err)
		require.False(t, updated)
		newest := load(resource)
		updated, err = newest.UpdateSystemSkillIfUnmodified(db, 2, "# Older release")
		require.NoError(t, err)
		require.False(t, updated)
		require.Equal(t, "# Newest", load(resource).Config.Data().Markdown)
	})
}
