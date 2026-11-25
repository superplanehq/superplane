package factories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	canvasyaml "github.com/superplanehq/superplane/pkg/yaml"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

func TestFactoryConfigurationRenamesAutomationWithoutReplacingIt(t *testing.T) {
	for _, tc := range []struct {
		name         string
		custom       bool
		renameSource bool
	}{
		{name: "built-in phase runner"},
		{name: "custom manifest-only", custom: true},
		{name: "custom matching source", custom: true, renameSource: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var f configurationFixture
			var canvas *models.Canvas
			if tc.custom {
				f = newConfigurationLifecycleFixture(t)
				bundle := f.export(t)
				addConfigurationLifecycleAutomation(t, bundle, "original-audit", "Original audit")
				plan := f.plan(t, bundle)
				require.Empty(t, plan.Diagnostics)
				_, err := f.apply(bundle, plan.BaseStateToken)
				require.NoError(t, err)
				canvas, err = models.FindCanvasByName(database.DB(t.Context()), f.r.Organization.ID, &f.factory.ID, "Original audit")
				require.NoError(t, err)
			} else {
				f = newConfigurationFixture(t)
				canvas = f.canvas
			}
			db := database.DB(t.Context())
			beforeVersion := resetTestLiveVersion(t, canvas)
			beforeLine, err := f.factory.FindLine(db, f.line.ID)
			require.NoError(t, err)
			before := configurationResourceCounts(t)
			exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{
				FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(),
			})
			require.NoError(t, err)
			var manifest factoryConfigurationManifest
			require.NoError(t, yaml.Unmarshal([]byte(exported.Bundle.Files[0].Content), &manifest))
			const desiredName = "Renamed automation"
			var identity string
			for i := range manifest.Stages {
				for j := range manifest.Stages[i].Automations {
					automation := &manifest.Stages[i].Automations[j]
					if automation.Name != canvas.Name {
						continue
					}
					identity = automation.ID
					automation.Name = desiredName
				}
			}
			require.NotEmpty(t, identity)
			content, err := yaml.Marshal(manifest)
			require.NoError(t, err)
			exported.Bundle.Files[0].Content = string(content)
			if tc.renameSource {
				exported.Bundle.Files[1].Content = strings.Replace(exported.Bundle.Files[1].Content, "name: Original audit", "name: "+desiredName, 1)
			}
			plan := f.plan(t, exported.Bundle)
			require.Empty(t, plan.Diagnostics)
			require.Len(t, plan.Changes, 1)
			require.True(t, strings.HasSuffix(plan.Changes[0].FieldPath, ".name"))
			require.Equal(t, canvas.Name, plan.Changes[0].PreviousValue)
			require.Equal(t, desiredName, plan.Changes[0].RequestedValue)
			applied, err := f.apply(exported.Bundle, plan.BaseStateToken)
			require.NoError(t, err)
			require.True(t, applied.Changed)
			current, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, canvas.ID)
			require.NoError(t, err)
			require.Equal(t, desiredName, current.Name)
			require.Equal(t, beforeVersion.ID, *current.LiveVersionID, "a name change must not publish a new graph")
			require.Equal(t, before, configurationResourceCounts(t))
			afterLine, err := f.factory.FindLine(db, f.line.ID)
			require.NoError(t, err)
			require.Equal(t, beforeLine, afterLine, "renaming must preserve phase runner and stage bindings")
			if tc.custom {
				source, err := models.FindAutomationSourceByArtifact(db, f.r.Organization.ID, canvas.ID.String())
				require.NoError(t, err)
				require.Equal(t, desiredName, source.Name)
				document, err := automationspec.Parse(source.Source)
				require.NoError(t, err)
				require.Equal(t, desiredName, document.Name)
				require.Equal(t, beforeVersion.ID.String(), source.ArtifactRevision)
			}
			plan = f.plan(t, exported.Bundle)
			require.Empty(t, plan.Diagnostics)
			require.Empty(t, plan.Changes)
			applied, err = f.apply(exported.Bundle, plan.BaseStateToken)
			require.NoError(t, err)
			require.False(t, applied.Changed)
			require.Equal(t, before, configurationResourceCounts(t))
			reexported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{
				FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(),
			})
			require.NoError(t, err)
			require.NoError(t, yaml.Unmarshal([]byte(reexported.Bundle.Files[0].Content), &manifest))
			found := false
			for _, stage := range manifest.Stages {
				for _, automation := range stage.Automations {
					if automation.ID == identity {
						found = true
						require.Equal(t, desiredName, automation.Name)
					}
				}
			}
			require.True(t, found)
			plan = f.plan(t, reexported.Bundle)
			require.Empty(t, plan.Diagnostics)
			require.Empty(t, plan.Changes)
			if !tc.custom {
				require.NoError(t, resetTestOrganization(t, f.r))
				current, err = models.FindCanvasInTransaction(db, f.r.Organization.ID, canvas.ID)
				require.NoError(t, err)
				require.Equal(t, desiredName, current.Name, "admin reset must preserve the automation name")
			}
		})
	}
}

func TestFactoryConfigurationRejectsRenameToAnotherStageAutomationName(t *testing.T) {
	f := newConfigurationFixture(t)
	other := resetTestTemplate(t, f.r, f.factory, "pr-closure", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	manifest.Stages[1].Automations[0].Name = other.Name
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	before := configurationResourceCounts(t)
	plan := f.plan(t, bundle)
	require.NotEmpty(t, plan.Diagnostics)
	require.Equal(t, "stages[1].automations[0].name", plan.Diagnostics[0].Field)
	require.Contains(t, strings.ToLower(plan.Diagnostics[0].Message), "unique")
	require.Empty(t, plan.BaseStateToken)
	_, err = f.apply(bundle, "invalid")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Equal(t, before, configurationResourceCounts(t))
	current, err := models.FindCanvasInTransaction(database.DB(t.Context()), f.r.Organization.ID, f.canvas.ID)
	require.NoError(t, err)
	require.Equal(t, f.canvas.Name, current.Name)
}

func TestFactoryConfigurationRenameRetainsDuplicateBuiltInIdentity(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	first := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
	second := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
	firstName, secondName := "Alpha confidence", "Beta confidence"
	require.NoError(t, canvases.UpdateCanvasInTransaction(db, first, &firstName, nil))
	require.NoError(t, canvases.UpdateCanvasInTransaction(db, second, &secondName, nil))
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	require.Len(t, manifest.Stages[2].Automations, 2)
	require.Equal(t, firstName, manifest.Stages[2].Automations[0].Name)
	const desiredName = "Zulu confidence"
	manifest.Stages[2].Automations[0].Name = desiredName
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	for _, expected := range []struct {
		canvas *models.Canvas
		name   string
	}{{first, desiredName}, {second, secondName}} {
		current, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, expected.canvas.ID)
		require.NoError(t, err)
		require.Equal(t, expected.name, current.Name)
	}
	before := configurationResourceCounts(t)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes, "renumbered aliases must still identify the original automations")
	applied, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, applied.Changed)
	require.Equal(t, before, configurationResourceCounts(t))
}

func TestFactoryConfigurationRejectsConflictingSharedAutomationNames(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	id := uuid.New()
	shared := models.FactoryLineStage{ID: id, Key: "custom-" + id.String(), Name: "Recheck", Kind: models.FactoryLineStagePhase}
	stages := f.line.StagesValue()
	done := stages[len(stages)-1]
	stages = append(stages[:len(stages)-1], shared, done)
	require.NoError(t, f.line.UpdateConfiguration(db, nil, nil, nil, stages))
	require.NoError(t, f.canvas.SetColumnKey(db, shared.Key))
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.ID.String()})
	require.NoError(t, err)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(exported.Bundle.Files[0].Content), &manifest))
	manifest.Stages[1].Automations[0].Name = "Renamed phase runner"
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	exported.Bundle.Files[0].Content = string(content)
	before := configurationResourceCounts(t)
	plan := f.plan(t, exported.Bundle)
	require.NotEmpty(t, plan.Diagnostics)
	require.Contains(t, strings.ToLower(plan.Diagnostics[0].Message), "shared")
	require.Empty(t, plan.BaseStateToken)
	_, err = f.apply(exported.Bundle, "invalid")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Equal(t, before, configurationResourceCounts(t))
}

func TestFactoryConfigurationRenamesAndUpdatesAutomationTogether(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	automation := &manifest.Stages[1].Automations[0]
	const desiredName = "Build the task"
	const instructions = "Follow the updated repository instructions."
	automation.Name = desiredName
	automation.With["additional_instructions"] = instructions
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	before := configurationRevisionCount(t, f.canvas)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 2)
	applied, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, applied.Changed)
	current, err := models.FindCanvasInTransaction(database.DB(t.Context()), f.r.Organization.ID, f.canvas.ID)
	require.NoError(t, err)
	require.Equal(t, desiredName, current.Name)
	require.Equal(t, before+1, configurationRevisionCount(t, current), "publish the setting change once")
	live := resetTestLiveVersion(t, current)
	require.Equal(t, instructions, findIntakeNode(live.Nodes, implementationAgentNodeID).Configuration["additionalInstructions"])
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationRenamesNewAutomationBeforeExport(t *testing.T) {
	f := newConfigurationLifecycleFixture(t)
	db := database.DB(t.Context())
	bundle := f.export(t)
	addConfigurationLifecycleAutomation(t, bundle, "friendly-audit", "Original audit")
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	created, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Original audit")
	require.NoError(t, err)
	before := configurationResourceCounts(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	const desiredName = "Renamed audit"
	manifest.Stages[2].Automations[0].Name = desiredName
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1, "the authored ID must identify the existing automation after a rename")
	require.True(t, strings.HasSuffix(plan.Changes[0].FieldPath, ".name"))
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	current, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, created.ID)
	require.NoError(t, err)
	require.Equal(t, desiredName, current.Name)
	require.Equal(t, created.LiveVersionID, current.LiveVersionID)
	require.Equal(t, before, configurationResourceCounts(t))
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)

	removed := proto.Clone(bundle).(*pb.FactoryConfigurationBundle)
	manifest.Stages[2].Automations = []factoryConfigurationAutomation{}
	content, err = yaml.Marshal(manifest)
	require.NoError(t, err)
	removed.Files[0].Content = string(content)
	removed.Files = removed.Files[:1]
	plan = f.plan(t, removed)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(removed, plan.BaseStateToken)
	require.NoError(t, err)
	_, err = models.FindCanvasInTransaction(db, f.r.Organization.ID, created.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = models.FindAutomationSourceByArtifact(db, f.r.Organization.ID, created.ID.String())
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	replacement, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, desiredName)
	require.NoError(t, err)
	require.NotEqual(t, created.ID, replacement.ID, "reusing the authored ID must create a new active automation")
	source, err := models.FindAutomationSourceByArtifact(db, f.r.Organization.ID, replacement.ID.String())
	require.NoError(t, err)
	require.Equal(t, desiredName, source.Name)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationRejectsSwappingDuplicateBuiltInNames(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	first := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(doc *canvasyaml.Canvas) {
		findYAMLNode(t, doc, "assess-risk").Configuration["additionalInstructions"] = "Alpha policy"
	})
	second := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(doc *canvasyaml.Canvas) {
		findYAMLNode(t, doc, "assess-risk").Configuration["additionalInstructions"] = "Beta policy"
	})
	firstName, secondName := "Alpha confidence", "Beta confidence"
	require.NoError(t, canvases.UpdateCanvasInTransaction(db, first, &firstName, nil))
	require.NoError(t, canvases.UpdateCanvasInTransaction(db, second, &secondName, nil))
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	require.Len(t, manifest.Stages[2].Automations, 2)
	manifest.Stages[2].Automations[0].Name = secondName
	manifest.Stages[2].Automations[1].Name = firstName
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	before := configurationResourceCounts(t)
	plan := f.plan(t, bundle)
	require.NotEmpty(t, plan.Diagnostics)
	require.Contains(t, strings.ToLower(plan.Diagnostics[0].Message), "unique")
	require.Empty(t, plan.BaseStateToken)
	_, err = f.apply(bundle, "invalid")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Equal(t, before, configurationResourceCounts(t))
	for _, expected := range []struct {
		canvas *models.Canvas
		name   string
		policy string
	}{{first, firstName, "Alpha policy"}, {second, secondName, "Beta policy"}} {
		current, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, expected.canvas.ID)
		require.NoError(t, err)
		require.Equal(t, expected.name, current.Name)
		require.Equal(t, expected.policy, findIntakeNode(resetTestLiveVersion(t, current).Nodes, "assess-risk").Configuration["additionalInstructions"])
	}
}
