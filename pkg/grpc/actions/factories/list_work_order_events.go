package factories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
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

	events, err := order.ListEvents(db, int(limit), before)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	totalCount, err := order.CountEvents(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	// Only comment events can carry reactions; a single grouped query
	// covers every comment on the page instead of one query per comment.
	reactionsByComment, err := loadCommentReactionSummaries(ctx, db, orderID, events)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	serialized, err := serializeWorkOrderEvents(events, reactionsByComment)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	return &pb.ListWorkOrderEventsResponse{
		Events:        serialized,
		TotalCount:    uint32(totalCount),
		HasNextPage:   hasWorkOrderEventsNextPage(len(events), int(limit), totalCount),
		LastTimestamp: lastWorkOrderEventTimestamp(events),
	}, nil
}

func loadCommentReactionSummaries(
	ctx context.Context,
	db *gorm.DB,
	orderID uuid.UUID,
	events []models.FactoryWorkOrderEvent,
) (map[uuid.UUID][]models.CommentReactionSummary, error) {
	commentIDs := make([]uuid.UUID, 0, len(events))
	for _, event := range events {
		if event.Type == factory.EventTypeOrderCommentAdded {
			commentIDs = append(commentIDs, event.ID)
		}
	}

	if len(commentIDs) == 0 {
		return map[uuid.UUID][]models.CommentReactionSummary{}, nil
	}

	// The caller's own reactions determine `reacted_by_me`; an
	// unauthenticated/system caller (shouldn't normally happen behind the
	// gateway) simply sees no reaction as "mine".
	var userID uuid.UUID
	if userIDStr, ok := authentication.GetUserIdFromMetadata(ctx); ok {
		if parsed, err := uuid.Parse(userIDStr); err == nil {
			userID = parsed
		}
	}

	return models.ListCommentReactionSummaries(db, orderID, commentIDs, userID)
}

func serializeWorkOrderEvents(
	events []models.FactoryWorkOrderEvent,
	reactionsByComment map[uuid.UUID][]models.CommentReactionSummary,
) ([]*pb.WorkOrderEvent, error) {
	result := make([]*pb.WorkOrderEvent, 0, len(events))

	for _, event := range events {
		serialized, err := serializeWorkOrderEvent(event, reactionsByComment[event.ID])
		if err != nil {
			return nil, err
		}
		result = append(result, serialized)
	}

	return result, nil
}

func serializeWorkOrderEvent(event models.FactoryWorkOrderEvent, reactions []models.CommentReactionSummary) (*pb.WorkOrderEvent, error) {
	var payload map[string]any
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return nil, err
	}

	if event.Type == factory.EventTypeOrderCommentAdded {
		payload["reactions"] = commentReactionSummariesToPayload(reactions)
	}

	s, err := structpb.NewStruct(payload)
	if err != nil {
		return nil, err
	}

	return &pb.WorkOrderEvent{
		Id:        event.ID.String(),
		Timestamp: timestamppb.New(event.CreatedAt),
		Type:      event.Type,
		Event:     s,
	}, nil
}

// commentReactionSummariesToPayload always returns a (possibly empty)
// slice, never nil, so the JSON payload has a stable `reactions: []`
// shape the frontend can rely on regardless of whether the comment has
// any reactions yet.
func commentReactionSummariesToPayload(reactions []models.CommentReactionSummary) []any {
	result := make([]any, 0, len(reactions))
	for _, r := range reactions {
		result = append(result, map[string]any{
			"emoji":       r.Emoji,
			"count":       r.Count,
			"reactedByMe": r.ReactedByMe,
		})
	}
	return result
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

func hasWorkOrderEventsNextPage(numResults, limit int, totalCount int64) bool {
	return int64(numResults) >= int64(limit) && int64(numResults) < totalCount
}

func lastWorkOrderEventTimestamp(events []models.FactoryWorkOrderEvent) *timestamppb.Timestamp {
	if len(events) == 0 {
		return nil
	}

	last := events[len(events)-1]
	timestamp := last.CreatedAt
	return timestamppb.New(timestamp)
}
