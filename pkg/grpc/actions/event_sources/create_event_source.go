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
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateEventSource(ctx context.Context, encryptor crypto.Encryptor, registry *registry.Registry, canvasID string, req *pb.CreateEventSourceRequest) (*pb.CreateEventSourceResponse, error) {
	canvas, err := models.FindCanvasByIDOnly(canvasID)
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
		resource, err = actions.ValidateResource(ctx, registry, integration, req.EventSource.Spec.Resource)
		if err != nil {
			return nil, err
		}
	}

	eventTypes, err := validateEventTypes(req.EventSource.Spec)
	if err != nil {
		return nil, err
	}

	//
	// Create the event source
	//
	eventSource, plainKey, err := builders.NewEventSourceBuilder(encryptor).
		InCanvas(canvas.ID).
		WithName(req.EventSource.Metadata.Name).
		WithDescription(req.EventSource.Metadata.Description).
		WithScope(models.EventSourceScopeExternal).
		ForIntegration(integration).
		ForResource(resource).
		WithEventTypes(eventTypes).
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

func validateEventTypes(spec *pb.EventSource_Spec) ([]models.EventType, error) {
	if spec == nil || spec.Events == nil {
		return []models.EventType{}, nil
	}

	out := []models.EventType{}
	for _, i := range spec.Events {
		filters, err := actions.ValidateFilters(i.Filters)
		if err != nil {
			return nil, err
		}

		out = append(out, models.EventType{
			Type:           i.Type,
			Filters:        filters,
			FilterOperator: actions.ProtoToFilterOperator(i.FilterOperator),
		})
	}

	return out, nil
}

func serializeEventSource(eventSource models.EventSource) (*pb.EventSource, error) {
	spec := &pb.EventSource_Spec{
		Events: []*pb.EventSource_EventType{},
	}

	//
	// Serialize event types
	//
	for _, eventType := range eventSource.EventTypes {
		filters, err := actions.SerializeFilters(eventType.Filters)
		if err != nil {
			return nil, err
		}

		spec.Events = append(spec.Events, &pb.EventSource_EventType{
			Type:           eventType.Type,
			Filters:        filters,
			FilterOperator: actions.FilterOperatorToProto(eventType.FilterOperator),
		})
	}

	//
	// Serialize integration and resource
	//
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

		spec.Resource = &integrationPb.ResourceRef{
			Type: resource.Type(),
			Name: resource.Name(),
		}
	}

	return &pb.EventSource{
		Metadata: &pb.EventSource_Metadata{
			Id:          eventSource.ID.String(),
			Name:        eventSource.Name,
			Description: eventSource.Description,
			CanvasId:    eventSource.CanvasID.String(),
			CreatedAt:   timestamppb.New(*eventSource.CreatedAt),
			UpdatedAt:   timestamppb.New(*eventSource.UpdatedAt),
		},
		Spec: spec,
	}, nil
}
