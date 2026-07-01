package organizations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func PreviousIntegrationSetupStep(ctx context.Context, registry *registry.Registry, orgID, id string) (*pb.PreviousIntegrationSetupStepResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization ID")
	}

	integrationID, err := uuid.Parse(id)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid integration ID")
	}

	integration, err := models.FindIntegration(org, integrationID)
	if err != nil {
		logrus.WithError(err).Error("failed to find integration")

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "integration not found")
		}

		return nil, grpcerrors.Internal(err, "failed to find integration")
	}

	//
	// Verify that we are in a revertable state
	//
	if integration.SetupState == nil {
		return nil, grpcerrors.InvalidArgument(nil, "current step is not set, cannot revert")
	}

	setupState := integration.SetupState.Data()
	if setupState.CurrentStep == nil {
		return nil, grpcerrors.InvalidArgument(nil, "current step is not set, cannot revert")
	}

	if len(setupState.PreviousSteps) == 0 {
		return nil, grpcerrors.InvalidArgument(nil, "no previous steps, cannot revert")
	}

	//
	// Find setup provider and revert integration to the previous step
	//
	setupProvider, err := registry.GetSetupProvider(integration.AppName)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to get setup provider")
	}

	logrus.WithField("integration_id", integration.ID).WithField("source", "setup_step_revert").Info("Integration operation may write secrets")
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		stepToRevert := setupState.PreviousSteps[len(setupState.PreviousSteps)-1]
		capabilityCtx := contexts.NewCapabilityContext(registry.AllCapabilities(integration.AppName), integration.Capabilities)
		ctx := core.SetupStepContext{
			Step:           core.StepInfo{Name: stepToRevert.Name},
			IntegrationID:  integration.ID,
			OrganizationID: orgID,
			HTTP:           registry.HTTPContextInTransaction(tx),
			Properties:     contexts.NewIntegrationPropertyStorage(integration),
			Secrets:        contexts.NewIntegrationSecretStorage(tx, registry.Encryptor, integration),
			Capabilities:   capabilityCtx,
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
		integration.Capabilities = capabilityCtx.States()
		return tx.Save(integration).Error
	})

	if err != nil {
		logrus.WithError(err).Error("failed to submit integration setup step")
		return nil, grpcerrors.Internal(err, "failed to submit integration setup step")
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
