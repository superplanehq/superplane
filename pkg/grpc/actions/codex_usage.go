package actions

import (
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	canvasespb "github.com/superplanehq/superplane/pkg/protos/canvases"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeCodexUsageLimit(limit models.CodexUsageLimit) *pb.CodexUsageLimit {
	view := &pb.CodexUsageLimit{
		LimitId:   limit.LimitID,
		LimitName: limit.LimitName,
		PlanType:  limit.PlanType,
		Primary:   SerializeCodexUsageWindow(limit.Primary),
		Secondary: SerializeCodexUsageWindow(limit.Secondary),
	}
	if limit.Credits != nil {
		view.Credits = &pb.CodexUsageCredits{
			HasCredits: limit.Credits.HasCredits,
			Unlimited:  limit.Credits.Unlimited,
			Balance:    limit.Credits.Balance,
		}
	}
	return view
}

func SerializeCodexUsageWindow(window *models.CodexUsageWindow) *pb.CodexUsageWindow {
	if window == nil || window.UsedPercent == nil {
		return nil
	}
	view := &pb.CodexUsageWindow{
		UsedPercent:        *window.UsedPercent,
		WindowDurationMins: window.WindowDurationMins,
	}
	if window.ResetsAt != nil {
		view.ResetsAt = timestamppb.New(time.Unix(*window.ResetsAt, 0))
	}
	return view
}

func SerializeSubscriptionFailure(failure *models.SubscriptionFailure) *canvasespb.SubscriptionFailure {
	if failure == nil {
		return nil
	}
	out := &canvasespb.SubscriptionFailure{Reason: failure.Reason, LimitId: failure.LimitID, SubscriptionOwnerId: failure.SubscriptionOwnerID}
	if failure.ResetsAt != nil {
		out.ResetsAt = timestamppb.New(*failure.ResetsAt)
	}
	return out
}
