package factories

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func FindPlanningSessionByWorkOrder(ctx context.Context, organizationID string, req *pb.FindPlanningSessionByWorkOrderRequest) (*pb.FindPlanningSessionByWorkOrderResponse, error) {
	orgID, factoryID, _, err := planningSessionActor(ctx, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, err
	}
	workOrderID, err := parseOptionalPlanningWorkOrderID(req.GetWorkOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to find planning session")
	}
	if workOrderID == uuid.Nil {
		return nil, factoryErrorToStatus(invalidArgument("work order id is required"), "failed to find planning session")
	}
	db := database.DB(ctx)
	factoryModel, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to find planning session")
	}
	session, err := models.FindPlanningSessionByDraftWorkOrder(db, orgID, factoryID, workOrderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to find planning session")
	}
	if session.State != models.PlanningSessionStateEnded {
		if err := session.Heartbeat(db); err != nil {
			return nil, factoryErrorToStatus(err, "failed to find planning session")
		}
	}
	serialized, err := serializePlanningSession(db, factoryModel, session)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to find planning session")
	}
	return &pb.FindPlanningSessionByWorkOrderResponse{Session: serialized}, nil
}

func DescribePlanningSession(ctx context.Context, organizationID string, req *pb.DescribePlanningSessionRequest) (*pb.DescribePlanningSessionResponse, error) {
	session, factoryModel, _, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	if session.State != models.PlanningSessionStateEnded {
		if err := session.Heartbeat(db); err != nil {
			return nil, factoryErrorToStatus(err, "failed to describe planning session")
		}
	}
	serialized, err := serializePlanningSession(db, factoryModel, session)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe planning session")
	}
	return &pb.DescribePlanningSessionResponse{Session: serialized}, nil
}

func EndPlanningSession(ctx context.Context, organizationID string, req *pb.EndPlanningSessionRequest) (*pb.EndPlanningSessionResponse, error) {
	session, factoryModel, _, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	if err := session.End(db); err != nil {
		return nil, factoryErrorToStatus(err, "failed to end planning session")
	}
	cancelPlanningSessionRun(ctx, db, session)
	serialized, err := serializePlanningSession(db, factoryModel, session)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to end planning session")
	}
	return &pb.EndPlanningSessionResponse{Session: serialized}, nil
}

func SendPlanningSessionMessage(ctx context.Context, organizationID string, req *pb.SendPlanningSessionMessageRequest) (*pb.SendPlanningSessionMessageResponse, error) {
	session, factoryModel, userID, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	if err := requireAnalysisPlanningSession(session); err != nil {
		return nil, factoryErrorToStatus(err, "failed to send planning session message")
	}
	db := database.DB(ctx)
	assigned := false
	if err := db.Transaction(func(tx *gorm.DB) error {
		draft, err := lockAnalysisSessionAndDraft(tx, session)
		if err != nil {
			return err
		}
		restartAnalysis := session.NeedsAnalysisRestart(tx)
		if restartAnalysis {
			if err := session.Reopen(tx); err != nil {
				return err
			}
			if err := session.DetachAgentRun(tx); err != nil {
				return err
			}
		}
		if err := session.SendUserMessage(tx, req.GetText(), userID); err != nil {
			return err
		}
		// Taking part in the refinement makes the sender an owner of the draft.
		if assigned, err = draft.AddAssignee(tx, userID, userID); err != nil {
			return err
		}
		if !restartAnalysis {
			return nil
		}
		if err := session.MarkUserMessagesDelivered(tx); err != nil {
			return err
		}
		return restartAnalysisCanvasRun(tx, factoryModel, session)
	}); err != nil {
		return nil, factoryErrorToStatus(err, "failed to send planning session message")
	}
	if assigned {
		publishDraftAssigneesUpdated(session)
	}
	serialized, err := serializePlanningSession(db, factoryModel, session)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to send planning session message")
	}
	return &pb.SendPlanningSessionMessageResponse{Session: serialized}, nil
}

func AnswerPlanningSessionSurvey(ctx context.Context, organizationID string, req *pb.AnswerPlanningSessionSurveyRequest) (*pb.AnswerPlanningSessionSurveyResponse, error) {
	response, err := SendPlanningSessionMessage(ctx, organizationID, &pb.SendPlanningSessionMessageRequest{
		FactoryId: req.GetFactoryId(),
		SessionId: req.GetSessionId(),
		Text:      req.GetText(),
	})
	if err != nil {
		return nil, err
	}
	return &pb.AnswerPlanningSessionSurveyResponse{Session: response.Session}, nil
}

// publishDraftAssigneesUpdated tells open boards that the draft has a new owner.
func publishDraftAssigneesUpdated(session *models.FactoryPlanningSession) {
	if session.DraftWorkOrderID == nil {
		return
	}
	if err := messages.PublishFactoryWorkOrderUpdated(
		session.FactoryID.String(),
		session.DraftWorkOrderID.String(),
		factoryevents.EventTypeOrderAssigneesUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for draft %s", *session.DraftWorkOrderID)
	}
}

// lockAnalysisSessionAndDraft locks the session and its draft for update and
// returns the draft, so the caller can change it in the same transaction.
func lockAnalysisSessionAndDraft(tx *gorm.DB, session *models.FactoryPlanningSession) (*models.FactoryWorkOrder, error) {
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", session.ID).First(session).Error; err != nil {
		return nil, err
	}
	if err := requireAnalysisPlanningSession(session); err != nil {
		return nil, err
	}
	if session.DraftWorkOrderID == nil {
		return nil, models.ErrFactoryPlanningSessionInvalid
	}
	var order models.FactoryWorkOrder
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"id = ? AND organization_id = ? AND factory_id = ?",
			*session.DraftWorkOrderID,
			session.OrganizationID,
			session.FactoryID,
		).
		First(&order).
		Error; err != nil {
		return nil, err
	}
	if order.State != models.FactoryWorkOrderStateDraft {
		return nil, models.ErrFactoryPlanningSessionInvalid
	}
	return &order, nil
}

func restartAnalysisCanvasRun(db *gorm.DB, factoryModel *models.Factory, session *models.FactoryPlanningSession) error {
	if session.DraftWorkOrderID == nil || session.CanvasID == nil {
		return models.ErrFactoryPlanningSessionInvalid
	}
	order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	if err != nil {
		return err
	}
	if err := workersctx.EmitWorkOrderCreatedOnCanvas(db, factoryModel, order, *session.CanvasID); err != nil {
		log.WithError(err).Warnf("failed to restart analysis run for session %s", session.ID)
		return err
	}
	return nil
}

func planningSessionActor(ctx context.Context, organizationID, factoryID string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, factoryErrorToStatus(err, "failed to load planning session")
	}
	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to load planning session")
	}
	factory, err := findFactory(database.DB(ctx), orgID, factoryID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, factoryErrorToStatus(err, "failed to load planning session")
	}
	return orgID, factory.ID, userID, nil
}

func parseSessionID(sessionID string) (uuid.UUID, error) {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid planning session id")
	}
	return id, nil
}

func parseOptionalPlanningWorkOrderID(workOrderID string) (uuid.UUID, error) {
	raw := strings.TrimSpace(workOrderID)
	if raw == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid work order id")
	}
	return id, nil
}

func loadPlanningSession(
	ctx context.Context,
	organizationID, factoryID, sessionID string,
) (*models.FactoryPlanningSession, *models.Factory, uuid.UUID, error) {
	orgID, parsedFactoryID, userID, err := planningSessionActor(ctx, organizationID, factoryID)
	if err != nil {
		return nil, nil, uuid.Nil, err
	}
	parsedSessionID, err := parseSessionID(sessionID)
	if err != nil {
		return nil, nil, uuid.Nil, factoryErrorToStatus(err, "failed to load planning session")
	}
	db := database.DB(ctx)
	factoryModel, err := models.FindFactory(db, orgID, parsedFactoryID)
	if err != nil {
		return nil, nil, uuid.Nil, factoryErrorToStatus(err, "failed to load planning session")
	}
	session, err := models.FindPlanningSession(db, orgID, parsedFactoryID, parsedSessionID)
	if err != nil {
		return nil, nil, uuid.Nil, factoryErrorToStatus(err, "failed to load planning session")
	}
	return session, factoryModel, userID, nil
}

func requireAnalysisPlanningSession(session *models.FactoryPlanningSession) error {
	if session == nil || !session.IsAnalysisSession() {
		return models.ErrFactoryPlanningSessionInvalid
	}
	return nil
}

func cancelPlanningSessionRun(ctx context.Context, db *gorm.DB, session *models.FactoryPlanningSession) {
	if session.CanvasID == nil || session.CanvasRunID == nil {
		return
	}
	cancelPlanningSessionRunByID(ctx, db, session.OrganizationID, *session.CanvasID, *session.CanvasRunID)
}

func cancelPlanningSessionRunByID(ctx context.Context, db *gorm.DB, organizationID, canvasID, runID uuid.UUID) {
	canvas, err := models.FindCanvasInTransaction(db, organizationID, canvasID)
	if err != nil {
		log.WithError(err).Warnf("Failed to load planning session canvas %s", canvasID)
		return
	}
	if _, err := canvases.CancelRun(ctx, db, canvas, runID); err != nil {
		log.WithError(err).Warnf("Failed to cancel planning session run %s", runID)
	}
}

func serializePlanningSession(tx *gorm.DB, factoryModel *models.Factory, session *models.FactoryPlanningSession) (*pb.PlanningSession, error) {
	orders, err := session.CreatedOrders(tx)
	if err != nil {
		return nil, err
	}
	created := make([]*pb.PlanningSessionCreatedOrder, 0, len(orders))
	for _, order := range orders {
		created = append(created, &pb.PlanningSessionCreatedOrder{
			Id:          order.ID.String(),
			Key:         factoryModel.WorkOrderKey(order.Number),
			Title:       order.Title,
			Description: order.Description,
		})
	}

	messages, err := models.ListPlanningSessionMessages(tx, session.ID)
	if err != nil {
		return nil, err
	}
	messagesOut := make([]*pb.PlanningSessionMessage, 0, len(messages))
	for _, message := range messages {
		messagesOut = append(messagesOut, serializePlanningSessionMessage(message))
	}
	activities, err := models.ListPlanningSessionActivities(tx, session.ID)
	if err != nil {
		return nil, err
	}
	activitiesOut := make([]*pb.PlanningSessionActivity, 0, len(activities))
	for _, activity := range activities {
		activitiesOut = append(activitiesOut, serializePlanningSessionActivity(activity))
	}

	out := &pb.PlanningSession{
		Id:         session.ID.String(),
		FactoryId:  session.FactoryID.String(),
		Repository: session.Repository,
		State:      session.State,
		Messages:   messagesOut,
		Created:    created,
		WaitState:  session.WaitState,
		Kind:       planningSessionKindToProto(session.Kind),
		Activities: activitiesOut,
	}
	if session.CanvasID != nil {
		out.CanvasId = session.CanvasID.String()
	}
	if session.CanvasRunID != nil {
		out.CanvasRunId = session.CanvasRunID.String()
	}
	executionID, err := planningSessionExecutionID(tx, session)
	if err != nil {
		return nil, err
	}
	out.ExecutionId = executionID
	out.SelectableModelKey = session.SelectableModelKey
	if draft := session.Draft(); strings.TrimSpace(draft.Title) != "" {
		out.Draft = &pb.PlanningSessionDraft{
			Title:       draft.Title,
			Description: draft.Description,
			WorkOrderId: draft.WorkOrderID,
		}
	}
	if survey := session.CurrentSurvey(); session.SurveyID != nil && len(survey.Questions) > 0 {
		out.Survey = &pb.PlanningSessionSurvey{
			Id:        session.SurveyID.String(),
			Questions: planningSessionSurveyQuestions(survey),
		}
	}
	return out, nil
}

func serializePlanningSessionMessage(message models.PlanningSessionMessage) *pb.PlanningSessionMessage {
	out := &pb.PlanningSessionMessage{
		Id:        message.ID.String(),
		Role:      message.Role,
		Text:      message.Text,
		CreatedAt: timestamppb.New(message.CreatedAt),
	}
	if message.UserID != nil {
		out.UserId = message.UserID.String()
	}
	if message.ActivityID != nil {
		out.ActivityId = message.ActivityID.String()
	}
	return out
}

func serializePlanningSessionActivity(activity models.PlanningSessionActivity) *pb.PlanningSessionActivity {
	snapshot := activity.Snapshot.Data()
	items := make([]*pb.PlanningSessionActivityItem, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		outputs := make([]*pb.PlanningSessionActivityOutput, 0, len(item.OutputStreams))
		for _, output := range item.OutputStreams {
			outputs = append(outputs, &pb.PlanningSessionActivityOutput{Stream: output.Stream, Text: output.Text})
		}
		serialized := &pb.PlanningSessionActivityItem{
			Type:          item.Type,
			Id:            item.ID,
			Kind:          item.Kind,
			Text:          item.Text,
			Name:          item.Name,
			Input:         item.Input,
			Output:        item.Output,
			OutputStreams: outputs,
			Status:        item.Status,
			Code:          item.Code,
			DurationMs:    item.DurationMs,
			Signal:        item.Signal,
			Truncated:     item.Truncated,
		}
		if item.StartedAt > 0 {
			serialized.StartedAt = timestamppb.New(time.UnixMilli(item.StartedAt))
		}
		if item.ExitCode != nil {
			exitCode := int32(*item.ExitCode)
			serialized.ExitCode = &exitCode
		}
		items = append(items, serialized)
	}
	out := &pb.PlanningSessionActivity{
		Id:            activity.ID.String(),
		SchemaVersion: int32(activity.SchemaVersion),
		Provider:      activity.Provider,
		Status:        activity.Status,
		LastSequence:  activity.LastSequence,
		StartedAt:     timestamppb.New(activity.StartedAt),
		Items:         items,
		Truncated:     snapshot.Truncated,
		Turn:          int32(snapshot.Turn),
	}
	if activity.CompletedAt != nil {
		out.CompletedAt = timestamppb.New(*activity.CompletedAt)
	}
	return out
}

func planningSessionKindToProto(kind string) pb.PlanningSessionKind {
	switch kind {
	case models.PlanningSessionKindTaskCreation:
		return pb.PlanningSessionKind_PLANNING_SESSION_KIND_TASK_CREATION
	case models.PlanningSessionKindWorkOrderAnalysis:
		return pb.PlanningSessionKind_PLANNING_SESSION_KIND_WORK_ORDER_ANALYSIS
	default:
		return pb.PlanningSessionKind_PLANNING_SESSION_KIND_UNSPECIFIED
	}
}

func planningSessionSurveyQuestions(survey models.PlanningSessionSurvey) []*pb.PlanningSessionSurveyQuestion {
	questions := make([]*pb.PlanningSessionSurveyQuestion, 0, len(survey.Questions))
	for _, question := range survey.Questions {
		questions = append(questions, &pb.PlanningSessionSurveyQuestion{
			Prompt:  question.Prompt,
			Options: question.Options,
		})
	}
	return questions
}

func planningSessionExecutionID(tx *gorm.DB, session *models.FactoryPlanningSession) (string, error) {
	if session.CanvasID == nil || session.CanvasRunID == nil {
		return "", nil
	}
	executions, err := models.ListExecutionsForRunsInTransaction(tx, *session.CanvasID, []uuid.UUID{*session.CanvasRunID})
	if err != nil {
		return "", err
	}
	for i := len(executions) - 1; i >= 0; i-- {
		if isPlanningSessionAgentNode(executions[i].NodeID) {
			return executions[i].ID.String(), nil
		}
	}
	return "", nil
}

func isPlanningSessionAgentNode(nodeID string) bool {
	return nodeID == intakeAnalysisNodeID || nodeID == backlogRefinementNodeID
}
