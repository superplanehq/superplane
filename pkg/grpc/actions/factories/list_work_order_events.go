package factories

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
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
	cursor := getWorkOrderEventsCursor(req.GetBefore(), req.GetBeforeId())

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	order, err := factory.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	events, err := order.ListEvents(db, int(limit), cursor)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

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
		HasNextPage:   hasWorkOrderEventsNextPage(len(events), int(limit), totalCount),
		LastTimestamp: lastWorkOrderEventTimestamp(events),
		LastId:        lastWorkOrderEventID(events),
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

// getWorkOrderEventsCursor builds the keyset cursor for the listing. beforeID
// is optional: without it the cursor cannot cross rows sharing a created_at,
// which is the behavior clients had before before_id existed.
func getWorkOrderEventsCursor(before *timestamppb.Timestamp, beforeID string) *models.KeysetCursor {
	if before == nil {
		return nil
	}

	t := before.AsTime()
	cursor := &models.KeysetCursor{CreatedAt: &t}
	if id, err := uuid.Parse(beforeID); err == nil {
		cursor.ID = id
	}

	return cursor
}

func hasWorkOrderEventsNextPage(numResults, limit int, totalCount int64) bool {
	return int64(numResults) >= int64(limit) && int64(numResults) < totalCount
}

// lastWorkOrderEventID returns the id of the last row of the page, which the
// client echoes back as before_id on the next request.
func lastWorkOrderEventID(events []models.FactoryWorkOrderEvent) string {
	if len(events) == 0 {
		return ""
	}

	return events[len(events)-1].ID.String()
}

func lastWorkOrderEventTimestamp(events []models.FactoryWorkOrderEvent) *timestamppb.Timestamp {
	if len(events) == 0 {
		return nil
	}

	last := events[len(events)-1]
	timestamp := last.CreatedAt
	return timestamppb.New(timestamp)
}
