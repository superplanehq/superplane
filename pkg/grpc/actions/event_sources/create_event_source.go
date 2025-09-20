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

	//
	// It is OK to create an event source without an integration.
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
		Key:         string(plainKey),
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
	case pb.EventSource_Schedule_TYPE_CRON:
		if spec.Schedule.Cron == nil {
			return nil, status.Error(codes.InvalidArgument, "cron schedule configuration is required")
		}
		if err := actions.ValidateCronExpression(spec.Schedule.Cron.Expression); err != nil {
			return nil, err
		}
		schedule.Cron = &models.CronSchedule{
			Expression: spec.Schedule.Cron.Expression,
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid schedule type")
	}

	return schedule, nil
}

func serializeEventSource(eventSource models.EventSource, statusInfo *models.EventSourceStatusInfo) (*pb.EventSource, error) {
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

	//
	// Serialize schedule
	//
	if eventSource.Schedule != nil {
		schedule := &pb.EventSource_Schedule{
			Type: actions.ScheduleTypeToProto(eventSource.Schedule.Type),
		}

		switch eventSource.Schedule.Type {
		case models.ScheduleTypeDaily:
			if eventSource.Schedule.Daily != nil {
				schedule.Daily = &pb.EventSource_DailySchedule{
					Time: eventSource.Schedule.Daily.Time,
				}
			}
		case models.ScheduleTypeWeekly:
			if eventSource.Schedule.Weekly != nil {
				schedule.Weekly = &pb.EventSource_WeeklySchedule{
					WeekDay: actions.WeekDayToProto(eventSource.Schedule.Weekly.WeekDay),
					Time:    eventSource.Schedule.Weekly.Time,
				}
			}
		case models.ScheduleTypeCron:
			if eventSource.Schedule.Cron != nil {
				schedule.Cron = &pb.EventSource_CronSchedule{
					Expression: eventSource.Schedule.Cron.Expression,
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
		Spec: spec,
	}

	if statusInfo != nil {
		status := &pb.EventSource_Status{
			History: &pb.EventSource_Status_History{
				Received:    uint32(statusInfo.ReceivedCount),
				RecentItems: []*pb.Event{},
			},
		}

		for _, event := range statusInfo.RecentEvents {
			pbEvent, err := actions.SerializeEvent(event)
			if err != nil {
				return nil, err
			}
			status.History.RecentItems = append(status.History.RecentItems, pbEvent)
		}

		pbEventSource.Status = status
	}

	return pbEventSource, nil
}
