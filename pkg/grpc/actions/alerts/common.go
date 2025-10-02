package alerts

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serializeAlerts(alerts []models.Alert) ([]*pb.Alert, error) {
	if len(alerts) == 0 {
		return []*pb.Alert{}, nil
	}

	result := make([]*pb.Alert, len(alerts))
	for i, alert := range alerts {
		result[i] = serializeAlert(&alert)
	}

	return result, nil
}

func serializeAlert(alert *models.Alert) *pb.Alert {
	pbAlert := &pb.Alert{
		Id:           alert.ID.String(),
		Type:         alertTypeToProto(alert.Type),
		Message:      alert.Message,
		SourceId:     alert.SourceID.String(),
		SourceType:   eventSourceTypeToProto(alert.SourceType),
		Acknowledged: alert.Acknowledged,
	}

	if alert.AcknowledgedAt != nil {
		pbAlert.AcknowledgedAt = timestamppb.New(*alert.AcknowledgedAt)
	}

	if alert.CreatedAt != nil {
		pbAlert.CreatedAt = timestamppb.New(*alert.CreatedAt)
	}

	return pbAlert
}

func alertTypeToProto(alertType string) pb.Alert_AlertType {
	switch alertType {
	case models.AlertTypeError:
		return pb.Alert_ALERT_TYPE_ERROR
	case models.AlertTypeWarning:
		return pb.Alert_ALERT_TYPE_WARNING
	case models.AlertTypeInfo:
		return pb.Alert_ALERT_TYPE_INFO
	default:
		return pb.Alert_ALERT_TYPE_UNKNOWN
	}
}

func eventSourceTypeToProto(sourceType string) pb.EventSourceType {
	switch sourceType {
	case models.SourceTypeEventSource:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE
	case models.SourceTypeStage:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_STAGE
	case models.SourceTypeConnectionGroup:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_CONNECTION_GROUP
	default:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN
	}
}
