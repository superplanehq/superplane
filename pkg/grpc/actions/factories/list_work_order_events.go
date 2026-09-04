package factories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultWorkOrderEventsLimit = 50
	maxWorkOrderEventsLimit     = 200
)

func ListWorkOrderEvents(ctx context.Context, organizationID string, req *pb.ListWorkOrderEventsRequest) (*pb.ListWorkOrderEventsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	limit := getWorkOrderEventsLimit(req.GetLimit())
	before := getWorkOrderEventsBefore(req.GetBefore())

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	order, err := factory.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	events, err := order.ListEvents(db, int(limit)+1, before)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	events, hasNext := trimWorkOrderEventsPage(events, int(limit))

	totalCount, err := order.CountEvents(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	serialized, err := serializeWorkOrderEvents(events)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	return &pb.ListWorkOrderEventsResponse{
		Events:        serialized,
		TotalCount:    uint32(totalCount),
		HasNextPage:   hasNext,
		LastTimestamp: lastWorkOrderEventTimestamp(events),
	}, nil
}

func serializeWorkOrderEvents(events []models.FactoryWorkOrderEvent) ([]*pb.WorkOrderEvent, error) {
	result := make([]*pb.WorkOrderEvent, 0, len(events))

	for _, event := range events {
		serialized, err := serializeWorkOrderEvent(event)
		if err != nil {
			return nil, err
		}
		result = append(result, serialized)
	}

	return result, nil
}

func serializeWorkOrderEvent(event models.FactoryWorkOrderEvent) (*pb.WorkOrderEvent, error) {
	var payload map[string]any
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return nil, err
	}

	s, err := structpb.NewStruct(payload)
	if err != nil {
		return nil, err
	}

	return &pb.WorkOrderEvent{
		Timestamp: timestamppb.New(event.CreatedAt),
		Type:      event.Type,
		Event:     s,
	}, nil
}

func getWorkOrderEventsLimit(limit uint32) uint32 {
	if limit <= 0 {
		return defaultWorkOrderEventsLimit
	}

	if limit > maxWorkOrderEventsLimit {
		return maxWorkOrderEventsLimit
	}

	return limit
}

func getWorkOrderEventsBefore(before *timestamppb.Timestamp) *time.Time {
	if before == nil {
		return nil
	}

	t := before.AsTime()
	return &t
}

// trimWorkOrderEventsPage drops the extra row fetched alongside the page
// (limit+1) and reports whether more events follow it. Events are paged by
// timestamp cursor while the total counts every event, so a full page is not
// on its own evidence that anything remains after the cursor.
func trimWorkOrderEventsPage(events []models.FactoryWorkOrderEvent, limit int) ([]models.FactoryWorkOrderEvent, bool) {
	if limit > 0 && len(events) > limit {
		return events[:limit], true
	}

	return events, false
}

func lastWorkOrderEventTimestamp(events []models.FactoryWorkOrderEvent) *timestamppb.Timestamp {
	if len(events) == 0 {
		return nil
	}

	last := events[len(events)-1]
	timestamp := last.CreatedAt
	return timestamppb.New(timestamp)
}
