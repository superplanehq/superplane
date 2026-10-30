package actions

import (
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
