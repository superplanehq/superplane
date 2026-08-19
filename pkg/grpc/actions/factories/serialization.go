package factories

import (
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serializeFactory(factory *models.Factory) *pb.Factory {
	return &pb.Factory{
		Id:          factory.ID.String(),
		Name:        factory.Name,
		Description: factory.Description,
		Key:         factory.Key,
		Onboarding:  serializeFactoryOnboarding(factory),
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
		Key:         factory.Key,
		Lines:       serializeFactoryLines(lines),
		Onboarding:  serializeFactoryOnboarding(factory),
	}
}

func serializeFactoryOnboarding(factory *models.Factory) *pb.FactoryOnboarding {
	config := factory.OnboardingConfigValue()
	onboarding := &pb.FactoryOnboarding{
		VcsIntegrationId:   config.VCSIntegrationID,
		AgentIntegrationId: config.AgentIntegrationID,
		AppRepository:      config.AppRepository,
		BacklogRepository:  config.BacklogRepository,
		IssuesSource:       serializeFactoryOnboardingIssuesSource(config.IssuesSource),
		AgentHarness:       serializeFactoryOnboardingAgentHarness(config.AgentHarness),
		ProvisionedAppId:   config.ProvisionedAppID,
		ProvisionedLineId:  config.ProvisionedLineID,
	}
	if factory.OnboardingCompletedAt != nil {
		onboarding.CompletedAt = timestamppb.New(*factory.OnboardingCompletedAt)
	}
	return onboarding
}

func serializeFactoryOnboardingIssuesSource(source string) pb.FactoryOnboarding_IssuesSource {
	switch source {
	case models.FactoryOnboardingIssuesSourceVCS:
		return pb.FactoryOnboarding_ISSUES_SOURCE_VCS
	case models.FactoryOnboardingIssuesSourceLinear:
		return pb.FactoryOnboarding_ISSUES_SOURCE_LINEAR
	case models.FactoryOnboardingIssuesSourceJira:
		return pb.FactoryOnboarding_ISSUES_SOURCE_JIRA
	case models.FactoryOnboardingIssuesSourceSkip:
		return pb.FactoryOnboarding_ISSUES_SOURCE_SKIP
	default:
		return pb.FactoryOnboarding_ISSUES_SOURCE_UNSPECIFIED
	}
}

func serializeFactoryOnboardingAgentHarness(harness string) pb.FactoryOnboarding_AgentHarness {
	switch harness {
	case models.FactoryOnboardingAgentHarnessClaudeCode:
		return pb.FactoryOnboarding_AGENT_HARNESS_CLAUDE_CODE
	case models.FactoryOnboardingAgentHarnessCursor:
		return pb.FactoryOnboarding_AGENT_HARNESS_CURSOR
	case models.FactoryOnboardingAgentHarnessCodex:
		return pb.FactoryOnboarding_AGENT_HARNESS_CODEX
	default:
		return pb.FactoryOnboarding_AGENT_HARNESS_UNSPECIFIED
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
			Type: step.Type,
			App: &pb.FactoryLine_AppStep{
				App:        step.AppID.String(),
				Entrypoint: step.Entrypoint,
			},
		}
		if step.MaxParallelism != nil {
			value := int32(*step.MaxParallelism)
			steps[i].MaxParallelism = &value
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

func serializeWorkOrder(
	f *models.Factory,
	order *models.FactoryWorkOrder,
	dispatches []models.FactoryWorkOrderLineDispatchRecord,
	createdByAutomation *factory.AutomationRef,
) *pb.WorkOrder {
	serializedDispatches := serializeWorkOrderLineDispatches(dispatches)

	var totalTokens, totalCostCents int64
	for _, d := range dispatches {
		for _, e := range d.Executions {
			totalTokens += e.TotalTokens
			totalCostCents += e.CostCents
		}
	}

	displayKey := ""
	if f != nil {
		displayKey = f.WorkOrderKey(order.Number)
	}

	return &pb.WorkOrder{
		Id:             order.ID.String(),
		Title:          order.Title,
		Description:    order.Description,
		Number:         order.Number,
		Key:            displayKey,
		State:          serializeWorkOrderState(order.State),
		Result:         serializeWorkOrderResult(order.Result),
		CreatedAt:      timestamppb.New(order.CreatedAt),
		UpdatedAt:      timestamppb.New(order.UpdatedAt),
		Assignees:      serializeWorkOrderAssignees(order.Assignees),
		LineDispatches: serializedDispatches,
		CreatedBy:      serializeWorkOrderCreator(order, createdByAutomation),
		TotalTokens:    totalTokens,
		TotalCostCents: totalCostCents,
	}
}

func serializeAutomationRef(ref *factory.AutomationRef) *pb.AutomationRef {
	if ref == nil {
		return nil
	}
	return &pb.AutomationRef{
		NodeId:   ref.NodeID,
		NodeName: ref.NodeName,
		AppId:    ref.AppID.String(),
		AppName:  ref.AppName,
	}
}

// The automation branch takes precedence: when a canvas automation opened the
// work order, its identity is the source of truth for authorship even if the
// underlying run was launched under a member's session.
func serializeWorkOrderCreator(
	order *models.FactoryWorkOrder,
	createdByAutomation *factory.AutomationRef,
) *pb.WorkOrderCreator {
	if createdByAutomation != nil {
		return &pb.WorkOrderCreator{
			Kind: &pb.WorkOrderCreator_Automation{Automation: serializeAutomationRef(createdByAutomation)},
		}
	}

	if order.CreatedByID == nil {
		return nil
	}

	name := order.CreatedByID.String()
	if order.CreatedBy != nil {
		name = order.CreatedBy.Name
	}

	return &pb.WorkOrderCreator{
		Kind: &pb.WorkOrderCreator_User{
			User: &pb.UserRef{
				Id:   order.CreatedByID.String(),
				Name: name,
			},
		},
	}
}

func serializeWorkOrderLineDispatches(dispatches []models.FactoryWorkOrderLineDispatchRecord) []*pb.WorkOrderLineDispatch {
	result := make([]*pb.WorkOrderLineDispatch, 0, len(dispatches))
	for _, dispatch := range dispatches {
		result = append(result, serializeWorkOrderLineDispatch(dispatch))
	}
	return result
}

func serializeWorkOrderLineDispatch(dispatch models.FactoryWorkOrderLineDispatchRecord) *pb.WorkOrderLineDispatch {
	item := &pb.WorkOrderLineDispatch{
		Id: dispatch.ID.String(),
		Line: &pb.LineRef{
			Id:   dispatch.LineID.String(),
			Name: dispatch.LineName,
		},
		Steps:          serializeExecutionSteps(dispatch.Steps, dispatch.Executions),
		State:          serializeLineDispatchState(dispatch.State),
		Result:         serializeLineDispatchResult(dispatch.Result),
		CreatedAt:      timestamppb.New(dispatch.CreatedAt),
		StepExecutions: serializeWorkOrderExecutions(dispatch.Executions),
	}
	if dispatch.FinishedAt != nil {
		item.FinishedAt = timestamppb.New(*dispatch.FinishedAt)
	}
	if dispatch.QueueItem != nil {
		item.QueueItem = serializeWorkOrderQueueItem(dispatch.QueueItem, dispatch.Steps)
	}
	return item
}

func serializeWorkOrderQueueItem(
	item *models.FactoryWorkOrderQueueItemRecord,
	steps []models.FactoryLineStep,
) *pb.WorkOrderQueueItem {
	result := &pb.WorkOrderQueueItem{
		Id:        item.ID.String(),
		StepName:  item.StepName,
		StepIndex: int32(item.StepIndex),
		Position:  int32(item.Position),
		CreatedAt: timestamppb.New(item.CreatedAt),
	}
	if item.StepIndex >= 0 && item.StepIndex < len(steps) {
		result.AppId = steps[item.StepIndex].AppID.String()
	}
	return result
}

func serializeLineDispatchState(state string) pb.WorkOrderLineDispatch_State {
	switch state {
	case models.FactoryWorkOrderLineDispatchStateActive:
		return pb.WorkOrderLineDispatch_STATE_ACTIVE
	case models.FactoryWorkOrderLineDispatchStateFinished:
		return pb.WorkOrderLineDispatch_STATE_FINISHED
	default:
		return pb.WorkOrderLineDispatch_STATE_UNKNOWN
	}
}

func serializeLineDispatchResult(result string) pb.WorkOrderLineDispatch_Result {
	switch result {
	case models.CanvasRunResultPassed:
		return pb.WorkOrderLineDispatch_RESULT_PASSED
	case models.CanvasRunResultFailed:
		return pb.WorkOrderLineDispatch_RESULT_FAILED
	case models.CanvasRunResultCancelled:
		return pb.WorkOrderLineDispatch_RESULT_CANCELLED
	default:
		return pb.WorkOrderLineDispatch_RESULT_UNKNOWN
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
		Id:          execution.ID.String(),
		Step:        execution.StepName,
		StepIndex:   int32(execution.StepIndex),
		State:       serializeWorkOrderExecutionState(execution.Status, execution.RunState),
		Result:      serializeWorkOrderExecutionResult(execution.Result, execution.RunResult),
		CreatedAt:   timestamppb.New(execution.CreatedAt),
		UpdatedAt:   timestamppb.New(execution.UpdatedAt),
		TotalTokens: execution.TotalTokens,
		CostCents:   execution.CostCents,
		Run: &pb.WorkOrderExecution_RunRef{
			Id:      execution.RunID.String(),
			AppId:   execution.CanvasID.String(),
			AppName: execution.CanvasName,
		},
	}
	if execution.FinishedAt != nil {
		item.FinishedAt = timestamppb.New(*execution.FinishedAt)
	}
	return item
}

func serializeExecutionSteps(
	steps []models.FactoryLineStep,
	executions []models.FactoryWorkOrderExecutionRecord,
) []*pb.WorkOrderExecutionStep {
	if len(steps) == 0 {
		return nil
	}

	nameByIndex := make(map[int]string, len(executions))
	for _, execution := range executions {
		if execution.CanvasName != "" {
			nameByIndex[execution.StepIndex] = execution.CanvasName
		}
	}

	result := make([]*pb.WorkOrderExecutionStep, len(steps))
	for i := range steps {
		result[i] = &pb.WorkOrderExecutionStep{
			Name:      nameByIndex[i],
			StepIndex: int32(i),
		}
	}
	return result
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
