package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
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
		source := &models.AutomationSource{ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID, Name: automation.GetName(), Source: "name: Preview environment", Revision: 1, ArtifactID: automation.GetId()}
		require.NoError(t, source.Save(database.DB(t.Context())))

		_, err := DeleteFactoryAutomation(ctx, orgID, &pb.DeleteFactoryAutomationRequest{
			FactoryId:    factory.ID.String(),
			AutomationId: automation.GetId(),
		})
		require.NoError(t, err)

		_, err = models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(automation.GetId()))
		assert.Error(t, err)
		_, err = models.FindAutomationSource(database.DB(t.Context()), r.Organization.ID, source.ID)
		assert.Error(t, err)
	})

	t.Run("rejects another workspace or organization", func(t *testing.T) {
		factory := newFactory(t)
		automation := createAutomation(t, factory)
		other := newFactory(t)
		for _, request := range []struct{ organizationID, factoryID string }{
			{orgID, other.ID.String()},
			{uuid.NewString(), factory.ID.String()},
		} {
			_, err := DeleteFactoryAutomation(ctx, request.organizationID, &pb.DeleteFactoryAutomationRequest{FactoryId: request.factoryID, AutomationId: automation.GetId()})
			require.Error(t, err)
			code, _, ok := grpcerrors.HandlerStatus(err)
			require.True(t, ok)
			assert.Equal(t, codes.NotFound, code)
		}
		_, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(automation.GetId()))
		require.NoError(t, err)
	})

	t.Run("rejects an intake canvas", func(t *testing.T) {
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
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "This canvas belongs to a factory intake, line, backlog, or PR feedback handler.", message)
	})

	t.Run("deletes task analysis without reprovisioning it", func(t *testing.T) {
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
		factory, err = models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		require.NoError(t, err)
		branch := "develop"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{DefaultBranch: &branch}))
		_, err = CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(), Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)
		has, err := factoryHasOnWorkOrderCanvas(database.DB(t.Context()), factory)
		require.NoError(t, err)
		assert.False(t, has)
		restored, err := CreateFactoryAutomation(ctx, deps, orgID, &pb.CreateFactoryAutomationRequest{
			FactoryId: factory.ID.String(), TemplateId: models.FactoryAppTemplateBacklogID,
		})
		require.NoError(t, err)
		assert.NotEqual(t, backlog.ID.String(), restored.GetAutomation().GetId())
		runs, err := models.ListCanvasRuns(database.DB(t.Context()), uuid.MustParse(restored.GetAutomation().GetId()), 10, nil, models.CanvasRunFilters{})
		require.NoError(t, err)
		assert.Empty(t, runs)
		_, err = CreateFactoryAutomation(ctx, deps, orgID, &pb.CreateFactoryAutomationRequest{FactoryId: factory.ID.String(), TemplateId: models.FactoryAppTemplateBacklogID})
		require.Error(t, err)
	})

	t.Run("deletes a line step and retains its stage", func(t *testing.T) {
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
		appID, lineID := automation.GetId(), line.ID.String()
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{ProvisionedAppID: &appID, ProvisionedLineID: &lineID}))

		_, err = DeleteFactoryAutomation(ctx, orgID, &pb.DeleteFactoryAutomationRequest{
			FactoryId:    factory.ID.String(),
			AutomationId: automation.GetId(),
		})
		require.NoError(t, err)
		reloaded, err := factory.FindLine(database.DB(t.Context()), line.ID)
		require.NoError(t, err)
		assert.Empty(t, reloaded.Steps)
		updatedFactory, err := models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.Empty(t, updatedFactory.OnboardingConfigValue().ProvisionedAppID)
		assert.Empty(t, updatedFactory.OnboardingConfigValue().ProvisionedLineID)
		require.Len(t, reloaded.StagesValue(), len(line.StagesValue()))
		assert.Equal(t, line.StagesValue()[1].ID, reloaded.StagesValue()[1].ID)
		assert.Nil(t, reloaded.StagesValue()[1].StepIndex)
		restored, err := CreateFactoryAutomation(ctx, deps, orgID, &pb.CreateFactoryAutomationRequest{
			FactoryId: factory.ID.String(), TemplateId: "line-implementation",
			InstallParams: map[string]string{"lineId": line.ID.String(), "stageKey": line.StagesValue()[1].Key},
		})
		require.NoError(t, err)
		reloaded, err = factory.FindLine(database.DB(t.Context()), line.ID)
		require.NoError(t, err)
		require.Len(t, reloaded.Steps, 1)
		assert.Equal(t, restored.GetAutomation().GetId(), reloaded.Steps[0].AppID.String())
		assert.Equal(t, line.StagesValue()[1].ID, reloaded.StagesValue()[1].ID)
		_, err = CreateFactoryAutomation(ctx, deps, orgID, &pb.CreateFactoryAutomationRequest{
			FactoryId: factory.ID.String(), TemplateId: "line-implementation",
			InstallParams: map[string]string{"lineId": line.ID.String(), "stageKey": line.StagesValue()[1].Key},
		})
		require.Error(t, err)
	})

	t.Run("rejects a PR feedback canvas", func(t *testing.T) {
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
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
	})
}
