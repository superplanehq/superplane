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
	"google.golang.org/protobuf/types/known/structpb"
)

func SubmitIntegrationSetupStep(ctx context.Context, registry *registry.Registry, orgID, id, stepName string, inputs *structpb.Struct) (*pb.SubmitIntegrationSetupStepResponse, error) {
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

	setupProvider, err := registry.GetSetupProvider(integration.AppName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get setup provider")
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		secretStorage, err := contexts.NewIntegrationSecretStorage(tx, registry.Encryptor, integration)
		if err != nil {
			return err
		}

		nextStep, err := setupProvider.OnStepSubmit(stepName, getStepInputs(inputs), core.SetupStepContext{
			IntegrationID:  integration.ID,
			OrganizationID: orgID,
			HTTP:           registry.HTTPContext(),
			Parameters:     contexts.NewIntegrationParameterStorage(integration),
			Capabilities:   contexts.NewIntegrationCapabilityRegistry(integration),
			Secrets:        secretStorage,
		})

		if err != nil {
			return err
		}

		//
		// If no next step, make integration ready
		//
		if nextStep == nil {
			integration.NextSetupStep = nil
			integration.State = models.IntegrationStateReady
			return tx.Save(integration).Error
		}

		//
		// Otherwise, set the next step
		//
		nextSetupStep := datatypes.NewJSONType(*nextStep)
		integration.NextSetupStep = &nextSetupStep
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

	return &pb.SubmitIntegrationSetupStepResponse{
		Integration: proto,
	}, nil
}

func getStepInputs(inputs *structpb.Struct) map[string]any {
	if inputs == nil {
		return map[string]any{}
	}

	return inputs.AsMap()
}
