package factories

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serializeFactory(factory *models.Factory) *pb.Factory {
	return &pb.Factory{
		Id:          factory.ID.String(),
		Name:        factory.Name,
		Description: factory.Description,
	}
}

func serializeFactoryWithLines(
	factory *models.Factory,
	lines []models.FactoryLine,
) *pb.Factory {
	return &pb.Factory{
		Id:          factory.ID.String(),
		Name:        factory.Name,
		Description: factory.Description,
		Lines:       serializeFactoryLines(lines),
	}
}

func serializeFactoryLines(lines []models.FactoryLine) []*pb.FactoryLine {
	result := make([]*pb.FactoryLine, len(lines))
	for i := range lines {
		result[i] = serializeFactoryLine(&lines[i])
	}
	return result
}

func serializeFactoryApps(canvases []models.Canvas) []*pb.Factory_App {
	result := make([]*pb.Factory_App, len(canvases))
	for i, canvas := range canvases {
		app := &pb.Factory_App{
			Id:          canvas.ID.String(),
			Name:        canvas.Name,
			Description: canvas.Description,
		}
		if canvas.CreatedAt != nil {
			app.CreatedAt = timestamppb.New(*canvas.CreatedAt)
		}
		if canvas.UpdatedAt != nil {
			app.UpdatedAt = timestamppb.New(*canvas.UpdatedAt)
		}
		result[i] = app
	}
	return result
}

func serializeFactoryLine(line *models.FactoryLine) *pb.FactoryLine {
	steps := make([]*pb.FactoryLine_Step, len(line.Steps))
	for i, step := range line.Steps {
		steps[i] = &pb.FactoryLine_Step{
			Name: step.Name,
			Type: step.Type,
			App: &pb.FactoryLine_AppStep{
				App:        step.AppID.String(),
				Entrypoint: step.Entrypoint,
			},
		}
	}

	return &pb.FactoryLine{
		Id:        line.ID.String(),
		Name:      line.Name,
		Steps:     steps,
		CreatedAt: timestamppb.New(line.CreatedAt),
		UpdatedAt: timestamppb.New(line.UpdatedAt),
	}
}

func serializeFactories(factories []models.Factory) []*pb.Factory {
	result := make([]*pb.Factory, len(factories))
	for i := range factories {
		result[i] = serializeFactory(&factories[i])
	}
	return result
}

func serializeWorkOrder(order *models.FactoryWorkOrder) *pb.WorkOrder {
	return &pb.WorkOrder{
		Id:          order.ID.String(),
		Title:       order.Title,
		Description: order.Description,
		State:       serializeWorkOrderState(order.State),
		Result:      serializeWorkOrderResult(order.Result),
		CreatedAt:   timestamppb.New(order.CreatedAt),
		UpdatedAt:   timestamppb.New(order.UpdatedAt),
		Assignees:   serializeWorkOrderAssignees(order.Assignees),
	}
}

func serializeWorkOrders(orders []models.FactoryWorkOrder) []*pb.WorkOrder {
	result := make([]*pb.WorkOrder, len(orders))
	for i := range orders {
		result[i] = serializeWorkOrder(&orders[i])
	}
	return result
}

func serializeWorkOrderState(state string) pb.WorkOrder_State {
	switch state {
	case models.FactoryWorkOrderStateOpen:
		return pb.WorkOrder_STATE_OPEN
	case models.FactoryWorkOrderStateClosed:
		return pb.WorkOrder_STATE_CLOSED
	default:
		return pb.WorkOrder_STATE_UNSPECIFIED
	}
}

func serializeWorkOrderResult(result string) pb.WorkOrder_Result {
	switch result {
	case models.FactoryWorkOrderResultCompleted:
		return pb.WorkOrder_RESULT_COMPLETED
	case models.FactoryWorkOrderResultRejected:
		return pb.WorkOrder_RESULT_REJECTED
	default:
		return pb.WorkOrder_RESULT_UNSPECIFIED
	}
}

func serializeWorkOrderAssignees(assignees []models.FactoryWorkOrderAssignee) []*pb.WorkOrder_Assignee {
	result := make([]*pb.WorkOrder_Assignee, 0, len(assignees))
	for _, assignee := range assignees {
		name := assignee.UserID.String()
		if assignee.User != nil {
			name = assignee.User.Name
		}
		result = append(result, &pb.WorkOrder_Assignee{
			Id:   assignee.UserID.String(),
			Name: name,
		})
	}
	return result
}
