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

func UpdateIntegrationSecret(
	ctx context.Context,
	registry *registry.Registry,
	orgID string,
	integrationID string,
	secretName string,
	value string,
) (*pb.UpdateIntegrationSecretResponse, error) {
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

	secret, err := findSecret(integration, secretName)
	if err != nil {
		return nil, err
	}

	if !secret.Editable {
		return nil, status.Errorf(codes.InvalidArgument, "secret %s is not editable", secretName)
	}

	setupProvider, err := registry.GetSetupProvider(integration.AppName)
	if err != nil {
		return nil, err
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		setupStep, err := setupProvider.OnSecretUpdate(core.SecretUpdateContext{
			SecretName:   secretName,
			Value:        value,
			Logger:       logrus.WithField("integration_id", integration.ID),
			HTTP:         registry.HTTPContext(),
			Secrets:      contexts.NewIntegrationSecretStorage(tx, registry.Encryptor, integration),
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
		logrus.WithError(err).Error("failed to update integration parameter")
		return nil, status.Errorf(codes.Internal, "failed to update integration secret: %v", err)
	}

	proto, err := serializeIntegration(registry, integration, []models.CanvasNodeReference{})
	if err != nil {
		return nil, err
	}

	return &pb.UpdateIntegrationSecretResponse{
		Integration: proto,
	}, nil
}

func findSecret(integration *models.Integration, secretName string) (*models.IntegrationSecret, error) {
	secrets, err := models.ListIntegrationSecrets(integration.ID)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		if secret.Name == secretName {
			return &secret, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "secret %s not found", secretName)
}
