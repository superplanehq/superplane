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
	"gorm.io/gorm"
)

func StartPlanningSession(ctx context.Context, organizationID string, req *pb.StartPlanningSessionRequest) (*pb.StartPlanningSessionResponse, error) {
	orgID, factoryID, userID, err := planningSessionActor(ctx, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, err
	}

	db := database.DB(ctx)
	var session *models.FactoryPlanningSession
	var factoryModel *models.Factory
	err = db.Transaction(func(tx *gorm.DB) error {
		var findErr error
		factoryModel, findErr = models.FindFactory(tx, orgID, factoryID)
		if findErr != nil {
			return findErr
		}
		repository := strings.TrimSpace(req.GetRepository())
		if repository == "" {
			repository = strings.TrimSpace(factoryModel.OnboardingConfigValue().AppRepository)
		}
		if repository == "" {
			return invalidArgument("repository is required")
		}
		canvas, entrypoint, findErr := ensurePlanningCanvas(tx, factoryModel, userID)
		if findErr != nil {
			return findErr
		}
		session, findErr = factoryModel.StartPlanningSession(tx, models.StartPlanningSessionParams{
			CreatedByUserID: userID,
			Repository:      repository,
			CanvasID:        canvas.ID,
			Entrypoint:      entrypoint,
		})
		return findErr
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start planning session")
	}

	if session.CanvasID != nil && session.CanvasRunID != nil {
		if err := messages.NewCanvasRunMessage(session.CanvasID.String(), session.CanvasRunID.String()).PublishPending(); err != nil {
			log.WithError(err).Warnf("Failed to publish planning session run %s", session.CanvasRunID)
		}
	}

	serialized, err := serializePlanningSession(db, factoryModel, session)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start planning session")
	}
	return &pb.StartPlanningSessionResponse{Session: serialized}, nil
}

func DescribePlanningSession(ctx context.Context, organizationID string, req *pb.DescribePlanningSessionRequest) (*pb.DescribePlanningSessionResponse, error) {
	session, factoryModel, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId(), true)
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
	session, factoryModel, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId(), false)
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
	session, factoryModel, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId(), false)
	if err != nil {
		return nil, err
	}
	if err := session.SendUserMessage(database.DB(ctx), req.GetText()); err != nil {
		return nil, factoryErrorToStatus(err, "failed to send planning session message")
	}
	serialized, err := serializePlanningSession(database.DB(ctx), factoryModel, session)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to send planning session message")
	}
	return &pb.SendPlanningSessionMessageResponse{Session: serialized}, nil
}

func UpdatePlanningSessionDraft(ctx context.Context, organizationID string, req *pb.UpdatePlanningSessionDraftRequest) (*pb.UpdatePlanningSessionDraftResponse, error) {
	session, factoryModel, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId(), false)
	if err != nil {
		return nil, err
	}
	if err := session.UpdateDraft(database.DB(ctx), models.PlanningSessionDraft{
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
	}); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update planning session draft")
	}
	serialized, err := serializePlanningSession(database.DB(ctx), factoryModel, session)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update planning session draft")
	}
	return &pb.UpdatePlanningSessionDraftResponse{Session: serialized}, nil
}

func CreatePlanningSessionWorkOrder(ctx context.Context, organizationID string, req *pb.CreatePlanningSessionWorkOrderRequest) (*pb.CreatePlanningSessionWorkOrderResponse, error) {
	session, factoryModel, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId(), false)
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	order, err := session.CreateDraftWorkOrder(db, factoryModel, session.CreatedByUserID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create planning session work order")
	}
	workersctx.EmitWorkOrderCreated(db, factoryModel, order)
	if err := messages.PublishFactoryWorkOrderUpdated(factoryModel.ID.String(), order.ID.String(), factoryevents.EventTypeOrderStatusUpdated); err != nil {
		log.WithError(err).Warnf("Failed to publish work order created from planning session %s", session.ID)
	}
	serialized, err := serializePlanningSession(db, factoryModel, session)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create planning session work order")
	}
	return &pb.CreatePlanningSessionWorkOrderResponse{Session: serialized}, nil
}

func SkipPlanningSessionDraft(ctx context.Context, organizationID string, req *pb.SkipPlanningSessionDraftRequest) (*pb.SkipPlanningSessionDraftResponse, error) {
	session, factoryModel, err := loadPlanningSession(ctx, organizationID, req.GetFactoryId(), req.GetSessionId(), false)
	if err != nil {
		return nil, err
	}
	if err := session.SkipDraft(database.DB(ctx)); err != nil {
		return nil, factoryErrorToStatus(err, "failed to skip planning session draft")
	}
	serialized, err := serializePlanningSession(database.DB(ctx), factoryModel, session)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to skip planning session draft")
	}
	return &pb.SkipPlanningSessionDraftResponse{Session: serialized}, nil
}

func planningSessionActor(ctx context.Context, organizationID, factoryID string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, factoryErrorToStatus(err, "failed to load planning session")
	}
	parsedFactoryID, err := parseFactoryID(factoryID)
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
	return orgID, parsedFactoryID, userID, nil
}

func parseSessionID(sessionID string) (uuid.UUID, error) {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid planning session id")
	}
	return id, nil
}

func loadPlanningSession(
	ctx context.Context,
	organizationID, factoryID, sessionID string,
	expireStale bool,
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
	if expireStale {
		if _, err := session.EndIfStale(db, time.Now()); err != nil {
			return nil, nil, factoryErrorToStatus(err, "failed to load planning session")
		}
		if session.State == models.PlanningSessionStateEnded {
			cancelPlanningSessionRun(ctx, db, session)
		}
	}
	return session, factoryModel, nil
}

func cancelPlanningSessionRun(ctx context.Context, db *gorm.DB, session *models.FactoryPlanningSession) {
	if session.CanvasID == nil || session.CanvasRunID == nil {
		return
	}
	canvas, err := models.FindCanvasInTransaction(db, session.OrganizationID, *session.CanvasID)
	if err != nil {
		log.WithError(err).Warnf("Failed to load planning session canvas %s", session.CanvasID)
		return
	}
	if _, err := canvases.CancelRun(ctx, db, canvas, *session.CanvasRunID); err != nil {
		log.WithError(err).Warnf("Failed to cancel planning session run %s", session.CanvasRunID)
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

	messagesOut := make([]*pb.PlanningSessionMessage, 0, len(session.Messages))
	for _, message := range session.Messages {
		messagesOut = append(messagesOut, &pb.PlanningSessionMessage{
			Id:   message.ID,
			Kind: message.Kind,
			Role: message.Role,
			Text: message.Text,
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
	if draft := session.PendingDraft.Data(); strings.TrimSpace(draft.Title) != "" {
		out.Draft = &pb.PlanningSessionDraft{Title: draft.Title, Description: draft.Description}
	}
	return out, nil
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
		if executions[i].NodeID == "agent" {
			return executions[i].ID.String(), nil
		}
	}
	if len(executions) == 0 {
		return "", nil
	}
	return executions[len(executions)-1].ID.String(), nil
}
