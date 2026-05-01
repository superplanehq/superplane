package organizations

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func UpdateIntegrationProperty(
	ctx context.Context,
	registry *registry.Registry,
	orgID string,
	integrationID string,
	propertyName string,
	value string,
) (*pb.UpdateIntegrationPropertyResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	id, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid integration ID")
	}

	integration, err := models.FindIntegration(org, id)
	if err != nil {
		return nil, err
	}

	property, err := findProperty(integration, propertyName)
	if err != nil {
		return nil, err
	}

	if !property.Editable {
		return nil, status.Errorf(codes.InvalidArgument, "property %s is not editable", propertyName)
	}

	setupProvider, err := registry.GetSetupProvider(integration.AppName)
	if err != nil {
		return nil, err
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		secretStorage, err := contexts.NewIntegrationSecretStorage(tx, registry.Encryptor, integration)
		if err != nil {
			return err
		}

		setupStep, err := setupProvider.OnPropertyUpdate(core.PropertyUpdateContext{
			PropertyName: propertyName,
			Value:        value,
			Logger:       logrus.WithField("integration_id", integration.ID),
			HTTP:         registry.HTTPContext(),
			Secrets:      secretStorage,
			Properties:   contexts.NewIntegrationPropertyStorage(integration),
			Capabilities: contexts.NewCapabilityContext(allCapabilities(setupProvider), integration.Capabilities),
		})

		if err != nil {
			return err
		}

		//
		// If a new setup step is returned, we need to update the setup state.
		//
		if setupStep != nil {
			newState := datatypes.NewJSONType(models.SetupState{
				CurrentStep:   setupStep,
				PreviousSteps: []core.SetupStep{},
			})

			integration.SetupState = &newState
		}

		now := time.Now()
		integration.UpdatedAt = &now
		return tx.Save(integration).Error
	})

	if err != nil {
		logrus.WithError(err).Error("failed to update integration property")
		return nil, status.Error(codes.Internal, "failed to update integration property")
	}

	proto, err := serializeIntegration(registry, integration, []models.CanvasNodeReference{})
	if err != nil {
		return nil, err
	}

	return &pb.UpdateIntegrationPropertyResponse{
		Integration: proto,
	}, nil
}

func findProperty(integration *models.Integration, propertyName string) (*core.IntegrationPropertyDefinition, error) {
	for _, property := range integration.Properties {
		if property.Name == propertyName {
			return &property, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "property %s not found", propertyName)
}
