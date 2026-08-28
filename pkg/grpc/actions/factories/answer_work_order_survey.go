package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func AnswerWorkOrderSurvey(
	ctx context.Context,
	organizationID string,
	req *pb.AnswerWorkOrderSurveyRequest,
) (*pb.AnswerWorkOrderSurveyResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to answer work order survey")
	}
	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to answer work order survey")
	}
	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to answer work order survey")
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to answer work order survey")
	}

	answers := make([]models.WorkOrderSurveyAnswer, 0, len(req.GetAnswers()))
	for _, answer := range req.GetAnswers() {
		answers = append(answers, models.WorkOrderSurveyAnswer{ID: answer.GetId(), Value: answer.GetValue()})
	}

	db := database.DB(ctx)
	var factoryModel *models.Factory
	var order *models.FactoryWorkOrder
	var surveyID uuid.UUID
	err = db.Transaction(func(tx *gorm.DB) error {
		var findErr error
		factoryModel, findErr = models.FindFactory(tx, orgID, factoryID)
		if findErr != nil {
			return findErr
		}
		order, findErr = factoryModel.FindWorkOrder(tx, orderID)
		if findErr != nil {
			return findErr
		}
		survey, findErr := order.PendingSurvey(tx)
		if findErr != nil {
			return findErr
		}
		surveyID = survey.ID
		return survey.Answer(tx, userID, answers)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to answer work order survey")
	}

	PublishWorkOrderSurveyUpdated(orgID, factoryID, orderID, surveyID, false)

	serialized, err := loadAndSerializeWorkOrder(ctx, factoryModel, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to answer work order survey")
	}
	return &pb.AnswerWorkOrderSurveyResponse{Order: serialized}, nil
}
