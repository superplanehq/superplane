package organizations

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListOrganizations(ctx context.Context, req *pb.ListOrganizationsRequest) (*pb.ListOrganizationsResponse, error) {
	organizations, err := models.ListOrganizations()
	if err != nil {
		return nil, err
	}

	response := &pb.ListOrganizationsResponse{
		Organizations: serializeOrganizations(organizations),
	}

	return response, nil
}

func serializeOrganizations(in []models.Organization) []*pb.Organization {
	out := []*pb.Organization{}
	for _, organization := range in {
		out = append(out, &pb.Organization{
			Metadata: &pb.Organization_Metadata{
				Id:          organization.ID.String(),
				Name:        organization.Name,
				DisplayName: organization.DisplayName,
				CreatedBy:   organization.CreatedBy.String(),
				CreatedAt:   timestamppb.New(*organization.CreatedAt),
				UpdatedAt:   timestamppb.New(*organization.UpdatedAt),
			},
		})
	}

	return out
}
