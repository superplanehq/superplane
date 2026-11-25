package factories

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type configurationFixture struct {
	r       *support.ResourceRegistry
	factory *models.Factory
	line    *models.FactoryLine
	canvas  *models.Canvas
	ctx     context.Context
}

func newConfigurationFixture(t *testing.T) configurationFixture {
	t.Helper()
	r, factory := resetTestFactory(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	support.CompleteFactoryOnboarding(t, factory)
	canvas := resetTestTemplate(t, r, factory, "line-implementation", models.ProviderGitHub, func(doc *yaml.Canvas) {
		agent := findYAMLNode(t, doc, implementationAgentNodeID)
		agent.Configuration["additionalInstructions"] = "Original instructions"
		agent.Configuration["thinkingLevel"] = "low"
		implementationStep(t, agent, "Implementation")["prompt"] = "Keep this installed managed prompt."
	})
	require.NoError(t, canvas.StampFactoryAppTemplateFor(database.DB(t.Context()), "onrun-implement", "line-implementation", 1, models.ProviderGitHub))
	_, err := models.UpsertCanvasVersionConsoleInTransaction(database.DB(t.Context()), canvas.ID,
		[]models.ConsolePanel{{ID: "instructions", Type: "markdown", Content: map[string]any{"text": "Preserve this console."}}},
		[]models.ConsoleLayoutItem{{I: "instructions", X: 1, Y: 2, W: 6, H: 4}},
	)
	require.NoError(t, err)
	line, err := factory.CreateLine(database.DB(t.Context()), "Delivery", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: canvas.ID, Entrypoint: "onrun-implement",
	}})
	require.NoError(t, err)
	// Selection uses the line binding and template marker, even after a rename.
	phaseName := "Build changes"
	stages := line.StagesValue()
	for i := range stages {
		if stages[i].Kind == models.FactoryLineStagePhase {
			stages[i].Name = phaseName
		}
	}
	require.NoError(t, line.UpdateConfiguration(database.DB(t.Context()), nil, nil, nil, stages))
	return configurationFixture{r: r, factory: factory, line: line, canvas: canvas, ctx: authentication.SetUserIdInMetadata(t.Context(), r.User.String())}
}

func (f configurationFixture) export(t *testing.T) *pb.FactoryConfigurationBundle {
	t.Helper()
	response, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{
		FactoryId: f.factory.Key, LineId: f.line.ID.String(),
	})
	require.NoError(t, err)
	require.Equal(t, factoryConfigurationCoverage, response.Coverage)
	require.Equal(t, ".superplane/main.yml", response.Bundle.ManifestPath)
	require.Len(t, response.Bundle.Files, 1)
	require.Contains(t, response.Bundle.Files[0].Content, "name: Build changes")
	require.NotContains(t, response.Bundle.Files[0].Content, f.canvas.ID.String())
	return response.Bundle
}

func (f configurationFixture) plan(t *testing.T, bundle *pb.FactoryConfigurationBundle) *pb.PlanFactoryConfigurationResponse {
	t.Helper()
	response, err := PlanFactoryConfiguration(f.ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.PlanFactoryConfigurationRequest{
		FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(), Bundle: bundle,
	})
	require.NoError(t, err)
	return response
}

func (f configurationFixture) apply(bundle *pb.FactoryConfigurationBundle, token string) (*pb.ApplyFactoryConfigurationResponse, error) {
	return ApplyFactoryConfiguration(f.ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.ApplyFactoryConfigurationRequest{
		FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(), Bundle: bundle, BaseStateToken: token,
	})
}

func configurationRevisionCount(t *testing.T, canvas *models.Canvas) int64 {
	t.Helper()
	var count int64
	require.NoError(t, database.DB(t.Context()).Model(&models.CanvasVersion{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
	return count
}

func configurationResourceCounts(t *testing.T) map[string]int64 {
	t.Helper()
	counts := map[string]int64{}
	for _, table := range []string{"factories", "factory_lines", "factory_intakes", "factory_pr_feedback_handlers", "automations_as_code", "workflows", "workflow_versions", "webhooks"} {
		var count int64
		require.NoError(t, database.DB(t.Context()).Table(table).Count(&count).Error)
		counts[table] = count
	}
	return counts
}

func TestFactoryConfigurationRoundTrip(t *testing.T) {
	f := newConfigurationFixture(t)
	unrelated := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*yaml.Canvas) {})
	unrelatedBefore := resetTestLiveVersion(t, unrelated)
	before := resetTestLiveVersion(t, f.canvas)
	beforeLine, err := f.factory.FindLine(database.DB(t.Context()), f.line.ID)
	require.NoError(t, err)
	beforeFactory, err := models.FindFactory(database.DB(t.Context()), f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	count := configurationRevisionCount(t, f.canvas)
	resources := configurationResourceCounts(t)
	bundle := f.export(t)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, count, configurationRevisionCount(t, f.canvas))
	require.Equal(t, resources, configurationResourceCounts(t))

	for _, instructions := range []string{"Follow AGENTS.md.", ""} {
		bundle.Files[0].Content = strings.ReplaceAll(bundle.Files[0].Content, "additional_instructions: Original instructions", "additional_instructions: \""+instructions+"\"")
		if instructions == "" {
			bundle.Files[0].Content = strings.ReplaceAll(bundle.Files[0].Content, "Follow AGENTS.md.", "")
		}
		plan = f.plan(t, bundle)
		require.Empty(t, plan.Diagnostics)
		require.Len(t, plan.Changes, 1)
		require.Equal(t, instructions, plan.Changes[0].RequestedValue)
		result, err = f.apply(bundle, plan.BaseStateToken)
		require.NoError(t, err)
		require.True(t, result.Changed)
		count++
		resources["workflow_versions"]++
		require.Equal(t, count, configurationRevisionCount(t, f.canvas))
		require.Equal(t, resources, configurationResourceCounts(t))
		live := resetTestLiveVersion(t, f.canvas)
		require.Equal(t, instructions, findIntakeNode(live.Nodes, implementationAgentNodeID).Configuration["additionalInstructions"])
		// The complete graph equality catches template rebuilds and unrelated changes.
		findIntakeNode(before.Nodes, implementationAgentNodeID).Configuration["additionalInstructions"] = instructions
		require.ElementsMatch(t, before.Nodes, live.Nodes)
		require.ElementsMatch(t, before.Edges, live.Edges)
		require.Equal(t, before.ConsolePanels, live.ConsolePanels)
		require.Equal(t, before.ConsoleLayout, live.ConsoleLayout)
		bundle.Files[0].Content = "# formatting only\n" + bundle.Files[0].Content
		plan = f.plan(t, bundle)
		require.Empty(t, plan.Changes)
		result, err = f.apply(bundle, plan.BaseStateToken)
		require.NoError(t, err)
		require.False(t, result.Changed)
		require.Equal(t, count, configurationRevisionCount(t, f.canvas))
	}
	afterLine, err := f.factory.FindLine(database.DB(t.Context()), f.line.ID)
	require.NoError(t, err)
	afterFactory, err := models.FindFactory(database.DB(t.Context()), f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, beforeLine, afterLine)
	require.Equal(t, beforeFactory, afterFactory)
	require.Equal(t, unrelatedBefore, resetTestLiveVersion(t, unrelated))
}

func TestFactoryConfigurationRejectsUnsupportedManifestWithoutWrites(t *testing.T) {
	f := newConfigurationFixture(t)
	exported := f.export(t)
	count := configurationRevisionCount(t, f.canvas)
	for _, tc := range []struct{ name, from, to, field string }{
		{"invalid YAML", "apiVersion: v2", "apiVersion: [", "$"},
		{"unknown field", "kind: Factory", "kind: Factory\nunknown: true", "unknown"},
		{"null instructions", "additional_instructions: Original instructions", "additional_instructions: null", "stages[1].automations[0].with.additional_instructions"},
		{"boolean instructions", "additional_instructions: Original instructions", "additional_instructions: true", "stages[1].automations[0].with.additional_instructions"},
		{"numeric instructions", "additional_instructions: Original instructions", "additional_instructions: 123", "stages[1].automations[0].with.additional_instructions"},
		{"YAML anchor", "additional_instructions: Original instructions", "additional_instructions: &instructions Original instructions", "stages[1].automations[0].with.additional_instructions"},
		{"unsupported alias", "@superplane/implement", "@superplane/planning", "stages[1].automations[0].uses"},
		{"zero concurrency", "run: implement", "run: implement\n    max_parallelism: 0", "stages[1].max_parallelism"},
		{"concurrency outside API range", "run: implement", "run: implement\n    max_parallelism: 2147483648", "stages[1].max_parallelism"},
		{"stage editing disabled", "name: Build changes", "name: Other phase", "stages"},
		{"uninstalled built-in", "      - id: implement", "      - id: second\n        name: Other\n        uses: \"@superplane/implement\"\n        with:\n          additional_instructions: Extra\n      - id: implement", "stages[1].automations[0].uses"},
		{"duplicate field", "kind: Factory", "kind: Factory\nkind: Factory", "kind"},
		{"extra document", "apiVersion: v2", "---\nname: extra\n---\napiVersion: v2", "$"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle := proto.Clone(exported).(*pb.FactoryConfigurationBundle)
			bundle.Files[0].Content = strings.Replace(bundle.Files[0].Content, tc.from, tc.to, 1)
			plan := f.plan(t, bundle)
			require.NotEmpty(t, plan.Diagnostics)
			require.Equal(t, bundle.ManifestPath, plan.Diagnostics[0].Path)
			require.Equal(t, tc.field, plan.Diagnostics[0].Field)
			require.Empty(t, plan.BaseStateToken)
			_, err := f.apply(bundle, "invalid")
			require.Error(t, err)
			require.Equal(t, count, configurationRevisionCount(t, f.canvas))
		})
	}
	for _, tc := range []struct {
		name   string
		change func(*pb.FactoryConfigurationBundle) *pb.FactoryConfigurationBundle
		field  string
	}{
		{"missing bundle", func(*pb.FactoryConfigurationBundle) *pb.FactoryConfigurationBundle { return nil }, "manifest_path"},
		{"unsafe manifest path", func(bundle *pb.FactoryConfigurationBundle) *pb.FactoryConfigurationBundle {
			bundle.ManifestPath = "../main.yml"
			return bundle
		}, "manifest_path"},
		{"unexpected file path", func(bundle *pb.FactoryConfigurationBundle) *pb.FactoryConfigurationBundle {
			bundle.Files[0].Path = "/tmp/main.yml"
			return bundle
		}, "files"},
		{"extra file", func(bundle *pb.FactoryConfigurationBundle) *pb.FactoryConfigurationBundle {
			bundle.Files = append(bundle.Files, &pb.FactoryConfigurationFile{Path: "extra.yaml", Content: "extra"})
			return bundle
		}, "files"},
		{"missing file", func(bundle *pb.FactoryConfigurationBundle) *pb.FactoryConfigurationBundle {
			bundle.Files = nil
			return bundle
		}, "files"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle := tc.change(proto.Clone(exported).(*pb.FactoryConfigurationBundle))
			plan := f.plan(t, bundle)
			require.NotEmpty(t, plan.Diagnostics)
			require.Equal(t, ".superplane/main.yml", plan.Diagnostics[0].Path)
			require.Equal(t, tc.field, plan.Diagnostics[0].Field)
			require.Empty(t, plan.BaseStateToken)
			_, err := f.apply(bundle, "invalid")
			require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
			require.Equal(t, count, configurationRevisionCount(t, f.canvas))
		})
	}
}

func TestFactoryConfigurationRejectsStalePlanAndRollsBackPublication(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	bundle.Files[0].Content = strings.ReplaceAll(bundle.Files[0].Content, "Original instructions", "Requested instructions")
	plan := f.plan(t, bundle)
	db := database.DB(t.Context())
	// This is the publication path used after a UI save acquires the Canvas lock.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		canvas, err := models.LockCanvasForUpdate(tx, f.r.Organization.ID, f.canvas.ID)
		if err != nil {
			return err
		}
		live, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
		if err != nil {
			return err
		}
		findIntakeNode(live.Nodes, implementationAgentNodeID).Configuration["additionalInstructions"] = "Saved in UI"
		return canvases.PublishGeneratedCanvasNodes(f.ctx, tx, canvas, f.r.User, "UI save", live.Nodes, live.Edges, intakePublisherOptions(testIntakeDependencies(f.r), f.r.Organization.ID))
	}))
	count := configurationRevisionCount(t, f.canvas)
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Equal(t, "Saved in UI", findIntakeNode(resetTestLiveVersion(t, f.canvas).Nodes, implementationAgentNodeID).Configuration["additionalInstructions"])
	require.Equal(t, count, configurationRevisionCount(t, f.canvas))
	require.Contains(t, f.export(t).Files[0].Content, "additional_instructions: Saved in UI")

	plan = f.plan(t, bundle)
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("factory_configuration_publication_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "workflow_nodes" {
			tx.AddError(errors.New("publication failed"))
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Update().Remove("factory_configuration_publication_failure") })
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.Error(t, err)
	require.Equal(t, count, configurationRevisionCount(t, f.canvas))
	require.Equal(t, "Saved in UI", findIntakeNode(resetTestLiveVersion(t, f.canvas).Nodes, implementationAgentNodeID).Configuration["additionalInstructions"])
}

func TestFactoryConfigurationGuardsDoNotWrite(t *testing.T) {
	for _, guard := range []string{"disabled", "permission revoked", "missing binding", "unsupported binding", "changed binding", "different line", "changed desired input", "wrong organization"} {
		t.Run(guard, func(t *testing.T) {
			f := newConfigurationFixture(t)
			bundle := f.export(t)
			bundle.Files[0].Content = strings.ReplaceAll(bundle.Files[0].Content, "Original instructions", "New instructions")
			plan := f.plan(t, bundle)
			db := database.DB(t.Context())
			count := configurationRevisionCount(t, f.canvas)
			switch guard {
			case "disabled":
				require.NoError(t, models.DisableExperimentalFeature(f.r.Organization.ID, features.FeatureFactoryAsCode))
			case "permission revoked":
				f.ctx = authentication.SetUserIdInMetadata(t.Context(), uuid.NewString())
			case "missing binding":
				require.NoError(t, db.Model(f.line).Update("steps", "[]").Error)
			case "unsupported binding":
				live := resetTestLiveVersion(t, f.canvas)
				findIntakeNode(live.Nodes, "onrun-implement").Metadata = models.FactoryAppTemplateMetadataFor("risk-score", 1, models.ProviderGitHub)
				require.NoError(t, db.Save(live).Error)
			case "changed binding":
				max := 3
				f.line.Steps[0].MaxParallelism = &max
				require.NoError(t, db.Save(f.line).Error)
			case "different line":
				line, err := f.factory.CreateLine(db, "Other delivery", f.line.Steps)
				require.NoError(t, err)
				stages := line.StagesValue()
				for i := range stages {
					if stages[i].Kind == models.FactoryLineStagePhase {
						stages[i].Name = "Build changes"
					}
				}
				require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
				f.line = line
			case "changed desired input":
				bundle.Files[0].Content = strings.ReplaceAll(bundle.Files[0].Content, "New instructions", "Other instructions")
			case "wrong organization":
				f.r.Organization.ID = uuid.New()
			}
			if guard == "changed binding" || guard == "different line" {
				fresh := f.plan(t, bundle)
				require.Empty(t, fresh.Diagnostics, "the target remains supported; only the old token is invalid")
				if guard == "changed binding" {
					require.Len(t, fresh.Changes, 2)
				} else {
					require.Len(t, fresh.Changes, 1)
				}
			}
			_, err := f.apply(bundle, plan.BaseStateToken)
			require.Error(t, err)
			if guard == "changed binding" || guard == "different line" {
				require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
			}
			require.Equal(t, count, configurationRevisionCount(t, f.canvas))
			if guard == "disabled" {
				_, err = ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.ID.String(), LineId: f.line.ID.String()})
				require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
				_, err = PlanFactoryConfiguration(f.ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.PlanFactoryConfigurationRequest{FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(), Bundle: bundle})
				require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
			}
		})
	}
}

func TestFactoryConfigurationResolvesWorkspaceWithoutLineID(t *testing.T) {
	f := newConfigurationFixture(t)
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.Key})
	require.NoError(t, err)
	exported.Bundle.Files[0].Content = strings.ReplaceAll(exported.Bundle.Files[0].Content, "Original instructions", "Workspace-selected instructions")
	plan, err := PlanFactoryConfiguration(f.ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.PlanFactoryConfigurationRequest{FactoryId: f.factory.Key, Bundle: exported.Bundle})
	require.NoError(t, err)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1)
	result, err := ApplyFactoryConfiguration(f.ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.ApplyFactoryConfigurationRequest{FactoryId: f.factory.Key, Bundle: exported.Bundle, BaseStateToken: plan.BaseStateToken})
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, "Workspace-selected instructions", findIntakeNode(resetTestLiveVersion(t, f.canvas).Nodes, implementationAgentNodeID).Configuration["additionalInstructions"])
	_, err = f.factory.CreateLine(database.DB(t.Context()), "Another phase", f.line.Steps)
	require.NoError(t, err)
	count := configurationRevisionCount(t, f.canvas)
	_, err = ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.Key})
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	_, err = PlanFactoryConfiguration(f.ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.PlanFactoryConfigurationRequest{FactoryId: f.factory.Key, Bundle: exported.Bundle})
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	_, err = ApplyFactoryConfiguration(f.ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.ApplyFactoryConfigurationRequest{FactoryId: f.factory.Key, Bundle: exported.Bundle, BaseStateToken: plan.BaseStateToken})
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Equal(t, count, configurationRevisionCount(t, f.canvas))
}
