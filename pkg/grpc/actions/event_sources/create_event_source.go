package eventsources

import (
	"context"
	"errors"
	"fmt"

	uuid "github.com/google/uuid"
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

func CreateEventSource(ctx context.Context, encryptor crypto.Encryptor, registry *registry.Registry, orgID, canvasID string, newSource *pb.EventSource) (*pb.CreateEventSourceResponse, error) {
	canvas, err := models.FindCanvasByID(canvasID, uuid.MustParse(orgID))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	logger := logging.ForCanvas(canvas)

	//
	// Validate request
	//
	if newSource == nil || newSource.Metadata == nil || newSource.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "event source name is required")
	}

	if newSource.Spec == nil || newSource.Spec.Type == pb.EventSource_TYPE_UNKNOWN {
		return nil, status.Error(codes.InvalidArgument, "event source type is required")
	}

	eventSourceType, err := actions.ProtoToEventSourceTypeSpec(newSource.Spec.Type)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event source type")
	}

	// Validate event source type constraints
	err = validateEventSourceType(eventSourceType, newSource.Spec)
	if err != nil {
		return nil, err
	}

	//
	// Validate integration if required by event source type.
	//
	var integration *models.Integration
	if newSource.Spec != nil && newSource.Spec.Integration != nil {
		integration, err = actions.ValidateIntegration(canvas, newSource.Spec.Integration)
		if err != nil {
			return nil, err
		}
	}

	//
	// If integration is defined, find the integration resource we are interested in.
	//
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

	schedule, err := validateSchedule(newSource.Spec, integration)
	if err != nil {
		return nil, err
	}

	//
	// Create the event source
	//
	eventSource, plainKey, err := builders.NewEventSourceBuilder(encryptor, registry).
		InCanvas(canvas.ID).
		WithName(newSource.Metadata.Name).
		WithDescription(newSource.Metadata.Description).
		WithScope(models.EventSourceScopeExternal).
		WithType(eventSourceType).
		ForIntegration(integration).
		ForResource(resource).
		WithEventTypes(eventTypes).
		WithSchedule(schedule).
		Create()

	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		if errors.Is(err, builders.ErrResourceAlreadyUsed) {
			return nil, status.Errorf(codes.InvalidArgument, "event source for %s %s already exists", resource.Type(), resource.Name())
		}

		log.Errorf("Error creating event source in canvas %s. Event source: %v. Error: %v", canvasID, newSource, err)
		return nil, err
	}

	protoSource, err := serializeEventSource(*eventSource, nil)
	if err != nil {
		return nil, err
	}

	response := &pb.CreateEventSourceResponse{
		EventSource: protoSource,
	}

	// Only return keys for webhook event sources
	if eventSourceType == models.EventSourceTypeWebhook {
		response.Key = string(plainKey)
	}

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

func validateSchedule(spec *pb.EventSource_Spec, integration *models.Integration) (*models.Schedule, error) {
	if spec == nil || spec.Schedule == nil {
		return nil, nil
	}

	if integration != nil {
		return nil, status.Error(codes.InvalidArgument, "schedules are not supported for event sources with integrations")
	}

	scheduleType, err := actions.ProtoToScheduleType(spec.Schedule.Type)
	if err != nil {
		return nil, err
	}

	schedule := &models.Schedule{
		Type: scheduleType,
	}

	switch spec.Schedule.Type {
	case pb.EventSource_Schedule_TYPE_HOURLY:
		if spec.Schedule.Hourly == nil {
			return nil, status.Error(codes.InvalidArgument, "hourly schedule configuration is required")
		}
		minute := int(spec.Schedule.Hourly.Minute)
		if minute < 0 || minute > 59 {
			return nil, status.Error(codes.InvalidArgument, "minute must be between 0 and 59")
		}
		schedule.Hourly = &models.HourlySchedule{
			Minute: minute,
		}
	case pb.EventSource_Schedule_TYPE_DAILY:
		if spec.Schedule.Daily == nil {
			return nil, status.Error(codes.InvalidArgument, "daily schedule configuration is required")
		}
		if err := actions.ValidateTime(spec.Schedule.Daily.Time); err != nil {
			return nil, err
		}
		schedule.Daily = &models.DailySchedule{
			Time: spec.Schedule.Daily.Time,
		}
	case pb.EventSource_Schedule_TYPE_WEEKLY:
		if spec.Schedule.Weekly == nil {
			return nil, status.Error(codes.InvalidArgument, "weekly schedule configuration is required")
		}
		weekDay, err := actions.ProtoToWeekDay(spec.Schedule.Weekly.WeekDay)
		if err != nil {
			return nil, err
		}
		if err := actions.ValidateTime(spec.Schedule.Weekly.Time); err != nil {
			return nil, err
		}
		schedule.Weekly = &models.WeeklySchedule{
			WeekDay: weekDay,
			Time:    spec.Schedule.Weekly.Time,
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid schedule type")
	}

	return schedule, nil
}

func serializeEventSource(eventSource models.EventSource, lastEvent *models.Event) (*pb.EventSource, error) {
	spec := &pb.EventSource_Spec{
		Type:   actions.EventSourceTypeSpecToProto(eventSource.Type),
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

	//
	// Serialize schedule
	//
	if eventSource.Schedule != nil {
		scheduleData := eventSource.Schedule.Data()
		schedule := &pb.EventSource_Schedule{
			Type: actions.ScheduleTypeToProto(scheduleData.Type),
		}

		switch scheduleData.Type {
		case models.ScheduleTypeHourly:
			if scheduleData.Hourly != nil {
				schedule.Hourly = &pb.EventSource_HourlySchedule{
					Minute: int32(scheduleData.Hourly.Minute),
				}
			}
		case models.ScheduleTypeDaily:
			if scheduleData.Daily != nil {
				schedule.Daily = &pb.EventSource_DailySchedule{
					Time: scheduleData.Daily.Time,
				}
			}
		case models.ScheduleTypeWeekly:
			if scheduleData.Weekly != nil {
				schedule.Weekly = &pb.EventSource_WeeklySchedule{
					WeekDay: actions.WeekDayToProto(scheduleData.Weekly.WeekDay),
					Time:    scheduleData.Weekly.Time,
				}
			}
		}

		spec.Schedule = schedule
	}

	pbEventSource := &pb.EventSource{
		Metadata: &pb.EventSource_Metadata{
			Id:          eventSource.ID.String(),
			Name:        eventSource.Name,
			Description: eventSource.Description,
			CanvasId:    eventSource.CanvasID.String(),
			CreatedAt:   timestamppb.New(*eventSource.CreatedAt),
			UpdatedAt:   timestamppb.New(*eventSource.UpdatedAt),
		},
		Spec:   spec,
		Status: &pb.EventSource_Status{},
	}

	// Add last event information if provided
	if lastEvent != nil {
		pbEvent, err := actions.SerializeEvent(*lastEvent)
		if err != nil {
			return nil, err
		}
		pbEventSource.Status.LastEvent = pbEvent
	}

	// Add schedule information if the event source has a schedule
	if eventSource.Schedule != nil {
		pbEventSource.Status.Schedule = &pb.EventSource_Status_Schedule{}
		if eventSource.LastTriggeredAt != nil {
			pbEventSource.Status.Schedule.LastTrigger = timestamppb.New(*eventSource.LastTriggeredAt)
		}

		if eventSource.NextTriggerAt != nil {
			pbEventSource.Status.Schedule.NextTrigger = timestamppb.New(*eventSource.NextTriggerAt)
		}
	}

	return pbEventSource, nil
}

func validateEventSourceType(eventSourceType string, spec *pb.EventSource_Spec) error {
	switch eventSourceType {
	case models.EventSourceTypeIntegrationResource:
		if spec == nil || spec.Integration == nil {
			return status.Error(codes.InvalidArgument, "integration is required for integration-resource event sources")
		}
		if spec.Resource == nil {
			return status.Error(codes.InvalidArgument, "resource is required for integration-resource event sources")
		}
	case models.EventSourceTypeScheduled:
		if spec == nil || spec.Schedule == nil {
			return status.Error(codes.InvalidArgument, "schedule is required for scheduled event sources")
		}
		if spec.Integration != nil {
			return status.Error(codes.InvalidArgument, "scheduled event sources cannot have integrations")
		}
	case models.EventSourceTypeManual, models.EventSourceTypeWebhook:
		if spec != nil && spec.Integration != nil {
			return status.Error(codes.InvalidArgument, "manual and webhook event sources cannot have integrations")
		}
	}
	return nil
}
