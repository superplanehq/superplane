package eventsources

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	integrationPb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	//
	// It is OK to create an event source without an integration.
	//
	var integration *models.Integration
	if req.EventSource.Spec != nil && req.EventSource.Spec.Integration != nil {
		integration, err = actions.ValidateIntegration(canvas, req.EventSource.Spec.Integration)
		if err != nil {
			return nil, err
		}
	}

	//
	// If integration is defined, find the integration resource we are interested in.
	//
	var resource integrations.Resource
	if integration != nil {
		resourceName, err := resourceName(req.EventSource.Spec, integration)
		if err != nil {
			return nil, err
		}

		resource, err = actions.ValidateResource(ctx, encryptor, integration, resourceName)
		if err != nil {
			return nil, err
		}
	}

	//
	// Create the event source
	//
	eventSource, plainKey, err := builders.NewEventSourceBuilder(encryptor).
		InCanvas(canvas).
		WithName(req.EventSource.Metadata.Name).
		WithScope(models.EventSourceScopeExternal).
		ForIntegration(integration).
		ForResource(resource).
		Create()

	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		if errors.Is(err, builders.ErrResourceAlreadyUsed) {
			return nil, status.Errorf(codes.InvalidArgument, "event source for %s %s already exists", resource.Type(), resource.Name())
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

		spec.Integration = &integrationPb.IntegrationRef{
			Name:       integration.Name,
			DomainType: actions.DomainTypeToProto(integration.DomainType),
		}

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

func resourceName(spec *pb.EventSource_Spec, integration *models.Integration) (string, error) {
	switch integration.Type {
	case models.IntegrationTypeSemaphore:
		if spec.Semaphore == nil || spec.Semaphore.Project == "" {
			return "", fmt.Errorf("semaphore project is required")
		}

		return spec.Semaphore.Project, nil
	default:
		return "", fmt.Errorf("integration type %s is not supported", integration.Type)
	}
}
