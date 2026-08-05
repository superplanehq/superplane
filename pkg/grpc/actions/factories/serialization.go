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

func serializeWorkOrder(order *models.FactoryWorkOrder, executions []models.FactoryWorkOrderExecutionRecord) *pb.WorkOrder {
	return &pb.WorkOrder{
		Id:             order.ID.String(),
		Title:          order.Title,
		Description:    order.Description,
		State:          serializeWorkOrderState(order.State),
		Result:         serializeWorkOrderResult(order.Result),
		CreatedAt:      timestamppb.New(order.CreatedAt),
		UpdatedAt:      timestamppb.New(order.UpdatedAt),
		StateUpdatedAt: timestamppb.New(order.StateUpdatedAt),
		Assignees:      serializeWorkOrderAssignees(order.Assignees),
		Executions:     serializeWorkOrderExecutions(executions),
		CreatedBy:      serializeWorkOrderCreator(order),
	}
}

func serializeWorkOrderCreator(order *models.FactoryWorkOrder) *pb.UserRef {
	if order.CreatedByID == nil {
		return nil
	}

	name := order.CreatedByID.String()
	if order.CreatedBy != nil {
		name = order.CreatedBy.Name
	}

	return &pb.UserRef{
		Id:   order.CreatedByID.String(),
		Name: name,
	}
}

func serializeWorkOrderExecutions(executions []models.FactoryWorkOrderExecutionRecord) []*pb.WorkOrderExecution {
	result := make([]*pb.WorkOrderExecution, 0, len(executions))
	for _, execution := range executions {
		result = append(result, serializeWorkOrderExecution(execution))
	}
	return result
}

func serializeWorkOrderExecution(execution models.FactoryWorkOrderExecutionRecord) *pb.WorkOrderExecution {
	item := &pb.WorkOrderExecution{
		Id:        execution.ID.String(),
		Step:      execution.StepName,
		State:     serializeWorkOrderExecutionState(execution.Status, execution.RunState),
		Result:    serializeWorkOrderExecutionResult(execution.Result, execution.RunResult),
		CreatedAt: timestamppb.New(execution.CreatedAt),
		UpdatedAt: timestamppb.New(execution.UpdatedAt),
		Line: &pb.WorkOrderExecution_LineRef{
			Id:   execution.LineID.String(),
			Name: execution.LineName,
		},
		Run: &pb.WorkOrderExecution_RunRef{
			Id:      execution.RunID.String(),
			AppId:   execution.CanvasID.String(),
			AppName: execution.CanvasName,
		},
	}
	return item
}

func serializeWorkOrderExecutionState(status, runState string) pb.WorkOrderExecution_State {
	if runState == models.CanvasRunStateCancelling {
		return pb.WorkOrderExecution_STATE_CANCELLING
	}

	switch status {
	case models.FactoryWorkOrderExecutionStatusPending:
		return pb.WorkOrderExecution_STATE_PENDING
	case models.FactoryWorkOrderExecutionStatusRunning:
		return pb.WorkOrderExecution_STATE_STARTED
	case models.FactoryWorkOrderExecutionStatusFinished:
		return pb.WorkOrderExecution_STATE_FINISHED
	default:
		return pb.WorkOrderExecution_STATE_UNKNOWN
	}
}

func serializeWorkOrderExecutionResult(executionResult, runResult string) pb.WorkOrderExecution_Result {
	result := executionResult
	if result == "" {
		result = runResult
	}

	switch result {
	case models.CanvasRunResultPassed:
		return pb.WorkOrderExecution_RESULT_PASSED
	case models.CanvasRunResultFailed:
		return pb.WorkOrderExecution_RESULT_FAILED
	case models.CanvasRunResultCancelled:
		return pb.WorkOrderExecution_RESULT_CANCELLED
	default:
		return pb.WorkOrderExecution_RESULT_UNKNOWN
	}
}

func serializeWorkOrderState(state string) pb.WorkOrder_State {
	switch state {
	case models.FactoryWorkOrderStateDraft:
		return pb.WorkOrder_STATE_DRAFT
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
	case models.FactoryWorkOrderResultFailed:
		return pb.WorkOrder_RESULT_FAILED
	default:
		return pb.WorkOrder_RESULT_UNSPECIFIED
	}
}

func serializeWorkOrderAssignees(assignees []models.FactoryWorkOrderAssignee) []*pb.UserRef {
	result := make([]*pb.UserRef, 0, len(assignees))
	for _, assignee := range assignees {
		name := assignee.UserID.String()
		if assignee.User != nil {
			name = assignee.User.Name
		}
		result = append(result, &pb.UserRef{
			Id:   assignee.UserID.String(),
			Name: name,
		})
	}
	return result
}
