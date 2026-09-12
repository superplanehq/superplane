package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
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
	session, factoryModel, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId())
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
	session, factoryModel, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId())
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
	session, factoryModel, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	if err := requireAnalysisPlanningSession(session); err != nil {
		return nil, factoryErrorToStatus(err, "failed to send planning session message")
	}
	db := database.DB(ctx)
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockAnalysisSessionAndDraft(tx, session); err != nil {
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
		if err := session.SendUserMessage(tx, req.GetText()); err != nil {
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

func lockAnalysisSessionAndDraft(tx *gorm.DB, session *models.FactoryPlanningSession) error {
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", session.ID).First(session).Error; err != nil {
		return err
	}
	if err := requireAnalysisPlanningSession(session); err != nil {
		return err
	}
	if session.DraftWorkOrderID == nil {
		return models.ErrFactoryPlanningSessionInvalid
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
		return err
	}
	if order.State != models.FactoryWorkOrderStateDraft {
		return models.ErrFactoryPlanningSessionInvalid
	}
	return nil
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
) (*models.FactoryPlanningSession, *models.Factory, error) {
	orgID, parsedFactoryID, _, err := planningSessionActor(ctx, organizationID, factoryID)
	if err != nil {
		return nil, nil, err
	}
	parsedSessionID, err := parseSessionID(sessionID)
	if err != nil {
		return nil, nil, factoryErrorToStatus(err, "failed to load planning session")
	}
	db := database.DB(ctx)
	factoryModel, err := models.FindFactory(db, orgID, parsedFactoryID)
	if err != nil {
		return nil, nil, factoryErrorToStatus(err, "failed to load planning session")
	}
	session, err := models.FindPlanningSession(db, orgID, parsedFactoryID, parsedSessionID)
	if err != nil {
		return nil, nil, factoryErrorToStatus(err, "failed to load planning session")
	}
	return session, factoryModel, nil
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
		messagesOut = append(messagesOut, &pb.PlanningSessionMessage{
			Id:        message.ID.String(),
			Role:      message.Role,
			Text:      message.Text,
			CreatedAt: timestamppb.New(message.CreatedAt),
		})
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
	return nodeID == intakeAnalysisNodeID
}
