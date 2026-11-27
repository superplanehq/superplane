package factories

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func TestFactoryConfigurationPortableManifestAndWorkspaceName(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	limit := 20
	f.line.Steps[0].MaxParallelism = &limit
	require.NoError(t, f.line.UpdateConfiguration(db, nil, f.line.Steps, nil, nil))
	bundle := f.export(t)
	require.NotContains(t, bundle.Files[0].Content, "max_parallelism:")
	revisions := configurationRevisionCount(t, f.canvas)
	bundle.ManifestPath = "workspaces/delivery.yaml"
	bundle.Files[0].Path = bundle.ManifestPath
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)
	bundle.Files[0].Content = strings.Replace(bundle.Files[0].Content, "name: "+f.factory.Name+"\n", "name: Shared workspace name\n", 1)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1)
	require.Equal(t, "name", plan.Changes[0].FieldPath)
	result, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	stored, err := models.FindFactory(db, f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, "Shared workspace name", stored.Name)
	require.Equal(t, f.factory.Key, stored.Key)
	line, err := stored.FindLine(db, f.line.ID)
	require.NoError(t, err)
	require.Equal(t, 20, line.Steps[0].EffectiveMaxParallelism())
	require.Equal(t, revisions, configurationRevisionCount(t, f.canvas))
}

func TestFactoryConfigurationBundleNormalizesSharedReferences(t *testing.T) {
	source := "apiVersion: v2\nkind: Automation\nname: Review\ntriggers:\n  - uses: manual\nsteps:\n  - name: Review\n    bash: echo review\n"
	manifest := "apiVersion: v2\nkind: Factory\nname: Workspace\nstages:\n  - id: verify\n    name: Verify\n    automations:\n      - id: first\n        uses: ../automations/./review.yaml\n      - id: second\n        uses: ../automations/review.yaml\n"
	bundle := &pb.FactoryConfigurationBundle{ManifestPath: "workspaces/delivery.yaml", Files: []*pb.FactoryConfigurationFile{
		{Path: "workspaces/delivery.yaml", Content: manifest}, {Path: "automations/review.yaml", Content: source},
	}}
	parsed, diagnostic := parseFactoryConfigurationBundle(bundle)
	require.Nil(t, diagnostic)
	require.Len(t, parsed.Sources, 1)
	require.Equal(t, "Review", parsed.Stages[0].Automations[0].Name)
	bundle.Files = append(bundle.Files, &pb.FactoryConfigurationFile{Path: "automations/./review.yaml", Content: source})
	_, diagnostic = parseFactoryConfigurationBundle(bundle)
	require.NotNil(t, diagnostic)
	require.Equal(t, "workspaces/delivery.yaml", diagnostic.Path)
	require.Contains(t, diagnostic.Message, "more than once")
}

func TestFactoryConfigurationIDChangeReplacesOptionalAutomation(t *testing.T) {
	f := newConfigurationLifecycleFixture(t)
	bundle := f.export(t)
	addConfigurationLifecycleAutomation(t, bundle, "audit", "Audit")
	applyConfigurationBundle(t, f, bundle)
	original := configurationCanvasByName(t, f, "Audit")
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		manifest.Stages[2].Automations[0].ID = "audit-v2"
	})
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 2)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	replacement := configurationCanvasByName(t, f, "Audit")
	require.NotEqual(t, original.ID, replacement.ID)
	require.Equal(t, "audit-v2", replacement.ConfigurationID)
	_, err = models.FindCanvasInTransaction(database.DB(t.Context()), f.r.Organization.ID, original.ID)
	require.Error(t, err)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationReportsReferencedMainFilename(t *testing.T) {
	f := newConfigurationLifecycleFixture(t)
	for _, failure := range []string{"parse", "placement"} {
		t.Run(failure, func(t *testing.T) {
			bundle := f.export(t)
			bundle.ManifestPath = "workspaces/delivery.yml"
			bundle.Files[0].Path = bundle.ManifestPath
			editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
				manifest.Stages[2].Automations = append(manifest.Stages[2].Automations, factoryConfigurationAutomation{ID: "audit", Uses: "../.superplane/main.yml"})
			})
			trigger := "use: manual"
			field := "triggers[0].use"
			if failure == "placement" {
				trigger = "uses: task_enters_stage"
				field = "triggers[0].uses"
			}
			source := "apiVersion: v2\nkind: Automation\nname: Audit\ntriggers:\n  - " + trigger + "\nsteps:\n  - name: Inspect\n    prompt: Inspect the task.\n"
			bundle.Files = append(bundle.Files, &pb.FactoryConfigurationFile{Path: factoryConfigurationManifestPath, Content: source})
			before := configurationResourceCounts(t)
			plan := f.plan(t, bundle)
			require.Len(t, plan.Diagnostics, 1)
			require.Equal(t, factoryConfigurationManifestPath, plan.Diagnostics[0].Path)
			require.Equal(t, field, plan.Diagnostics[0].Field)
			require.Contains(t, plan.Diagnostics[0].Message, "line")
			_, err := f.apply(bundle, "invalid")
			require.Error(t, err)
			require.Equal(t, before, configurationResourceCounts(t))
		})
	}
}
