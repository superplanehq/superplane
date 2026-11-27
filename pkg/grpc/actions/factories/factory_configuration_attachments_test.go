package factories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

func TestFactoryConfigurationPreservesSharedStageAttachments(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	id := uuid.New()
	shared := models.FactoryLineStage{ID: id, Key: "custom-" + id.String(), Name: "Recheck", Kind: models.FactoryLineStagePhase}
	stages := f.line.StagesValue()
	done := stages[len(stages)-1]
	stages = append(stages[:len(stages)-1], shared, done)
	require.NoError(t, f.line.UpdateConfiguration(db, nil, nil, nil, stages))
	require.NoError(t, f.canvas.SetColumnKey(db, shared.Key))
	beforeSteps, err := f.line.ExecutionPlan(db)
	require.NoError(t, err)
	require.Len(t, beforeSteps, 2)
	beforeRevisionCount := configurationRevisionCount(t, f.canvas)
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.Key})
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(exported.Bundle.Files[0].Content, "@superplane/implement"), "Export one workspace-wide automation identity while retaining both execution bindings")
	plan := f.plan(t, exported.Bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	exported.Bundle.Files[0].Content = strings.ReplaceAll(exported.Bundle.Files[0].Content, "Original instructions", "Keep both stage bindings.")
	plan = f.plan(t, exported.Bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1, "Plan shared settings once")
	applied, err := f.apply(exported.Bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, applied.Changed)
	require.Equal(t, beforeRevisionCount+1, configurationRevisionCount(t, f.canvas), "Publish one revision for the shared automation")
	line, err := f.factory.FindLine(db, f.line.ID)
	require.NoError(t, err)
	afterSteps, err := line.ExecutionPlan(db)
	require.NoError(t, err)
	require.Equal(t, beforeSteps, afterSteps)
	canvas, err := models.FindCanvasInTransaction(db, f.factory.OrganizationID, f.canvas.ID)
	require.NoError(t, err)
	require.Equal(t, shared.Key, *canvas.ColumnKey)
}

func TestFactoryConfigurationPreservesSharedPhaseRunner(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	firstLimit, secondLimit := 3, 20
	steps := []models.FactoryLineStep{f.line.Steps[0], f.line.Steps[0]}
	steps[0].MaxParallelism, steps[1].MaxParallelism = &firstLimit, &secondLimit
	require.NoError(t, f.line.UpdateConfiguration(db, nil, steps, nil, nil))
	stages := f.line.StagesValue()
	for i := range stages {
		if stages[i].StepIndex != nil && *stages[i].StepIndex == 1 {
			stages[i].Name = "Recheck shared task"
		}
	}
	require.NoError(t, f.line.UpdateConfiguration(db, nil, nil, nil, stages))
	before := configurationResourceCounts(t)
	revisions := configurationRevisionCount(t, f.canvas)
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	var boundPhases []int
	declaredPhase := -1
	declarations := 0
	for i, stage := range manifest.Stages {
		if stage.Run == f.canvas.ConfigurationID {
			boundPhases = append(boundPhases, i)
		}
		for _, automation := range stage.Automations {
			if automation.ID == f.canvas.ConfigurationID {
				declaredPhase = i
				declarations++
			}
		}
	}
	require.Len(t, boundPhases, 2)
	require.Equal(t, 1, declarations)
	require.Contains(t, boundPhases, declaredPhase)
	otherPhase := boundPhases[0]
	if otherPhase == declaredPhase {
		otherPhase = boundPhases[1]
	}
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	applied, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, applied.Changed)

	// A declaration can move between its bound phases without moving its execution placement.
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		first, second := &manifest.Stages[declaredPhase], &manifest.Stages[otherPhase]
		second.Automations = first.Automations
		first.Automations = []factoryConfigurationAutomation{}
	})
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	applied, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, applied.Changed)
	require.Equal(t, before, configurationResourceCounts(t))
	require.Equal(t, revisions, configurationRevisionCount(t, f.canvas))

	bundle.Files[0].Content = strings.ReplaceAll(bundle.Files[0].Content, "Original instructions", "Keep both phase bindings.")
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1)
	applied, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, applied.Changed)
	require.Equal(t, revisions+1, configurationRevisionCount(t, f.canvas))
	line, err := f.factory.FindLine(db, f.line.ID)
	require.NoError(t, err)
	require.Equal(t, f.line.Steps, line.Steps)
	canvas, err := models.FindCanvasInTransaction(db, f.factory.OrganizationID, f.canvas.ID)
	require.NoError(t, err)
	require.Nil(t, canvas.ColumnKey)
	require.Equal(t, "Keep both phase bindings.", findIntakeNode(resetTestLiveVersion(t, canvas).Nodes, implementationAgentNodeID).Configuration["additionalInstructions"])

	for _, placement := range []string{"missing declaration", "unrelated stage"} {
		t.Run(placement, func(t *testing.T) {
			invalid := proto.Clone(bundle).(*pb.FactoryConfigurationBundle)
			editConfigurationManifest(t, invalid, func(manifest *factoryConfigurationManifest) {
				declared := manifest.Stages[otherPhase].Automations
				manifest.Stages[otherPhase].Automations = []factoryConfigurationAutomation{}
				if placement == "unrelated stage" {
					manifest.Stages[len(manifest.Stages)-2].Automations = declared
				}
			})
			before := configurationResourceCounts(t)
			plan := f.plan(t, invalid)
			require.NotEmpty(t, plan.Diagnostics)
			require.Empty(t, plan.BaseStateToken)
			_, err := f.apply(invalid, "invalid")
			require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
			require.Equal(t, before, configurationResourceCounts(t))
		})
	}
}
