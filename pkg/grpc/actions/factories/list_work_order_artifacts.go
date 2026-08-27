package factories

import (
	"context"
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListWorkOrderArtifacts(
	ctx context.Context,
	organizationID string,
	req *pb.ListWorkOrderArtifactsRequest,
) (*pb.ListWorkOrderArtifactsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order artifacts")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order artifacts")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order artifacts")
	}

	db := database.DB(ctx)
	factoryModel, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order artifacts")
	}

	order, err := factoryModel.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order artifacts")
	}

	artifacts, err := order.ListArtifacts(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order artifacts")
	}

	serialized, err := serializeArtifacts(artifacts)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order artifacts")
	}

	return &pb.ListWorkOrderArtifactsResponse{Artifacts: serialized}, nil
}

func serializeArtifacts(artifacts []models.FactoryWorkOrderArtifact) ([]*pb.WorkOrderArtifact, error) {
	result := make([]*pb.WorkOrderArtifact, 0, len(artifacts))
	for i := range artifacts {
		serialized, err := serializeArtifact(&artifacts[i])
		if err != nil {
			return nil, err
		}
		result = append(result, serialized)
	}
	return result, nil
}

func serializeArtifact(artifact *models.FactoryWorkOrderArtifact) (*pb.WorkOrderArtifact, error) {
	data, err := decodeArtifactStruct(artifact.Data)
	if err != nil {
		return nil, err
	}

	return &pb.WorkOrderArtifact{
		Id:        artifact.ID.String(),
		Type:      artifactTypeToProto(artifact.Type),
		Data:      data,
		CreatedBy: serializeArtifactCreator(artifact),
		CreatedAt: timestamppb.New(artifact.CreatedAt),
	}, nil
}

func serializeArtifactCreator(artifact *models.FactoryWorkOrderArtifact) *pb.UserRef {
	if artifact.CreatedByID == nil {
		return nil
	}

	name := artifact.CreatedByID.String()
	if artifact.CreatedBy != nil {
		name = artifact.CreatedBy.Name
	}

	return &pb.UserRef{Id: artifact.CreatedByID.String(), Name: name}
}

func artifactTypeToProto(t string) pb.WorkOrderArtifact_Type {
	switch t {
	case models.FactoryWorkOrderArtifactTypeMarkdown:
		return pb.WorkOrderArtifact_TYPE_MARKDOWN
	case models.FactoryWorkOrderArtifactTypeBranch:
		return pb.WorkOrderArtifact_TYPE_BRANCH
	case models.FactoryWorkOrderArtifactTypeLink:
		return pb.WorkOrderArtifact_TYPE_LINK
	default:
		return pb.WorkOrderArtifact_TYPE_UNSPECIFIED
	}
}

func artifactTypeFromProto(t pb.WorkOrderArtifact_Type) (string, bool) {
	switch t {
	case pb.WorkOrderArtifact_TYPE_MARKDOWN:
		return models.FactoryWorkOrderArtifactTypeMarkdown, true
	case pb.WorkOrderArtifact_TYPE_BRANCH:
		return models.FactoryWorkOrderArtifactTypeBranch, true
	case pb.WorkOrderArtifact_TYPE_LINK:
		return models.FactoryWorkOrderArtifactTypeLink, true
	}
	return "", false
}

func decodeArtifactStruct(raw []byte) (*structpb.Struct, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	return structpb.NewStruct(data)
}
