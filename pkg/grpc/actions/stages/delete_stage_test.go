package stages

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__DeleteStage(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})

	t.Run("wrong canvas -> error", func(t *testing.T) {
		stage := models.Stage{
			CanvasID:      r.Canvas.ID,
			Name:          "test-stage-wrong-canvas",
			Description:   "Test Stage",
			ExecutorType:  models.ExecutorTypeHTTP,
			ExecutorName:  "test-executor",
			ExecutorSpec:  datatypes.JSON(`{}`),
			Conditions:    datatypes.NewJSONSlice([]models.StageCondition{}),
			Inputs:        datatypes.NewJSONSlice([]models.InputDefinition{}),
			InputMappings: datatypes.NewJSONSlice([]models.InputMapping{}),
			Outputs:       datatypes.NewJSONSlice([]models.OutputDefinition{}),
			Secrets:       datatypes.NewJSONSlice([]models.ValueDefinition{}),
		}
		err := database.Conn().Create(&stage).Error
		require.NoError(t, err)

		_, err = DeleteStage(context.Background(), uuid.NewString(), stage.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage that does not exist -> error", func(t *testing.T) {
		_, err := DeleteStage(context.Background(), r.Canvas.ID.String(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("delete stage by id successfully", func(t *testing.T) {
		stage := models.Stage{
			CanvasID:      r.Canvas.ID,
			Name:          "test-stage-by-id",
			Description:   "Test Stage By ID",
			ExecutorType:  models.ExecutorTypeHTTP,
			ExecutorName:  "test-executor",
			ExecutorSpec:  datatypes.JSON(`{}`),
			Conditions:    datatypes.NewJSONSlice([]models.StageCondition{}),
			Inputs:        datatypes.NewJSONSlice([]models.InputDefinition{}),
			InputMappings: datatypes.NewJSONSlice([]models.InputMapping{}),
			Outputs:       datatypes.NewJSONSlice([]models.OutputDefinition{}),
			Secrets:       datatypes.NewJSONSlice([]models.ValueDefinition{}),
		}
		err := database.Conn().Create(&stage).Error
		require.NoError(t, err)

		response, err := DeleteStage(context.Background(), r.Canvas.ID.String(), stage.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindStageByID(r.Canvas.ID.String(), stage.ID.String())
		assert.Error(t, err)
		assert.True(t, err == gorm.ErrRecordNotFound)

		softDeletedStages, err := models.ListUnscopedSoftDeletedStages(10)
		require.NoError(t, err)
		found := false
		for _, s := range softDeletedStages {
			if s.ID == stage.ID {
				found = true
				assert.Contains(t, s.Name, "deleted-")
				break
			}
		}
		assert.True(t, found, "Stage should be in soft deleted list")
	})

	t.Run("delete stage by name successfully", func(t *testing.T) {
		stage := models.Stage{
			CanvasID:      r.Canvas.ID,
			Name:          "test-stage-by-name",
			Description:   "Test Stage By Name",
			ExecutorType:  models.ExecutorTypeHTTP,
			ExecutorName:  "test-executor",
			ExecutorSpec:  datatypes.JSON(`{}`),
			Conditions:    datatypes.NewJSONSlice([]models.StageCondition{}),
			Inputs:        datatypes.NewJSONSlice([]models.InputDefinition{}),
			InputMappings: datatypes.NewJSONSlice([]models.InputMapping{}),
			Outputs:       datatypes.NewJSONSlice([]models.OutputDefinition{}),
			Secrets:       datatypes.NewJSONSlice([]models.ValueDefinition{}),
		}
		err := database.Conn().Create(&stage).Error
		require.NoError(t, err)

		response, err := DeleteStage(context.Background(), r.Canvas.ID.String(), stage.Name)
		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindStageByName(r.Canvas.ID.String(), stage.Name)
		assert.Error(t, err)
		assert.True(t, err == gorm.ErrRecordNotFound)
	})
}
