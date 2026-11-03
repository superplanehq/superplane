package factories

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// A person adds checks with the agent rules: the checks cannot replace
// a SuperPlane check, and remove deletes only those checks.
func AddWorkOrderChecks(
	ctx context.Context,
	organizationID string,
	req *pb.AddWorkOrderChecksRequest,
) (*pb.AddWorkOrderChecksResponse, error) {
	const failure = "failed to add work order checks"
	params, err := models.AgentCheckParams(checkInputsFromProto(req.GetChecks()), nil)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, err.Error())
	}

	db := database.DB(ctx)
	factoryModel, order, err := openWorkOrderForChecks(db, organizationID, req.GetFactoryId(), req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, failure)
	}

	checks, err := order.ReportAgentChecks(db, params)
	if err != nil {
		return nil, factoryErrorToStatus(err, failure)
	}
	publishCheckUpdate(factoryModel, order)

	serialized, err := serializeChecks(checks)
	if err != nil {
		return nil, factoryErrorToStatus(err, failure)
	}
	return &pb.AddWorkOrderChecksResponse{Checks: serialized}, nil
}

func RemoveWorkOrderChecks(
	ctx context.Context,
	organizationID string,
	req *pb.RemoveWorkOrderChecksRequest,
) (*pb.RemoveWorkOrderChecksResponse, error) {
	const failure = "failed to remove work order checks"
	keys, err := models.AgentCheckKeys(req.GetKeys())
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, err.Error())
	}

	db := database.DB(ctx)
	factoryModel, order, err := openWorkOrderForChecks(db, organizationID, req.GetFactoryId(), req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, failure)
	}

	removed, err := order.RemoveAgentChecks(db, keys)
	if err != nil {
		return nil, factoryErrorToStatus(err, failure)
	}
	if len(removed) > 0 {
		publishCheckUpdate(factoryModel, order)
	}
	return &pb.RemoveWorkOrderChecksResponse{Removed: removed}, nil
}

func openWorkOrderForChecks(
	db *gorm.DB,
	organizationID, factoryRef, orderRef string,
) (*models.Factory, *models.FactoryWorkOrder, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, nil, err
	}
	factoryModel, err := findFactory(db, orgID, factoryRef)
	if err != nil {
		return nil, nil, err
	}
	order, err := findWorkOrder(db, factoryModel, orderRef)
	if err != nil {
		return nil, nil, err
	}
	if order.IsClosed() {
		return nil, nil, grpcerrors.Conflict(nil, "Task is closed or archived")
	}
	return factoryModel, order, nil
}

func checkInputsFromProto(checks []*pb.AddWorkOrderChecksRequest_Check) []models.AgentCheckInput {
	inputs := make([]models.AgentCheckInput, 0, len(checks))
	for _, check := range checks {
		score := check.GetScore()
		maxScore := check.GetMaxScore()
		inputs = append(inputs, models.AgentCheckInput{
			Key:      check.GetKey(),
			Name:     check.GetName(),
			Score:    &score,
			MaxScore: &maxScore,
			Level:    checkLevelFromProto(check.GetLevel()),
			Reason:   check.GetReason(),
		})
	}
	return inputs
}

func publishCheckUpdate(factoryModel *models.Factory, order *models.FactoryWorkOrder) {
	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryModel.ID.String(),
		order.ID.String(),
		factory.EventTypeOrderCheckReported,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}
}
