package organizations

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func organizationToProto(organization *models.Organization) *pb.Organization {
	providers := []string(organization.AllowedProviders)
	if providers == nil {
		providers = []string{}
	}
	return &pb.Organization{
		Metadata: &pb.Organization_Metadata{
			Id:          organization.ID.String(),
			Name:        organization.Name,
			Description: organization.Description,
			CreatedAt:   timestamppb.New(*organization.CreatedAt),
			UpdatedAt:   timestamppb.New(*organization.UpdatedAt),
		},
		Spec: &pb.Organization_Spec{
			ChangeManagementEnabled: &organization.ChangeManagementEnabled,
			AllowedOauthProviders: &pb.Organization_AllowedOAuthProviders{
				Providers: providers,
			},
		},
	}
}
