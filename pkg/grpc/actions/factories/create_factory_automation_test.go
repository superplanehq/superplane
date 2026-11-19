package factories

import (
	"context"
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

func Test__CreateFactoryAutomation(t *testing.T) {
	r := support.Setup(t)

	t.Run("feature off -> error", func(t *testing.T) {
		_, err := CreateFactoryAutomation(context.Background(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAutomationRequest{
			FactoryId: "00000000-0000-0000-0000-000000000001",
			Name:      "Preview environment",
			ColumnKey: models.CanvasColumnKeyVerify,
		})
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "Custom automations are not enabled for this organization.", message)
	})

	t.Run("feature on continues past the flag", func(t *testing.T) {
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))

		_, err := CreateFactoryAutomation(context.Background(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAutomationRequest{
			FactoryId: "00000000-0000-0000-0000-000000000001",
			Name:      "Preview environment",
			ColumnKey: "invalid",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})
	t.Run("custom automation cannot bypass the scheduled intake gate", func(t *testing.T) {
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
		_, err := CreateFactoryAutomation(context.Background(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAutomationRequest{
			FactoryId: "00000000-0000-0000-0000-000000000001", ColumnKey: models.CanvasColumnKeyBacklog,
		})
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, code)
		require.Contains(t, message, "scheduled intake template")
	})

	t.Run("attaches an automation to a custom stage", func(t *testing.T) {
		db := database.DB(t.Context())
		require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
		t.Cleanup(func() {
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
		})
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomStages))
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		support.CompleteFactoryOnboarding(t, factory)
		line, err := factory.CreateLine(db, "Main", nil)
		require.NoError(t, err)
		stageID := uuid.New()
		stageKey := "custom-" + stageID.String()
		stages := line.StagesValue()
		stages = append(stages[:len(stages)-1], models.FactoryLineStage{
			ID: stageID, Key: stageKey, Name: "QA", Kind: models.FactoryLineStageCustom,
		}, stages[len(stages)-1])
		require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))

		response, err := CreateFactoryAutomation(
			onboardingContext(r),
			onboardingDeps(r),
			r.Organization.ID.String(),
			&pb.CreateFactoryAutomationRequest{FactoryId: factory.ID.String(), Name: "QA checks", ColumnKey: stageKey},
		)
		require.NoError(t, err)
		assert.Equal(t, stageKey, response.GetAutomation().GetColumnKey())
	})

}
