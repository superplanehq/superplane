package organizations

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetOktaSettings(ctx context.Context, orgID string) (*pb.GetOktaSettingsResponse, error) {
	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "organization not found")
	}

	config, err := models.FindOrganizationOktaConfig(org.ID)
	if err != nil {
		// No config yet
		return &pb.GetOktaSettingsResponse{
			Settings: &pb.OktaSettings{
				SamlIssuer:      "",
				SamlCertificate: "",
				EnforceSso:      false,
				HasScimToken:    false,
			},
		}, nil
	}

	return &pb.GetOktaSettingsResponse{
		Settings: &pb.OktaSettings{
			SamlIssuer:      config.SamlIssuer,
			SamlCertificate: config.SamlCertificate,
			EnforceSso:      config.EnforceSSO,
			HasScimToken:    config.ScimTokenHash != "",
		},
	}, nil
}

func UpdateOktaSettings(ctx context.Context, orgID string, settings *pb.OktaSettings) (*pb.UpdateOktaSettingsResponse, error) {
	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "organization not found")
	}

	config, err := models.FindOrganizationOktaConfig(org.ID)
	if err != nil {
		config = &models.OrganizationOktaConfig{
			OrganizationID: org.ID,
		}
	}

	config.SamlIssuer = settings.SamlIssuer
	config.SamlCertificate = settings.SamlCertificate
	config.EnforceSSO = settings.EnforceSso

	if err := models.SaveOrganizationOktaConfig(config); err != nil {
		log.Errorf("Error saving Okta settings for org %s: %v", org.ID, err)
		return nil, status.Error(codes.Internal, "failed to save Okta settings")
	}

	return &pb.UpdateOktaSettingsResponse{}, nil
}

func RotateOktaSCIMToken(ctx context.Context, orgID string) (*pb.RotateOktaSCIMTokenResponse, error) {
	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "organization not found")
	}

	config, err := models.FindOrganizationOktaConfig(org.ID)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "Okta not configured for organization")
	}

	token, err := crypto.Base64String(32)
	if err != nil {
		log.Errorf("Error generating SCIM token for org %s: %v", org.ID, err)
		return nil, status.Error(codes.Internal, "failed to generate SCIM token")
	}

	config.ScimTokenHash = models.HashSCIMToken(token)

	if err := models.SaveOrganizationOktaConfig(config); err != nil {
		log.Errorf("Error saving SCIM token for org %s: %v", org.ID, err)
		return nil, status.Error(codes.Internal, "failed to save SCIM token")
	}

	return &pb.RotateOktaSCIMTokenResponse{
		Token: token,
	}, nil
}
