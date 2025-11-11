package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func ListApplications(ctx context.Context, orgID string) (*pb.ListApplicationsResponse, error) {
	appInstallations, err := models.ListAppInstallations(uuid.MustParse(orgID))
	if err != nil {
		return nil, err
	}

	protos := []*pb.AppInstallation{}
	for _, appInstallation := range appInstallations {
		proto, err := serializeAppInstallation(&appInstallation)
		if err != nil {
			return nil, err
		}

		protos = append(protos, proto)
	}

	return &pb.ListApplicationsResponse{
		Applications: protos,
	}, nil
}
