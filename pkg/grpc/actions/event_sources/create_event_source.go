package eventsources

import (
	"context"
	"errors"
	"fmt"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func CreateEventSource(ctx context.Context, encryptor crypto.Encryptor, req *pb.CreateEventSourceRequest) (*pb.CreateEventSourceResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasIdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	logger := logging.ForCanvas(canvas)

	//
	// Validate request
	//
	if req.EventSource == nil || req.EventSource.Metadata == nil || req.EventSource.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "event source name is required")
	}

	integration, resource, err := validateIntegrationResource(ctx, encryptor, canvas, req.EventSource.Spec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	//
	// Create new source
	//
	plainKey, encryptedKey, err := crypto.NewRandomKey(ctx, encryptor, req.EventSource.Metadata.Name)
	if err != nil {
		logger.Errorf("Error generating event source key. Request: %v. Error: %v", req, err)
		return nil, status.Error(codes.Internal, "error generating key")
	}

	//
	// Create the event source
	//
	var eventSource *models.EventSource
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var resourceID *uuid.UUID
		if integration != nil && resource != nil {
			r, err := integration.CreateResourceInTransaction(tx, resource.Type(), resource.Id(), resource.Name())
			if err != nil {
				return err
			}

			resourceID = &r.ID
		}

		eventSource, err = canvas.CreateEventSourceInTransaction(tx, req.EventSource.Metadata.Name, encryptedKey, models.EventSourceScopeExternal, resourceID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating event source. Request: %v. Error: %v", req, err)
		return nil, err
	}

	protoSource, err := serializeEventSource(*eventSource)
	if err != nil {
		return nil, err
	}

	response := &pb.CreateEventSourceResponse{
		EventSource: protoSource,
		Key:         string(plainKey),
	}

	logger.Infof("Created event source. Request: %v", req)

	err = messages.NewEventSourceCreatedMessage(eventSource).Publish()

	if err != nil {
		logger.Errorf("failed to publish event source created message: %v", err)
	}

	return response, nil
}

func validateIntegrationResource(ctx context.Context, encryptor crypto.Encryptor, canvas *models.Canvas, spec *pb.EventSource_Spec) (*models.Integration, integrations.Resource, error) {
	//
	// It is OK to have an event source without an integration
	//
	if spec == nil {
		return nil, nil, nil
	}

	integrationRecord, err := validateIntegration(canvas, spec.Integration)
	if err != nil {
		return nil, nil, err
	}

	resourceType, resourceName, err := getResourceTypeAndName(integrationRecord, spec)
	if err != nil {
		return nil, nil, err
	}

	integration, err := integrations.NewIntegration(ctx, integrationRecord, encryptor)
	if err != nil {
		return nil, nil, fmt.Errorf("error building integration: %v", err)
	}

	resource, err := integration.Get(resourceType, resourceName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s %s not found: %v", resourceType, resourceName, err)
	}

	return integrationRecord, resource, nil
}

func validateIntegration(canvas *models.Canvas, integrationRef *pb.IntegrationRef) (*models.Integration, error) {
	if integrationRef == nil {
		return nil, nil
	}

	if integrationRef.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "integration name is required")
	}

	// TODO: support for organization level integration
	integration, err := models.FindIntegrationByName(authorization.DomainCanvas, canvas.ID, integrationRef.Name)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %v", err)
	}

	return integration, nil
}

func getResourceTypeAndName(integrationRecord *models.Integration, spec *pb.EventSource_Spec) (string, string, error) {
	switch integrationRecord.Type {
	case models.IntegrationTypeSemaphore:
		if spec.Semaphore == nil {
			return "", "", status.Error(codes.InvalidArgument, "missing semaphore resource")
		}

		return integrations.ResourceTypeProject, spec.Semaphore.Project, nil

	default:
		return "", "", status.Error(codes.InvalidArgument, "unsupported integration type")
	}
}

func serializeEventSource(eventSource models.EventSource) (*pb.EventSource, error) {
	spec := &pb.EventSource_Spec{}
	if eventSource.ResourceID != nil {
		resource, err := models.FindResourceByID(*eventSource.ResourceID)
		if err != nil {
			return nil, fmt.Errorf("resource not found: %v", err)
		}

		integration, err := models.FindIntegrationByID(resource.IntegrationID)
		if err != nil {
			return nil, fmt.Errorf("integration not found: %v", err)
		}

		spec.Integration = &pb.IntegrationRef{Name: integration.Name}

		switch integration.Type {
		case models.IntegrationTypeSemaphore:
			spec.Semaphore = &pb.EventSource_Spec_Semaphore{
				Project: resource.ResourceName,
			}
		}
	}

	return &pb.EventSource{
		Metadata: &pb.EventSource_Metadata{
			Id:        eventSource.ID.String(),
			Name:      eventSource.Name,
			CanvasId:  eventSource.CanvasID.String(),
			CreatedAt: timestamppb.New(*eventSource.CreatedAt),
			UpdatedAt: timestamppb.New(*eventSource.UpdatedAt),
		},
		Spec: spec,
	}, nil
}
