package factories

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/proto"
	goyaml "gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// Installation is owned by the plan/apply boundary: graph-only tests cannot
// detect missing subscriptions, ownership, skipped imports, or partial commits.
func TestFactoryConfigurationInstallsMissingBuiltIns(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	bundle.Files[0].Content = strings.Replace(bundle.Files[0].Content, "name: Backlog\n    automations: []", `name: Backlog
    automations:
      - id: issues
        name: Issues from YAML
        uses: "@superplane/intake/github-issues"`, 1)
	before := configurationResourceCounts(t)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Equal(t, before, configurationResourceCounts(t), "plan must be read-only")
	require.NotEmpty(t, plan.Changes)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	db := database.DB(t.Context())
	intakes, err := f.factory.ListIntakes(db)
	require.NoError(t, err)
	require.Len(t, intakes, 1)
	require.Equal(t, models.FactoryIntakeInitialImportStatusSkipped, intakes[0].InitialImportStatus)
	installed := intakes[0].Canvas
	live := resetTestLiveVersion(t, installed)
	require.NotEmpty(t, live.Nodes)
	for _, node := range live.Nodes {
		require.Nil(t, node.ErrorMessage, node.ID)
	}
	owners, err := models.ListFactoryConfigurationAutomations(db, f.factory.OrganizationID, f.factory.ID, f.line.ID)
	require.NoError(t, err)
	require.Len(t, owners, 1)
	require.Equal(t, "issues", owners[0].ManifestID)
	require.Equal(t, installed.ID, owners[0].CanvasID)
	resources := configurationResourceCounts(t)
	revision := configurationRevisionCount(t, installed)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	result, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, resources, configurationResourceCounts(t))
	require.Equal(t, revision, configurationRevisionCount(t, installed))

	exported := f.export(t)
	require.Contains(t, exported.Files[0].Content, "id: issues")
	exportedIntake := configurationManifestAutomation(t, exported, "backlog", "issues")
	require.Equal(t, &factoryConfigurationBinding{IntegrationName: integrationName(t, f.r.Organization.ID, f.factory.OnboardingConfigValue().VCSIntegrationID)}, exportedIntake.Binding)
	renamed := proto.Clone(exported).(*pb.FactoryConfigurationBundle)
	renamed.Files[0].Content = strings.Replace(renamed.Files[0].Content, "Issues from YAML", "Renamed issues", 1)
	plan = f.plan(t, renamed)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(renamed, plan.BaseStateToken)
	require.NoError(t, err)
	require.Equal(t, revision, configurationRevisionCount(t, installed), "rename must not publish a revision")
	require.Contains(t, f.export(t).Files[0].Content, "id: issues")

	// A deleted installation must receive a new canvas and intake record.
	require.NoError(t, installed.SoftDeleteInTransaction(db))
	plan = f.plan(t, exported)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(exported, plan.BaseStateToken)
	require.NoError(t, err)
	intakes, err = f.factory.ListIntakes(db)
	require.NoError(t, err)
	require.Len(t, intakes, 1)
	require.NotEqual(t, installed.ID, intakes[0].CanvasID)
	require.NotEqual(t, owners[0].CanvasID, intakes[0].CanvasID)
	require.Equal(t, exported.Files[0].Content, f.export(t).Files[0].Content)
}

func TestFactoryConfigurationInstallationRollsBackAfterPartialProgress(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	bundle.Files[0].Content = strings.Replace(bundle.Files[0].Content, "name: Backlog\n    automations: []", `name: Backlog
    automations:
      - id: first
        name: First intake
        uses: "@superplane/intake/github-issues"
      - id: second
        name: Second intake
        uses: "@superplane/intake/github-issues"`, 1)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	before := configurationResourceCounts(t)
	db := database.DB(t.Context())
	created := 0
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("configuration_intake_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "factory_intakes" {
			created++
			if created == 2 {
				tx.AddError(errors.New("second intake failed"))
			}
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Create().Remove("configuration_intake_failure") })
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.Error(t, err)
	require.Equal(t, 2, created, "failure must happen after the first installation")
	require.Equal(t, before, configurationResourceCounts(t))
}

func TestFactoryConfigurationRestoresFiveDeletedBuiltIns(t *testing.T) {
	f := configurationCatalogFixture(t)
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	entries := []struct{ stage, id, name, uses string }{
		{"backlog", "analysis", "Backlog", "@superplane/task-analysis"},
		{"done", "closure", "PR Closure", "@superplane/pr-closure"},
		{"backlog", "issues", "GitHub issues", "@superplane/intake/github-issues"},
		{"backlog", "schedule", "Scheduled intake", "@superplane/scheduled-intake"},
		{"verify", "confidence", "Merge confidence", "@superplane/merge-confidence"},
		{"verify", "discussion", "PR discussion", "@superplane/pr-discussion"},
		{"verify", "checks", "PR checks", "@superplane/pr-checks"},
	}
	for _, entry := range entries {
		for i := range manifest.Stages {
			if manifest.Stages[i].Key != entry.stage {
				continue
			}
			automation := factoryConfigurationAutomation{ID: entry.id, Name: entry.name, Uses: entry.uses}
			if entry.id == "checks" {
				automation.With = map[string]any{"pr_feedback": map[string]any{"check_names": []any{"CI"}}}
			}
			manifest.Stages[i].Automations = append(manifest.Stages[i].Automations, automation)
		}
	}
	content, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	exported := f.export(t)
	db := database.DB(t.Context())
	canvases, err := f.factory.ListCanvases(db)
	require.NoError(t, err)
	require.Len(t, canvases, 8)
	deleted := map[string]uuid.UUID{}
	retained := map[uuid.UUID]uuid.UUID{}
	for i := range canvases {
		canvas := &canvases[i]
		if canvas.ID == f.canvas.ID || canvas.Name == "Backlog" || canvas.Name == "PR Closure" {
			retained[canvas.ID] = *canvas.LiveVersionID
			continue
		}
		deleted[canvas.Name] = canvas.ID
		require.NoError(t, canvas.SoftDeleteInTransaction(db))
	}
	require.Len(t, deleted, 5)
	plan = f.plan(t, exported)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 5)
	_, err = f.apply(exported, plan.BaseStateToken)
	require.NoError(t, err)
	canvases, err = f.factory.ListCanvases(db)
	require.NoError(t, err)
	require.Len(t, canvases, 8)
	for _, canvas := range canvases {
		if previous, exists := retained[canvas.ID]; exists {
			require.Equal(t, previous, *canvas.LiveVersionID)
			continue
		}
		require.NotEqual(t, deleted[canvas.Name], canvas.ID)
	}
	require.Equal(t, exported.Files[0].Content, f.export(t).Files[0].Content)
}
