package eventsources

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateEventSource(ctx context.Context, encryptor crypto.Encryptor, registry *registry.Registry, orgID, canvasID, idOrName string, newSource *pb.EventSource) (*pb.UpdateEventSourceResponse, error) {
	_, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvas, err := models.FindCanvasByID(canvasID, uuid.MustParse(orgID))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = actions.ValidateUUIDs(idOrName)
	var eventSource *models.EventSource
	if err != nil {
		eventSource, err = models.FindEventSourceByName(canvasID, idOrName)
	} else {
		eventSource, err = models.FindEventSource(uuid.MustParse(idOrName))
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "event source not found")
		}
		return nil, err
	}

	if newSource == nil || newSource.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "event source metadata is required")
	}

	if newSource.Metadata.Name != "" && newSource.Metadata.Name != eventSource.Name {
		_, err := models.FindEventSourceByName(canvasID, newSource.Metadata.Name)
		if err == nil {
			return nil, status.Error(codes.InvalidArgument, "event source name already in use")
		}
		eventSource.Name = newSource.Metadata.Name
	}

	if newSource.Metadata.Description != "" {
		eventSource.Description = newSource.Metadata.Description
	}

	var integration *models.Integration
	if newSource.Spec != nil && newSource.Spec.Integration != nil {
		integration, err = actions.ValidateIntegration(canvas, newSource.Spec.Integration)
		if err != nil {
			return nil, err
		}
	}

	var resource integrations.Resource
	if integration != nil {
		resource, err = actions.ValidateResource(ctx, registry, integration, newSource.Spec.Resource)
		if err != nil {
			return nil, err
		}
	}

	eventTypes, err := validateEventTypes(newSource.Spec)
	if err != nil {
		return nil, err
	}

	eventSource, plainKey, err := builders.NewEventSourceBuilder(encryptor, registry).
		WithContext(ctx).
		WithExistingEventSource(eventSource).
		WithName(eventSource.Name).
		WithDescription(eventSource.Description).
		WithScope(models.EventSourceScopeExternal).
		ForIntegration(integration).
		ForResource(resource).
		WithEventTypes(eventTypes).
		Update()

	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, builders.ErrResourceAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, "event source for this resource already exists")
		}
		return nil, err
	}

	serialized, err := serializeEventSource(*eventSource)
	if err != nil {
		return nil, err
	}

	response := &pb.UpdateEventSourceResponse{
		EventSource: serialized,
		Key:         plainKey,
	}

	return response, nil
}
