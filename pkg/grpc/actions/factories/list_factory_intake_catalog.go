package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListFactoryIntakeCatalog(
	ctx context.Context,
	organizationID string,
	_ *pb.ListFactoryIntakeCatalogRequest,
) (*pb.ListFactoryIntakeCatalogResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list intake catalog")
	}

	catalog, err := models.ListIntakeCatalogForOrganization(database.DB(ctx), orgID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list intake catalog")
	}

	entries := make([]*pb.FactoryIntakeCatalogEntry, 0, len(catalog))
	for _, item := range catalog {
		entries = append(entries, &pb.FactoryIntakeCatalogEntry{
			Key:        item.Entry.Key,
			Name:       item.Entry.Name,
			Category:   item.Entry.Category,
			Status:     item.Entry.Status,
			StatusNote: item.Entry.StatusNote,
			Available:  item.Available,
		})
	}
	return &pb.ListFactoryIntakeCatalogResponse{Entries: entries}, nil
}
