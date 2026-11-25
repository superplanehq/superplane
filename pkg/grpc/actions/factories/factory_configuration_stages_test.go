package factories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"google.golang.org/grpc/codes"
	yamlv3 "gopkg.in/yaml.v3"
)

func TestFactoryConfigurationIncludesEveryStageAndPhase(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(f.r.Organization.ID, features.FeatureFactoryCustomAutomations))
	second := resetTestTemplate(t, f.r, f.factory, "line-implementation", models.ProviderGitHub, func(*yaml.Canvas) {})
	steps := append([]models.FactoryLineStep{}, f.line.Steps...)
	steps = append(steps, models.FactoryLineStep{Type: models.FactoryLineStepTypeRunApp, AppID: second.ID, Entrypoint: "onrun-implement"})
	require.NoError(t, f.line.Update(db, nil, steps, nil))
	stages := f.line.StagesValue()
	customID := uuid.New()
	custom := models.FactoryLineStage{ID: customID, Key: "custom-" + customID.String(), Name: "Security", Kind: models.FactoryLineStagePhase}
	byKey := map[string]models.FactoryLineStage{}
	for _, stage := range stages {
		byKey[stage.Key] = stage
	}
	stages = []models.FactoryLineStage{byKey["backlog"], byKey["phase-0"], custom, byKey["phase-1"], byKey["verify"], byKey["done"]}
	require.NoError(t, f.line.UpdateConfiguration(db, nil, nil, nil, stages))
	closure := resetTestTemplate(t, f.r, f.factory, "pr-closure", models.ProviderGitHub, func(*yaml.Canvas) {})
	before := configurationResourceCounts(t)
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yamlv3.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	require.Len(t, manifest.Stages, 6)
	// Stage kinds are live state, not exported configuration.
	var document map[string]any
	require.NoError(t, yamlv3.Unmarshal([]byte(bundle.Files[0].Content), &document))
	require.Equal(t, "Factory", document["kind"])
	for _, stage := range document["stages"].([]any) {
		require.NotContains(t, stage.(map[string]any), "kind")
	}
	require.Empty(t, manifest.Stages[2].Automations)
	require.Empty(t, manifest.Stages[2].Run)
	require.Empty(t, manifest.Stages[4].Run)
	require.Len(t, manifest.Stages[5].Automations, 1)
	require.Equal(t, "@superplane/pr-closure", manifest.Stages[5].Automations[0].Uses)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, before, configurationResourceCounts(t))

	manifest.Stages[2].Name = "Phase"
	manifest.Stages[2], manifest.Stages[3] = manifest.Stages[3], manifest.Stages[2]
	maxParallelism := 3
	manifest.Stages[2].MaxParallelism = &maxParallelism
	encoded, err := yamlv3.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(encoded)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes)
	result, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	reloaded, err := f.factory.FindLine(db, f.line.ID)
	require.NoError(t, err)
	require.Equal(t, custom.ID, reloaded.Stages[3].ID)
	require.Equal(t, models.FactoryLineStagePhase, reloaded.Stages[3].Kind)
	require.Equal(t, "Phase", reloaded.Stages[3].Name)
	require.Equal(t, 3, reloaded.Steps[1].EffectiveMaxParallelism())
	require.Equal(t, before, configurationResourceCounts(t), "stage edits must not publish automation revisions")
	require.Equal(t, *closure.LiveVersionID, resetTestLiveVersion(t, closure).ID)
}

func TestFactoryConfigurationTracksEveryAutomationRevision(t *testing.T) {
	f := newConfigurationFixture(t)
	verify := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*yaml.Canvas) {})
	bundle := f.export(t)
	bundle.Files[0].Content = strings.Replace(bundle.Files[0].Content, "Original instructions", "Requested instructions", 1)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1)
	live := resetTestLiveVersion(t, verify)
	// A UI save on another stage must invalidate a plan for Implement.
	findIntakeNode(live.Nodes, "assess-risk").Configuration["workingDirectory"] = "src"
	require.NoError(t, canvases.PublishGeneratedCanvasNodes(f.ctx, database.DB(t.Context()), verify, f.r.User,
		"UI save", live.Nodes, live.Edges, intakePublisherOptions(testIntakeDependencies(f.r), f.r.Organization.ID)))
	before := configurationResourceCounts(t)
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Equal(t, before, configurationResourceCounts(t))
	require.Equal(t, "Original instructions", findIntakeNode(resetTestLiveVersion(t, f.canvas).Nodes, implementationAgentNodeID).Configuration["additionalInstructions"])
}

func TestFactoryConfigurationPreservesStageRestrictionsWithoutKinds(t *testing.T) {
	f := newConfigurationFixture(t)
	for _, tc := range []struct {
		name   string
		change func(*factoryConfigurationManifest)
		field  string
	}{
		{"added stage", func(m *factoryConfigurationManifest) {
			m.Stages = append(m.Stages, factoryConfigurationStage{Key: "new", Name: "New"})
		}, "stages"},
		{"removed stage", func(m *factoryConfigurationManifest) {
			m.Stages = m.Stages[1:]
		}, "stages"},
		{"unknown key", func(m *factoryConfigurationManifest) {
			m.Stages[1].Key = "unknown"
		}, "stages[1].key"},
		{"duplicate key", func(m *factoryConfigurationManifest) {
			m.Stages[1].Key = m.Stages[0].Key
		}, "stages[1].key"},
		{"backlog moved", func(m *factoryConfigurationManifest) {
			m.Stages[0], m.Stages[1] = m.Stages[1], m.Stages[0]
		}, "stages[0]"},
		{"done moved", func(m *factoryConfigurationManifest) {
			last := len(m.Stages) - 1
			m.Stages[last], m.Stages[last-1] = m.Stages[last-1], m.Stages[last]
		}, "stages"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle := f.export(t)
			var manifest factoryConfigurationManifest
			require.NoError(t, yamlv3.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
			tc.change(&manifest)
			encoded, err := yamlv3.Marshal(manifest)
			require.NoError(t, err)
			bundle.Files[0].Content = string(encoded)
			plan := f.plan(t, bundle)
			require.Len(t, plan.Diagnostics, 1)
			require.Equal(t, tc.field, plan.Diagnostics[0].Field)
			require.Empty(t, plan.BaseStateToken)
		})
	}
}
