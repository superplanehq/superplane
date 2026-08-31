package factories

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func serializeFactory(factory *models.Factory) *pb.Factory {
	serialized := &pb.Factory{
		Id:          factory.ID.String(),
		Name:        factory.Name,
		Description: factory.Description,
		Key:         factory.Key,
		Onboarding:  serializeFactoryOnboarding(factory),
	}
	if factory.HostedSpendBudgetCents != nil {
		serialized.HostedSpendBudgetCents = factory.HostedSpendBudgetCents
	}
	return serialized
}

func serializeFactoryWithLines(
	factory *models.Factory,
	lines []models.FactoryLine,
	metricsByLine map[uuid.UUID]*pb.FactoryLineMetrics,
) *pb.Factory {
	serialized := serializeFactory(factory)
	serialized.Lines = serializeFactoryLines(lines, metricsByLine)
	return serialized
}

func serializeFactoryWithLineMetrics(
	tx *gorm.DB,
	factory *models.Factory,
	lines []models.FactoryLine,
) (*pb.Factory, error) {
	if len(lines) == 0 {
		return serializeFactoryWithLines(factory, lines, nil), nil
	}
	metricsByLine, err := loadFactoryLineMetrics(tx, factory.ID)
	if err != nil {
		return nil, err
	}
	return serializeFactoryWithLines(factory, lines, metricsByLine), nil
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

func serializeFactoryLines(lines []models.FactoryLine, metricsByLine map[uuid.UUID]*pb.FactoryLineMetrics) []*pb.FactoryLine {
	result := make([]*pb.FactoryLine, len(lines))
	for i := range lines {
		serialized := serializeFactoryLine(&lines[i])
		if metricsByLine != nil {
			serialized.Metrics = metricsByLine[lines[i].ID]
		}
		result[i] = serialized
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

func serializeFactoryIntakes(intakes []models.FactoryIntake, specs map[uuid.UUID]models.LiveCanvasSpec) []*pb.FactoryIntake {
	result := make([]*pb.FactoryIntake, len(intakes))
	for i := range intakes {
		result[i] = serializeFactoryIntake(&intakes[i], specs[intakes[i].CanvasID])
	}
	return result
}

func serializeFactoryIntake(intake *models.FactoryIntake, spec models.LiveCanvasSpec) *pb.FactoryIntake {
	graph := resolveIntakeGraph(intake.Source, spec)

	serialized := &pb.FactoryIntake{
		Id:        intake.ID.String(),
		FactoryId: intake.FactoryID.String(),
		CanvasId:  intake.CanvasID.String(),
		Name:      intake.Name(),
		Source:    serializeFactoryIntakeSource(intake.Source),
		Settings:  serializeIntakeSettings(intakeSettingsFromGraph(graph, spec)),
		Healthy:   graph.Healthy(spec.Edges),
		CreatedAt: timestamppb.New(intake.CreatedAt),
		UpdatedAt: timestamppb.New(intake.UpdatedAt),
	}

	if intake.Canvas != nil {
		serialized.Description = intake.Canvas.Description
	}

	return serialized
}

func serializeFactoryIntakeSource(source string) pb.FactoryIntake_Source {
	switch source {
	case models.FactoryIntakeSourceGitHubIssues:
		return pb.FactoryIntake_SOURCE_GITHUB_ISSUES
	case models.FactoryIntakeSourceSentryExceptions:
		return pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS
	case models.FactoryIntakeSourcePagerDutyIncidents:
		return pb.FactoryIntake_SOURCE_PAGERDUTY_INCIDENTS
	default:
		return pb.FactoryIntake_SOURCE_UNSPECIFIED
	}
}

func parseFactoryIntakeSource(source pb.FactoryIntake_Source) (string, error) {
	switch source {
	case pb.FactoryIntake_SOURCE_GITHUB_ISSUES:
		return models.FactoryIntakeSourceGitHubIssues, nil
	case pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS:
		return models.FactoryIntakeSourceSentryExceptions, nil
	case pb.FactoryIntake_SOURCE_PAGERDUTY_INCIDENTS:
		return models.FactoryIntakeSourcePagerDutyIncidents, nil
	default:
		return "", invalidArgument("intake source is required")
	}
}

func serializeFactoryPRFeedbackHandlers(handlers []models.FactoryPRFeedbackHandler, specs map[uuid.UUID]models.LiveCanvasSpec) []*pb.FactoryPRFeedbackHandler {
	result := make([]*pb.FactoryPRFeedbackHandler, len(handlers))
	for i := range handlers {
		result[i] = serializeFactoryPRFeedbackHandler(&handlers[i], specs[handlers[i].CanvasID])
	}
	return result
}

func serializeFactoryPRFeedbackHandler(handler *models.FactoryPRFeedbackHandler, spec models.LiveCanvasSpec) *pb.FactoryPRFeedbackHandler {
	graph := resolvePRFeedbackGraph(spec)

	serialized := &pb.FactoryPRFeedbackHandler{
		Id:        handler.ID.String(),
		FactoryId: handler.FactoryID.String(),
		CanvasId:  handler.CanvasID.String(),
		Name:      handler.Name(),
		Subject:   serializeFactoryPRFeedbackHandlerSubject(handler.Subject),
		Source:    serializeFactoryPRFeedbackHandlerSource(handler.Source),
		Settings:  serializePRFeedbackSettings(prFeedbackSettingsFromGraph(graph, spec)),
		Healthy:   graph.Healthy(spec),
		CreatedAt: timestamppb.New(handler.CreatedAt),
		UpdatedAt: timestamppb.New(handler.UpdatedAt),
	}

	if handler.Canvas != nil {
		serialized.Description = handler.Canvas.Description
	}

	return serialized
}

func serializeFactoryPRFeedbackHandlerSubject(subject string) pb.FactoryPRFeedbackHandler_Subject {
	switch subject {
	case models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest:
		return pb.FactoryPRFeedbackHandler_SUBJECT_GITHUB_PULL_REQUEST
	default:
		return pb.FactoryPRFeedbackHandler_SUBJECT_UNSPECIFIED
	}
}

func serializeFactoryPRFeedbackHandlerSource(source string) pb.FactoryPRFeedbackHandler_Source {
	switch source {
	case models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion:
		return pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_DISCUSSION
	default:
		return pb.FactoryPRFeedbackHandler_SOURCE_UNSPECIFIED
	}
}

func parseFactoryPRFeedbackHandlerSubject(subject pb.FactoryPRFeedbackHandler_Subject) (string, error) {
	switch subject {
	case pb.FactoryPRFeedbackHandler_SUBJECT_UNSPECIFIED, pb.FactoryPRFeedbackHandler_SUBJECT_GITHUB_PULL_REQUEST:
		return models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, nil
	default:
		return "", invalidArgument("PR feedback handler subject is not supported")
	}
}

func parseFactoryPRFeedbackHandlerSource(source pb.FactoryPRFeedbackHandler_Source) (string, error) {
	switch source {
	case pb.FactoryPRFeedbackHandler_SOURCE_UNSPECIFIED, pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_DISCUSSION:
		return models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion, nil
	default:
		return "", invalidArgument("PR feedback handler source is not supported")
	}
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

func serializeFactories(factories []models.Factory, linesByFactory map[uuid.UUID][]models.FactoryLine) []*pb.Factory {
	result := make([]*pb.Factory, len(factories))
	for i := range factories {
		result[i] = serializeFactoryWithLines(&factories[i], linesByFactory[factories[i].ID], nil)
	}
	return result
}

func serializeWorkOrder(
	f *models.Factory,
	order *models.FactoryWorkOrder,
	dispatches []models.FactoryWorkOrderLineDispatchRecord,
	createdByAutomation *factory.AutomationRef,
	usage models.UsageTotals,
) (*pb.WorkOrder, error) {
	serializedDispatches := serializeWorkOrderLineDispatches(dispatches)

	displayKey := ""
	if f != nil {
		displayKey = f.WorkOrderKey(order.Number)
	}

	statusNotes, err := serializeWorkOrderStatusNotes(order)
	if err != nil {
		return nil, err
	}

	return &pb.WorkOrder{
		Id:                   order.ID.String(),
		Title:                order.Title,
		Description:          order.Description,
		Number:               order.Number,
		Key:                  displayKey,
		State:                serializeWorkOrderState(order.State),
		Result:               serializeWorkOrderResult(order.Result),
		CreatedAt:            timestamppb.New(order.CreatedAt),
		UpdatedAt:            timestamppb.New(order.UpdatedAt),
		Assignees:            serializeWorkOrderAssignees(order.Assignees),
		LineDispatches:       serializedDispatches,
		CreatedBy:            serializeWorkOrderCreator(order, createdByAutomation),
		TotalTokens:          usage.TotalTokens,
		TotalCostCents:       usage.CostCents(),
		TotalDurationSeconds: usage.DurationSeconds,
		StatusNotes:          statusNotes,
		Origin:               serializeWorkOrderOrigin(order),
	}, nil
}

func serializeWorkOrderOrigin(order *models.FactoryWorkOrder) *pb.WorkOrderOrigin {
	origin := order.Origin()
	if origin == nil {
		return nil
	}

	return &pb.WorkOrderOrigin{
		Url:   origin.URL,
		Label: origin.Label,
	}
}

func serializeWorkOrderStatusNotes(order *models.FactoryWorkOrder) ([]*pb.WorkOrderStatusNote, error) {
	notes, err := order.StatusNotes()
	if err != nil {
		return nil, err
	}
	if len(notes) == 0 {
		return nil, nil
	}

	serialized := make([]*pb.WorkOrderStatusNote, 0, len(notes))
	for i := range notes {
		serialized = append(serialized, serializeWorkOrderStatusNote(&notes[i]))
	}
	return serialized, nil
}

func serializeWorkOrderStatusNote(note *models.FactoryWorkOrderStatusNote) *pb.WorkOrderStatusNote {
	serialized := &pb.WorkOrderStatusNote{
		Key:                 note.Key,
		Kind:                note.Kind,
		Headline:            note.Headline,
		Body:                note.Body,
		CtaLabel:            note.CtaLabel,
		CtaUrl:              note.CtaURL,
		Automation:          serializeAutomationRef(note.Automation),
		UpdatedAt:           timestamppb.New(note.UpdatedAt),
		ShowOnlyWhenWaiting: note.ShowOnlyWhenWaiting,
	}
	if note.Run != nil {
		serialized.RunId = note.Run.ID.String()
	}

	return serialized
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
		Id:              execution.ID.String(),
		Step:            execution.StepName,
		StepIndex:       int32(execution.StepIndex),
		State:           serializeWorkOrderExecutionState(execution.Status, execution.RunState),
		Result:          serializeWorkOrderExecutionResult(execution.Result, execution.RunResult),
		CreatedAt:       timestamppb.New(execution.CreatedAt),
		UpdatedAt:       timestamppb.New(execution.UpdatedAt),
		TotalTokens:     execution.TotalTokens,
		CostCents:       execution.CostCents,
		DurationSeconds: execution.DurationSeconds,
	}
	if execution.RunID != nil {
		runRef := &pb.WorkOrderExecution_RunRef{
			Id:      execution.RunID.String(),
			AppName: execution.CanvasName,
		}
		if execution.CanvasID != nil {
			runRef.AppId = execution.CanvasID.String()
		}
		item.Run = runRef
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
		name := execution.CanvasName
		if name == "" {
			name = execution.StepName
		}
		if name != "" {
			nameByIndex[execution.StepIndex] = name
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
