package organizations

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func GetOktaIdpSettings(orgID string) (*pb.GetOktaIdpSettingsResponse, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization id")
	}

	row, dbErr := models.FindOrganizationOktaIDPByOrganizationID(orgID)
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return &pb.GetOktaIdpSettingsResponse{
				Settings: &pb.OktaIdpSettings{
					OrganizationId: orgUUID.String(),
					Configured:     false,
				},
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to load Okta IdP settings")
	}

	return &pb.GetOktaIdpSettingsResponse{
		Settings: serializeOktaIdpSettings(row),
	}, nil
}

func UpdateOktaIdpSettings(
	orgID string,
	req *pb.UpdateOktaIdpSettingsRequest,
) (*pb.UpdateOktaIdpSettingsResponse, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization id")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}

	var out *models.OrganizationOktaIDP
	txErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		row, e := models.FindOrganizationOktaIDPByOrganizationIDInTransaction(tx, orgID)
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return status.Error(codes.Internal, "failed to load Okta IdP settings")
		}

		if errors.Is(e, gorm.ErrRecordNotFound) {
			ssoURL := ""
			if req.SamlIdpSsoUrl != nil {
				ssoURL = strings.TrimSpace(*req.SamlIdpSsoUrl)
			}
			issuer := ""
			if req.SamlIdpIssuer != nil {
				issuer = strings.TrimSpace(*req.SamlIdpIssuer)
			}
			if ssoURL == "" || issuer == "" {
				return status.Error(codes.InvalidArgument, "saml_idp_sso_url and saml_idp_issuer are required to create Okta IdP settings")
			}
			if vErr := validateOktaSSOURL(ssoURL); vErr != nil {
				return vErr
			}

			now := time.Now()
			row = &models.OrganizationOktaIDP{
				OrganizationID: orgUUID,
				SamlIdpSSOURL:  ssoURL,
				SamlIdpIssuer:  issuer,
				SamlEnabled:    false,
				ScimEnabled:    false,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if req.SamlIdpCertificatePem != nil {
				row.SamlIdpCertificatePEM = strings.TrimSpace(*req.SamlIdpCertificatePem)
			}
			if req.SamlEnabled != nil {
				row.SamlEnabled = *req.SamlEnabled
			}
			if req.ScimEnabled != nil {
				row.ScimEnabled = *req.ScimEnabled
			}

			if e := syncOktaProviderWhenSAMLEnabled(tx, orgID, row); e != nil {
				return e
			}
			if e := models.CreateOrganizationOktaIDPInTransaction(tx, row); e != nil {
				return status.Error(codes.Internal, "failed to save Okta IdP settings")
			}
			out = row
			return nil
		}

		if req.SamlIdpSsoUrl != nil {
			ssoURL := strings.TrimSpace(*req.SamlIdpSsoUrl)
			if ssoURL == "" {
				return status.Error(codes.InvalidArgument, "saml_idp_sso_url cannot be empty")
			}
			if vErr := validateOktaSSOURL(ssoURL); vErr != nil {
				return vErr
			}
			row.SamlIdpSSOURL = ssoURL
		}
		if req.SamlIdpIssuer != nil {
			issuer := strings.TrimSpace(*req.SamlIdpIssuer)
			if issuer == "" {
				return status.Error(codes.InvalidArgument, "saml_idp_issuer cannot be empty")
			}
			row.SamlIdpIssuer = issuer
		}
		if req.SamlIdpCertificatePem != nil {
			cert := strings.TrimSpace(*req.SamlIdpCertificatePem)
			if cert != "" {
				row.SamlIdpCertificatePEM = cert
			}
		}
		if req.SamlEnabled != nil {
			row.SamlEnabled = *req.SamlEnabled
		}
		if req.ScimEnabled != nil {
			row.ScimEnabled = *req.ScimEnabled
		}

		if e := syncOktaProviderWhenSAMLEnabled(tx, orgID, row); e != nil {
			return e
		}
		if e := models.SaveOrganizationOktaIDPInTransaction(tx, row); e != nil {
			return status.Error(codes.Internal, "failed to save Okta IdP settings")
		}
		out = row
		return nil
	})
	if txErr != nil {
		if s, ok := status.FromError(txErr); ok {
			return nil, s.Err()
		}
		return nil, txErr
	}

	return &pb.UpdateOktaIdpSettingsResponse{
		Settings: serializeOktaIdpSettings(out),
	}, nil
}

func RotateOktaScimBearerToken(orgID string) (*pb.RotateOktaScimBearerTokenResponse, error) {
	_, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization id")
	}

	token, err := crypto.Base64String(32)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate SCIM token")
	}
	hash := crypto.HashToken(token)

	var out *models.OrganizationOktaIDP
	txErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		row, e := models.FindOrganizationOktaIDPByOrganizationIDInTransaction(tx, orgID)
		if e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "Okta IdP is not configured for this organization")
			}
			return status.Error(codes.Internal, "failed to load Okta IdP settings")
		}
		row.ScimBearerTokenHash = &hash
		if e := models.SaveOrganizationOktaIDPInTransaction(tx, row); e != nil {
			return status.Error(codes.Internal, "failed to save SCIM token")
		}
		out = row
		return nil
	})
	if txErr != nil {
		if s, ok := status.FromError(txErr); ok {
			return nil, s.Err()
		}
		return nil, txErr
	}

	return &pb.RotateOktaScimBearerTokenResponse{
		ScimBearerToken: token,
		Settings:        serializeOktaIdpSettings(out),
	}, nil
}

func serializeOktaIdpSettings(row *models.OrganizationOktaIDP) *pb.OktaIdpSettings {
	certConfigured := len(row.SamlIdpCertificatePEM) > 0
	scimTokenConfigured := row.ScimBearerTokenHash != nil && *row.ScimBearerTokenHash != ""
	return &pb.OktaIdpSettings{
		OrganizationId:               row.OrganizationID.String(),
		Configured:                   true,
		SamlIdpSsoUrl:                row.SamlIdpSSOURL,
		SamlIdpIssuer:                row.SamlIdpIssuer,
		SamlIdpCertificateConfigured: certConfigured,
		SamlEnabled:                  row.SamlEnabled,
		ScimEnabled:                  row.ScimEnabled,
		ScimBearerTokenConfigured:    scimTokenConfigured,
		CreatedAt:                    timestamppb.New(row.CreatedAt),
		UpdatedAt:                    timestamppb.New(row.UpdatedAt),
	}
}

func validateOktaSSOURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return status.Error(codes.InvalidArgument, "saml_idp_sso_url must be a valid https URL")
	}
	return nil
}

func syncOktaProviderWhenSAMLEnabled(tx *gorm.DB, orgID string, row *models.OrganizationOktaIDP) error {
	if !row.SamlEnabled {
		return nil
	}
	if row.SamlIdpCertificatePEM == "" {
		return status.Error(codes.FailedPrecondition, "configure SAML certificate before enabling Okta SAML sign-in")
	}
	return models.EnsureOrganizationAllowsProviderInTransaction(tx, orgID, models.ProviderOkta)
}
