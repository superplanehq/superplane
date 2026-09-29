package public

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// dispatchPlanningWorkOrder starts a draft on the configured line. Tests
// replace it to count calls. Production uses factories.DispatchWorkOrder.
var dispatchPlanningWorkOrder = factories.DispatchWorkOrder

// maybeAutoStartPlanningDraft starts the draft when this wait is the first
// transition into pending and every automatic-start guard passes. A skip or
// a dispatch error leaves the draft and does not fail the planning turn.
func maybeAutoStartPlanningDraft(db *gorm.DB, session *models.FactoryPlanningSession) {
	if session == nil || !session.IsAnalysisSession() {
		return
	}
	reason, err := autoStartPlanningDraft(db, session)
	if err != nil {
		log.WithError(err).WithFields(planningAutoStartFields(session)).Error("automatic start failed")
		return
	}
	if reason == "" {
		return
	}
	log.WithFields(planningAutoStartFields(session)).WithField("reason", reason).Info("skipped automatic start")
}

func autoStartPlanningDraft(db *gorm.DB, session *models.FactoryPlanningSession) (string, error) {
	factoryModel, err := models.FindFactory(db, session.OrganizationID, session.FactoryID)
	if err != nil {
		return "", err
	}
	planning := factoryModel.Planning()
	if !planning.AutoStart {
		return "automatic start is off", nil
	}
	if !planning.Confidence {
		return "confidence check is off", nil
	}
	if hasOutstandingPlanningQuestion(session) {
		return "survey is pending", nil
	}
	if session.DraftWorkOrderID == nil {
		return "work order is missing", nil
	}

	order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	if err != nil {
		return "", err
	}
	if order.State != models.FactoryWorkOrderStateDraft || !order.IsDispatchable() {
		return "work order is not a draft", nil
	}
	ready, err := confidenceScoreIsFive(db, order)
	if err != nil {
		return "", err
	}
	if !ready {
		return "confidence score is not 5", nil
	}

	lineName := planning.AutoStartLine
	if lineName == "" {
		return "line is missing", nil
	}
	line, err := factoryModel.FindLineByName(db, lineName)
	if errors.Is(err, models.ErrFactoryLineNotFound) {
		return "line is missing", nil
	}
	if err != nil {
		return "", err
	}
	if len(line.Steps) == 0 {
		return "line has no steps", nil
	}

	_, err = dispatchPlanningWorkOrder(planningAutoStartContext(db), session.OrganizationID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: session.FactoryID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	if err != nil {
		return "", err
	}
	return "", nil
}

func confidenceScoreIsFive(db *gorm.DB, order *models.FactoryWorkOrder) (bool, error) {
	checks, err := order.ListChecks(db)
	if err != nil {
		return false, err
	}
	for _, check := range checks {
		if check.Key != models.PlanningConfidenceCheckKey {
			continue
		}
		return check.Score == float64(models.PlanningScoreMax), nil
	}
	return false, nil
}

func planningAutoStartContext(db *gorm.DB) context.Context {
	ctx := context.Background()
	if db != nil && db.Statement != nil && db.Statement.Context != nil {
		ctx = db.Statement.Context
	}
	return context.WithoutCancel(ctx)
}

func planningAutoStartFields(session *models.FactoryPlanningSession) log.Fields {
	fields := log.Fields{
		"factory_id":          session.FactoryID,
		"planning_session_id": session.ID,
	}
	if session.DraftWorkOrderID != nil {
		fields["work_order_id"] = *session.DraftWorkOrderID
	}
	return fields
}
