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
		GitProvider:    r.GitProvider,
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

	t.Run("rejects a backlog canvas", func(t *testing.T) {
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
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
	})

	t.Run("rejects a line step canvas", func(t *testing.T) {
		factory := newFactory(t)
		automation := createAutomation(t, factory)
		_, err := factory.CreateLine(database.DB(t.Context()), "ship", []models.FactoryLineStep{
			{
				Type:       models.FactoryLineStepTypeRunApp,
				AppID:      uuid.MustParse(automation.GetId()),
				Entrypoint: "start",
			},
		})
		require.NoError(t, err)

		_, err = DeleteFactoryAutomation(ctx, orgID, &pb.DeleteFactoryAutomationRequest{
			FactoryId:    factory.ID.String(),
			AutomationId: automation.GetId(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
	})
}
