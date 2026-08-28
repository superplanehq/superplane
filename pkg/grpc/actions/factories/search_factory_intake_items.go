package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func SearchFactoryIntakeItems(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.SearchFactoryIntakeItemsRequest,
) (*pb.SearchFactoryIntakeItemsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search factory intake items")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search factory intake items")
	}

	intakeID, err := parseIntakeID(req.GetIntakeId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search factory intake items")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search factory intake items")
	}

	intake, err := factory.FindIntake(db, intakeID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search factory intake items")
	}

	source, err := deps.itemSource(ctx, db, intake)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search factory intake items")
	}

	items, err := source.Search(ctx, req.GetQuery(), intakeItemLimit(req.GetQuery(), int(req.GetLimit())))
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search factory intake items")
	}

	serialized := make([]*pb.FactoryIntakeItem, 0, len(items))
	for _, item := range items {
		serialized = append(serialized, serializeFactoryIntakeItem(item))
	}

	return &pb.SearchFactoryIntakeItemsResponse{Items: serialized}, nil
}

func serializeFactoryIntakeItem(item IntakeItem) *pb.FactoryIntakeItem {
	return &pb.FactoryIntakeItem{
		Id:    item.ID,
		Key:   item.Key,
		Title: item.Title,
		Body:  item.Body,
		Url:   item.URL,
	}
}
