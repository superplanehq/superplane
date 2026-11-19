package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestUpdateFactoryLineStages(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomStages))

	newLine := func(t *testing.T) (*models.Factory, *models.FactoryLine) {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		line, err := factory.CreateLine(db, "ship", nil)
		require.NoError(t, err)
		return factory, line
	}

	t.Run("adds, reorders, renames, colors, and removes custom stages", func(t *testing.T) {
		factory, line := newLine(t)
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:1], append([]*pb.FactoryLine_Stage{
			{Key: "new-security", Name: " Security review ", Kind: pb.FactoryLine_STAGE_KIND_CUSTOM},
			{Key: "new-release", Name: "Release", Kind: pb.FactoryLine_STAGE_KIND_CUSTOM},
		}, stages[1:]...)...)

		created, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			ColumnColors: map[string]string{
				"new-security": "purple",
			},
			StageConfiguration: &pb.FactoryLine_StageConfiguration{
				Stages: stages,
			},
		})
		require.NoError(t, err)
		require.Len(t, created.Line.Stages, 5)
		security := created.Line.Stages[1]
		release := created.Line.Stages[2]
		assert.Equal(t, "Security review", security.Name)
		assert.NotEmpty(t, security.Id)
		assert.Contains(t, security.Key, "custom-")
		assert.Equal(t, "purple", created.Line.ColumnColors[security.Key])

		reordered := []*pb.FactoryLine_Stage{
			created.Line.Stages[0],
			release,
			{Id: security.Id, Key: security.Key, Name: "Security approval", Kind: security.Kind},
			created.Line.Stages[3],
			created.Line.Stages[4],
		}
		updated, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			ColumnColors: map[string]string{
				security.Key: "purple",
			},
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: reordered},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"Backlog", "Release", "Security approval", "Verify", "Done"}, stageNames(updated.Line.Stages))
		assert.Equal(t, "purple", updated.Line.ColumnColors[security.Key])

		withoutSecurity := append([]*pb.FactoryLine_Stage{}, updated.Line.Stages[:2]...)
		withoutSecurity = append(withoutSecurity, updated.Line.Stages[3:]...)
		removed, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: withoutSecurity},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"Backlog", "Release", "Verify", "Done"}, stageNames(removed.Line.Stages))
		assert.NotContains(t, removed.Line.ColumnColors, security.Key)

		reloaded, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"Backlog", "Release", "Verify", "Done"}, modelStageNames(reloaded.StagesValue()))
	})

	t.Run("rejects system stage removal", func(t *testing.T) {
		factory, line := newLine(t)
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:1], stages[2:]...)

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: stages},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects removal when a canvas references the custom stage", func(t *testing.T) {
		factory, line := newLine(t)
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:1], append([]*pb.FactoryLine_Stage{{Name: "Security", Kind: pb.FactoryLine_STAGE_KIND_CUSTOM}}, stages[1:]...)...)
		created, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: stages},
		})
		require.NoError(t, err)

		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "security")
		require.NoError(t, db.Model(canvas).Update("column_key", created.Line.Stages[1].Key).Error)
		withoutCustom := append([]*pb.FactoryLine_Stage{created.Line.Stages[0]}, created.Line.Stages[2:]...)
		_, err = UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: withoutCustom},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
	})

	t.Run("requires the custom stages feature", func(t *testing.T) {
		factory, line := newLine(t)
		require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomStages))
		t.Cleanup(func() {
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomStages))
		})

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: factoryLineStageRequests(line.StagesValue())},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
	})

	t.Run("rejects replacing steps and stages together", func(t *testing.T) {
		factory, line := newLine(t)
		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			Steps:              []*pb.FactoryLine_Step{{}},
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: factoryLineStageRequests(line.StagesValue())},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})
}

func factoryLineStageRequests(stages []models.FactoryLineStage) []*pb.FactoryLine_Stage {
	return serializeFactoryLineStages(stages)
}

func stageNames(stages []*pb.FactoryLine_Stage) []string {
	result := make([]string, len(stages))
	for index, stage := range stages {
		result[index] = stage.GetName()
	}
	return result
}

func modelStageNames(stages []models.FactoryLineStage) []string {
	result := make([]string, len(stages))
	for index, stage := range stages {
		result[index] = stage.Name
	}
	return result
}
