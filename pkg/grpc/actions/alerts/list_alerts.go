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

const MaxTimespan = 30 * 24 * time.Hour

func ListAlerts(ctx context.Context, canvasID string, includeAcknowledged bool, before *timestamppb.Timestamp) (*pb.ListAlertsResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas ID: %w", err)
	}

	var beforeTime *time.Time
	if before != nil && before.IsValid() {
		t := before.AsTime()
		beforeTime = &t
	}

	maxTime := time.Now().Add(-MaxTimespan)
	if beforeTime == nil || beforeTime.After(maxTime) {
		beforeTime = &maxTime
	}

	alerts, err := models.ListAlerts(canvasUUID, includeAcknowledged, beforeTime)
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
