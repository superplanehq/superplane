package alerts

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	MinLimit     = 1
	MaxLimit     = 100
	DefaultLimit = 50
)

func ListAlerts(ctx context.Context, canvasID string, includeAcknowledged bool, before *timestamppb.Timestamp, limit *uint32) (*pb.ListAlertsResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas ID: %w", err)
	}

	var beforeTime *time.Time
	if before != nil && before.IsValid() {
		t := before.AsTime()
		beforeTime = &t
	}

	normalizedLimit := getLimit(limit)
	alerts, err := models.ListAlerts(canvasUUID, includeAcknowledged, beforeTime, &normalizedLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts for canvas: %w", err)
	}

	serialized, err := serializeAlerts(alerts)
	if err != nil {
		return nil, err
	}

	response := &pb.ListAlertsResponse{
		Alerts: serialized,
	}

	return response, nil
}

func getLimit(limit *uint32) uint32 {
	if limit == nil || *limit == 0 {
		return DefaultLimit
	}

	if *limit > MaxLimit {
		return MaxLimit
	}

	if *limit < MinLimit {
		return MinLimit
	}

	return *limit
}
