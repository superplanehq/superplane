package eventsources

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
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

	resourceId, err := validateIntegrationResource(ctx, encryptor, canvas, req.EventSource.Spec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	//
	// Create new source
	//
	plainKey, encryptedKey, err := genNewEventSourceKey(ctx, encryptor, req.EventSource.Metadata.Name)
	if err != nil {
		logger.Errorf("Error generating event source key. Request: %v. Error: %v", req, err)
		return nil, status.Error(codes.Internal, "error generating key")
	}

	eventSource, err := canvas.CreateEventSource(req.EventSource.Metadata.Name, encryptedKey, resourceId)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating event source. Request: %v. Error: %v", req, err)
		return nil, err
	}

	response := &pb.CreateEventSourceResponse{
		EventSource: serializeEventSource(*eventSource),
		Key:         string(plainKey),
	}

	logger.Infof("Created event source. Request: %v", req)

	err = messages.NewEventSourceCreatedMessage(eventSource).Publish()

	if err != nil {
		logger.Errorf("failed to publish event source created message: %v", err)
	}

	return response, nil
}

func validateIntegrationResource(ctx context.Context, encryptor crypto.Encryptor, canvas *models.Canvas, spec *pb.EventSource_Spec) (*uuid.UUID, error) {
	//
	// It is OK to have an event source without an integration
	//
	if spec == nil {
		return nil, nil
	}

	if spec.Integration == nil {
		return nil, nil
	}

	if spec.Integration.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "integration name is required")
	}

	if spec.Integration.Resource == nil {
		return nil, status.Error(codes.InvalidArgument, "integration resource is required")
	}

	// TODO: support for canvas level integration
	record, err := models.FindIntegrationByName(authorization.DomainCanvas, canvas.ID, spec.Integration.Name)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %v", err)
	}

	integration, err := integrations.NewIntegration(ctx, record, encryptor)
	if err != nil {
		return nil, fmt.Errorf("error building integration: %v", err)
	}

	resourceRef := spec.Integration.Resource
	resourceType := protoToResourceType(resourceRef.Type)
	resource, err := integration.GetResource(resourceType, resourceRef.Name)
	if err != nil {
		return nil, fmt.Errorf("%s %s not found: %v", resourceType, resourceRef.Name, err)
	}

	// TODO: resource IDs can be of other types
	resourceID := uuid.MustParse(resource.ID())
	_, err = record.CreateResource(resourceType, resourceID, resourceRef.Name)
	if err != nil {
		return nil, fmt.Errorf("error creating integration resource: %v", err)
	}

	return &resourceID, nil
}

func serializeEventSource(eventSource models.EventSource) *pb.EventSource {
	return &pb.EventSource{
		Metadata: &pb.EventSource_Metadata{
			Id:        eventSource.ID.String(),
			Name:      eventSource.Name,
			CanvasId:  eventSource.CanvasID.String(),
			CreatedAt: timestamppb.New(*eventSource.CreatedAt),
			UpdatedAt: timestamppb.New(*eventSource.UpdatedAt),
		},
		Spec: &pb.EventSource_Spec{},
	}
}

func genNewEventSourceKey(ctx context.Context, encryptor crypto.Encryptor, name string) (string, []byte, error) {
	plainKey, _ := crypto.Base64String(32)
	encrypted, err := encryptor.Encrypt(ctx, []byte(plainKey), []byte(name))
	if err != nil {
		return "", nil, err
	}

	return plainKey, encrypted, nil
}

func protoToResourceType(resourceType pb.IntegrationResource_Type) string {
	switch resourceType {
	case pb.IntegrationResource_TYPE_PROJECT:
		return integrations.ResourceTypeProject
	case pb.IntegrationResource_TYPE_TASK:
		return integrations.ResourceTypeTask
	case pb.IntegrationResource_TYPE_REPOSITORY:
		return integrations.ResourceTypeRepository
	default:
		return ""
	}
}
