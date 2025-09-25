package integrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateIntegration(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	domainType, domainID, idOrName string,
	spec *pb.Integration,
) (*pb.UpdateIntegrationResponse, error) {
	_, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if spec == nil || spec.Metadata == nil || spec.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "integration name is required")
	}

	integration, err := findIntegration(domainType, uuid.MustParse(domainID), idOrName)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	resourceCount, err := models.CountResourcesByIntegration(integration.ID)
	if err != nil {
		log.Errorf("Error checking resources for integration. Error: %v", err)
		return nil, status.Error(codes.Internal, "failed to check integration resources")
	}

	if resourceCount > 0 {
		return nil, status.Error(codes.FailedPrecondition, "integration cannot be updated as it is being used by existing resources")
	}

	updatedIntegration, err := buildIntegration(ctx, encryptor, registry, domainType, uuid.MustParse(domainID), spec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	integration.Name = updatedIntegration.Name
	integration.Type = updatedIntegration.Type
	integration.URL = updatedIntegration.URL
	integration.AuthType = updatedIntegration.AuthType
	integration.Auth = updatedIntegration.Auth

	err = integration.Update()
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error updating integration. Error: %v", err)
		return nil, err
	}

	response := &pb.UpdateIntegrationResponse{
		Integration: serializeIntegration(*integration),
	}

	return response, nil
}

func findIntegration(domainType string, domainID uuid.UUID, idOrName string) (*models.Integration, error) {
	err := actions.ValidateUUIDs(idOrName)
	var integration *models.Integration
	if err != nil {
		integration, err = models.FindIntegrationByName(domainType, domainID, idOrName)
	} else {
		integration, err = models.FindDomainIntegration(domainType, domainID, uuid.MustParse(idOrName))
	}

	if err != nil {
		return nil, fmt.Errorf("integration %s not found", idOrName)
	}
	return integration, nil
}
