package factories

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	canvasyaml "github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

func TestFactoryConfigurationSharedSourcePreservesDestinationIdentity(t *testing.T) {
	f := newConfigurationLifecycleFixture(t)
	db := database.DB(t.Context())
	bundle := f.export(t)
	bundle.ManifestPath = "workspaces/first.yml"
	bundle.Files[0].Path = bundle.ManifestPath
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	// The authored ID can also be a stage ID because namespaces are separate.
	identity := manifest.Stages[2].ID
	manifest.Stages[2].Automations = append(manifest.Stages[2].Automations, factoryConfigurationAutomation{ID: identity, Uses: "../automations/review.yml"})
	manifest.Repositories = nil
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	source := "# Shared across workspaces.\napiVersion: v2\nkind: Automation\nname: Shared review\nrun_context: task\ntriggers:\n  - uses: manual\nsteps:\n  - name: Review\n    prompt: Review the task.\n"
	bundle.Files = append(bundle.Files, &pb.FactoryConfigurationFile{Path: "automations/review.yml", Content: source})
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	applied, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, applied.Changed)
	first, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Shared review")
	require.NoError(t, err)
	firstVersion := *first.LiveVersionID

	destination, err := models.CreateFactory(db, f.r.Organization.ID, "Destination workspace", "", "DEST")
	require.NoError(t, err)
	support.CompleteFactoryOnboarding(t, destination)
	destination.OnboardingConfig = f.factory.OnboardingConfig
	require.NoError(t, db.Model(destination).Update("onboarding_config", destination.OnboardingConfig).Error)
	runner := resetTestTemplate(t, f.r, destination, "line-implementation", models.ProviderGitHub, func(doc *canvasyaml.Canvas) {
		agent := findYAMLNode(t, doc, implementationAgentNodeID)
		agent.Configuration["additionalInstructions"] = "Original instructions"
		agent.Configuration["thinkingLevel"] = "low"
	})
	require.NoError(t, runner.SetConfigurationID(db, f.canvas.ConfigurationID))
	require.NoError(t, runner.StampFactoryAppTemplateFor(db, "onrun-implement", "line-implementation", 1, models.ProviderGitHub))
	line, err := destination.CreateLine(db, "Delivery", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: runner.ID, Entrypoint: "onrun-implement"}})
	require.NoError(t, err)
	stages := line.StagesValue()
	originalStages := f.line.StagesValue()
	require.Len(t, stages, len(originalStages))
	for index := range stages {
		stages[index].Name = originalStages[index].Name
		stages[index].ConfigurationID = originalStages[index].ConfigurationID
	}
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	other := configurationFixture{r: f.r, factory: destination, line: line, canvas: runner, ctx: f.ctx}
	copied := proto.Clone(bundle).(*pb.FactoryConfigurationBundle)
	copied.ManifestPath = "workspaces/second.yaml"
	copied.Files[0].Path = copied.ManifestPath
	plan = other.plan(t, copied)
	require.Empty(t, plan.Diagnostics)
	applied, err = other.apply(copied, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, applied.Changed)
	second, err := models.FindCanvasByName(db, f.r.Organization.ID, &destination.ID, "Shared review")
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)
	currentDestination, err := models.FindFactory(db, f.r.Organization.ID, destination.ID)
	require.NoError(t, err)
	require.Equal(t, f.factory.Name, currentDestination.Name)
	require.Equal(t, "DEST", currentDestination.Key)

	before := configurationResourceCounts(t)
	bundle.ManifestPath = "manifests/moved.yml"
	bundle.Files[0].Path = bundle.ManifestPath
	manifest.Stages[2].Automations[len(manifest.Stages[2].Automations)-1].Uses = "../shared/inspection.yaml"
	content, err = yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	bundle.Files[1].Path = "shared/inspection.yaml"
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	applied, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, applied.Changed)
	require.Equal(t, before, configurationResourceCounts(t))
	currentFirst, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, first.ID)
	require.NoError(t, err)
	require.Equal(t, firstVersion, *currentFirst.LiveVersionID)
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.ID.String()})
	require.NoError(t, err)
	parsed, diagnostic := parseFactoryConfigurationBundle(exported.Bundle)
	require.Nil(t, diagnostic)
	found := false
	for _, stage := range parsed.Stages {
		for _, automation := range stage.Automations {
			if automation.Name == "Shared review" {
				require.Equal(t, identity, automation.ID)
				found = true
			}
		}
	}
	require.True(t, found)
	plan = other.plan(t, copied)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationRejectsWorkspaceWideIDReuseWithoutMutation(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	original := manifest.Stages[1].Automations[0]
	manifest.Stages[2].Automations = append(manifest.Stages[2].Automations, factoryConfigurationAutomation{ID: original.ID, Name: "Another automation", Uses: "@superplane/merge-confidence"})
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	before := configurationResourceCounts(t)
	plan := f.plan(t, bundle)
	require.NotEmpty(t, plan.Diagnostics)
	require.Equal(t, "stages[2].automations[0].id", plan.Diagnostics[0].Field)
	_, err = f.apply(bundle, "invalid")
	require.Error(t, err)
	require.Equal(t, before, configurationResourceCounts(t))
}
