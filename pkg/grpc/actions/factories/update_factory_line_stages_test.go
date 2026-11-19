package factories

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
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

	t.Run("removes optional system stages", func(t *testing.T) {
		factory, line := newLine(t)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "verify")
		require.NoError(t, canvas.SetColumnKey(db, models.CanvasColumnKeyVerify))
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:1], stages[2:]...)

		updated, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: stages},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"Backlog", "Done"}, stageNames(updated.Line.Stages))
		reloaded, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		require.NotNil(t, reloaded.ColumnKey)
		assert.Equal(t, models.CanvasColumnKeyVerify, *reloaded.ColumnKey)
	})

	t.Run("keeps Backlog and Done fixed", func(t *testing.T) {
		factory, line := newLine(t)
		stages := factoryLineStageRequests(line.StagesValue())

		for _, invalid := range [][]*pb.FactoryLine_Stage{stages[1:], stages[:len(stages)-1]} {
			_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
				FactoryId:          factory.ID.String(),
				LineId:             line.ID.String(),
				StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: invalid},
			})
			code, _, ok := grpcerrors.HandlerStatus(err)
			assert.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, code)
		}
	})

	t.Run("reorders optional system stages", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start")
		line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
			Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint,
		}})
		require.NoError(t, err)
		stages := factoryLineStageRequests(line.StagesValue())
		reordered := []*pb.FactoryLine_Stage{stages[0], stages[2], stages[1], stages[3]}

		updated, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: reordered},
		})
		require.NoError(t, err)
		require.Len(t, updated.Line.Stages, 4)
		assert.Equal(t, []string{stages[0].Id, stages[2].Id, stages[1].Id, stages[3].Id}, []string{
			updated.Line.Stages[0].Id,
			updated.Line.Stages[1].Id,
			updated.Line.Stages[2].Id,
			updated.Line.Stages[3].Id,
		})
	})

	t.Run("detaches automations when their stage is removed", func(t *testing.T) {
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
		require.NoError(t, err)
		reloaded, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		assert.Nil(t, reloaded.ColumnKey)
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

	t.Run("rejects a step update that would exceed the stage limit", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start")
		step := models.FactoryLineStep{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      app.ID,
			Entrypoint: entrypoint,
		}
		line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{step})
		require.NoError(t, err)

		customStages := make([]models.FactoryLineStage, 60)
		for index := range customStages {
			id := uuid.New()
			customStages[index] = models.FactoryLineStage{
				ID:   id,
				Key:  "custom-" + id.String(),
				Name: fmt.Sprintf("Review %d", index+1),
				Kind: models.FactoryLineStageCustom,
			}
		}
		stages := line.StagesValue()
		stages = append(stages[:len(stages)-1], append(customStages, stages[len(stages)-1])...)
		require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))

		stepRequest := &pb.FactoryLine_Step{
			Type: models.FactoryLineStepTypeRunApp,
			App:  &pb.FactoryLine_AppStep{App: app.ID.String(), Entrypoint: entrypoint},
		}
		_, err = UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			Steps:     []*pb.FactoryLine_Step{stepRequest, stepRequest},
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
