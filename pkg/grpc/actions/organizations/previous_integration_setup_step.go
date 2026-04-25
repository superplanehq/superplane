package organizations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func PreviousIntegrationSetupStep(ctx context.Context, registry *registry.Registry, orgID, id string) (*pb.PreviousIntegrationSetupStepResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	integrationID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid integration ID")
	}

	integration, err := models.FindIntegration(org, integrationID)
	if err != nil {
		logrus.WithError(err).Error("failed to find integration")

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "integration not found")
		}

		return nil, status.Error(codes.Internal, "failed to find integration")
	}

	//
	// Verify that we are in a revertable state
	//
	setupState := integration.SetupState.Data()
	if setupState.CurrentStep == nil {
		return nil, status.Error(codes.InvalidArgument, "current step is not set, cannot revert")
	}

	if setupState.PreviousSteps == nil || len(setupState.PreviousSteps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no previous steps, cannot revert")
	}

	//
	// Find setup provider and revert integration to the previous step
	//
	setupProvider, err := registry.GetSetupProvider(integration.AppName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get setup provider")
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		secretStorage, err := contexts.NewIntegrationSecretStorage(tx, registry.Encryptor, integration)
		if err != nil {
			return err
		}

		stepToRevert := setupState.PreviousSteps[len(setupState.PreviousSteps)-1]

		ctx := core.SetupStepContext{
			Step:           stepToRevert.Name,
			IntegrationID:  integration.ID,
			OrganizationID: orgID,
			HTTP:           registry.HTTPContext(),
			Parameters:     contexts.NewIntegrationParameterStorage(integration),
			Capabilities:   contexts.NewIntegrationCapabilityRegistry(integration),
			Secrets:        secretStorage,
		}

		err = setupProvider.OnStepRevert(ctx)
		if err != nil {
			return err
		}

		//
		// Current step becomes the step we just reverted to,
		// and previous steps are all previous steps before it.
		//
		newState := models.SetupState{}
		newState.CurrentStep = &stepToRevert
		newState.PreviousSteps = setupState.PreviousSteps[:len(setupState.PreviousSteps)-1]
		newStateWrapper := datatypes.NewJSONType(newState)
		integration.SetupState = &newStateWrapper
		return tx.Save(integration).Error
	})

	if err != nil {
		logrus.WithError(err).Error("failed to submit integration setup step")
		return nil, status.Error(codes.Internal, "failed to submit integration setup step")
	}

	proto, err := serializeIntegration(registry, integration, []models.CanvasNodeReference{})
	if err != nil {
		logrus.WithError(err).Error("failed to serialize integration")
		return nil, err
	}

	return &pb.PreviousIntegrationSetupStepResponse{
		Integration: proto,
	}, nil
}
