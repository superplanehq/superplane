package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__DeleteFactoryAutomation(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))

	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	createAutomation := func(t *testing.T, factory *models.Factory) *pb.Factory_Automation {
		t.Helper()
		response, err := CreateFactoryAutomation(ctx, deps, orgID, &pb.CreateFactoryAutomationRequest{
			FactoryId: factory.ID.String(),
			Name:      "Preview environment",
		})
		require.NoError(t, err)
		return response.GetAutomation()
	}

	t.Run("deletes a custom automation", func(t *testing.T) {
		factory := newFactory(t)
		automation := createAutomation(t, factory)

		_, err := DeleteFactoryAutomation(ctx, orgID, &pb.DeleteFactoryAutomationRequest{
			FactoryId:    factory.ID.String(),
			AutomationId: automation.GetId(),
		})
		require.NoError(t, err)

		_, err = models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(automation.GetId()))
		assert.Error(t, err)
	})

	t.Run("deletes an intake and its registration", func(t *testing.T) {
		factory := newFactory(t)
		response, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		_, err = DeleteFactoryAutomation(ctx, orgID, &pb.DeleteFactoryAutomationRequest{
			FactoryId:    factory.ID.String(),
			AutomationId: response.GetIntake().GetCanvasId(),
		})
		require.NoError(t, err)
		_, err = models.FindFactoryIntakeByCanvasID(database.DB(t.Context()), uuid.MustParse(response.GetIntake().GetCanvasId()))
		assert.ErrorIs(t, err, models.ErrFactoryIntakeNotFound)
	})

	t.Run("deletes a backlog canvas", func(t *testing.T) {
		factory := newFactory(t)
		_, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)
		backlog := liveBacklogCanvas(t, factory)

		_, err = DeleteFactoryAutomation(ctx, orgID, &pb.DeleteFactoryAutomationRequest{
			FactoryId:    factory.ID.String(),
			AutomationId: backlog.ID.String(),
		})
		require.NoError(t, err)
		_, err = models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, backlog.ID)
		assert.Error(t, err)
	})

	t.Run("deletes the final step and preserves its stage", func(t *testing.T) {
		factory := newFactory(t)
		automation := createAutomation(t, factory)
		line, err := factory.CreateLine(database.DB(t.Context()), "ship", []models.FactoryLineStep{
			{
				Type:       models.FactoryLineStepTypeRunApp,
				AppID:      uuid.MustParse(automation.GetId()),
				Entrypoint: "start",
			},
		})
		require.NoError(t, err)
		stages := line.StagesValue()
		colors := line.ColumnColorsValue()

		_, err = DeleteFactoryAutomation(ctx, orgID, &pb.DeleteFactoryAutomationRequest{
			FactoryId:    factory.ID.String(),
			AutomationId: automation.GetId(),
		})
		require.NoError(t, err)
		line, err = factory.FindLine(database.DB(t.Context()), line.ID)
		require.NoError(t, err)
		assert.Empty(t, line.Steps)
		stages[1].StepIndex = nil
		assert.Equal(t, stages, line.StagesValue())
		assert.Equal(t, colors, line.ColumnColorsValue())
		_, err = line.ExecutionPlan(database.DB(t.Context()))
		assert.ErrorIs(t, err, models.ErrFactoryLineHasNoSteps)
	})

	t.Run("deletes a PR feedback canvas and its registration", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		response, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)

		_, err = DeleteFactoryAutomation(ctx, orgID, &pb.DeleteFactoryAutomationRequest{
			FactoryId:    factory.ID.String(),
			AutomationId: response.GetHandler().GetCanvasId(),
		})
		require.NoError(t, err)
		handler, err := models.FindPRFeedbackHandlerByCanvasID(database.DB(t.Context()), uuid.MustParse(response.GetHandler().GetCanvasId()))
		require.NoError(t, err)
		assert.Nil(t, handler)
	})

	t.Run("remaps shared steps without changing stages on a later settings save", func(t *testing.T) {
		factory := newFactory(t)
		deleted := uuid.MustParse(createAutomation(t, factory).GetId())
		retained := uuid.MustParse(createAutomation(t, factory).GetId())
		db := database.DB(t.Context())
		var lines []*models.FactoryLine
		for _, name := range []string{"first", "second"} {
			line, err := factory.CreateLine(db, name, []models.FactoryLineStep{
				{Type: models.FactoryLineStepTypeRunApp, AppID: deleted, Entrypoint: "start"},
				{Type: models.FactoryLineStepTypeRunApp, AppID: retained, Entrypoint: "start"},
				{Type: models.FactoryLineStepTypeRunApp, AppID: deleted, Entrypoint: "start"},
			})
			require.NoError(t, err)
			lines = append(lines, line)
		}
		_, err := DeleteFactoryAutomation(ctx, orgID, &pb.DeleteFactoryAutomationRequest{FactoryId: factory.ID.String(), AutomationId: deleted.String()})
		require.NoError(t, err)
		for _, original := range lines {
			line, err := factory.FindLine(db, original.ID)
			require.NoError(t, err)
			require.Len(t, line.Steps, 1)
			assert.Equal(t, retained, line.Steps[0].AppID)
			expected := original.StagesValue()
			expected[1].StepIndex, expected[3].StepIndex = nil, nil
			index := 0
			expected[2].StepIndex = &index
			assert.Equal(t, expected, line.StagesValue())
			max := 3
			steps := []models.FactoryLineStep(line.Steps)
			steps[0].MaxParallelism = &max
			require.NoError(t, line.Update(db, nil, steps, nil))
			assert.Equal(t, expected, line.StagesValue())
			plan, err := line.ExecutionPlan(db)
			require.NoError(t, err)
			require.Len(t, plan, 1)
			assert.Equal(t, "phase-1", plan[0].StageKey)
			assert.Equal(t, retained, plan[0].AppID)
			assert.Equal(t, 3, plan[0].EffectiveMaxParallelism())
		}
	})

	t.Run("rejects another workspace or organization", func(t *testing.T) {
		factory := newFactory(t)
		automation := createAutomation(t, factory)
		other := newFactory(t)
		for _, request := range []struct{ org, factory string }{{orgID, other.ID.String()}, {uuid.NewString(), factory.ID.String()}} {
			_, err := DeleteFactoryAutomation(ctx, request.org, &pb.DeleteFactoryAutomationRequest{FactoryId: request.factory, AutomationId: automation.GetId()})
			code, _, ok := grpcerrors.HandlerStatus(err)
			assert.True(t, ok)
			assert.Equal(t, codes.NotFound, code)
		}
		_, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(automation.GetId()))
		require.NoError(t, err)
	})
}
