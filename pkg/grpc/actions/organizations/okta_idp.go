package organizations

import (
	"context"
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

const oktaOAuthClientSecretCredentialName = "okta_oauth_client_secret"

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
	ctx context.Context,
	encryptor crypto.Encryptor,
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
			issuer := ""
			if req.IssuerBaseUrl != nil {
				issuer = strings.TrimSpace(*req.IssuerBaseUrl)
			}
			clientID := ""
			if req.OauthClientId != nil {
				clientID = strings.TrimSpace(*req.OauthClientId)
			}
			if issuer == "" || clientID == "" {
				return status.Error(codes.InvalidArgument, "issuer_base_url and oauth_client_id are required to create Okta IdP settings")
			}
			if vErr := validateOktaIssuerBaseURL(issuer); vErr != nil {
				return vErr
			}

			now := time.Now()
			row = &models.OrganizationOktaIDP{
				OrganizationID: orgUUID,
				IssuerBaseURL:  issuer,
				OAuthClientID:  clientID,
				OIDCEnabled:    false,
				ScimEnabled:    false,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if req.OidcEnabled != nil {
				row.OIDCEnabled = *req.OidcEnabled
			}
			if req.ScimEnabled != nil {
				row.ScimEnabled = *req.ScimEnabled
			}
			if req.OauthClientSecret != nil {
				secret := strings.TrimSpace(*req.OauthClientSecret)
				if secret != "" {
					ct, encErr := encryptor.Encrypt(ctx, []byte(secret), []byte(oktaOAuthClientSecretCredentialName))
					if encErr != nil {
						return status.Error(codes.Internal, "failed to encrypt OAuth client secret")
					}
					row.OAuthClientSecretCiphertext = ct
				}
			}

			if e := models.CreateOrganizationOktaIDPInTransaction(tx, row); e != nil {
				return status.Error(codes.Internal, "failed to save Okta IdP settings")
			}
			out = row
			return nil
		}

		if req.IssuerBaseUrl != nil {
			issuer := strings.TrimSpace(*req.IssuerBaseUrl)
			if issuer == "" {
				return status.Error(codes.InvalidArgument, "issuer_base_url cannot be empty")
			}
			if vErr := validateOktaIssuerBaseURL(issuer); vErr != nil {
				return vErr
			}
			row.IssuerBaseURL = issuer
		}
		if req.OauthClientId != nil {
			cid := strings.TrimSpace(*req.OauthClientId)
			if cid == "" {
				return status.Error(codes.InvalidArgument, "oauth_client_id cannot be empty")
			}
			row.OAuthClientID = cid
		}
		if req.OauthClientSecret != nil {
			secret := strings.TrimSpace(*req.OauthClientSecret)
			if secret != "" {
				ct, encErr := encryptor.Encrypt(ctx, []byte(secret), []byte(oktaOAuthClientSecretCredentialName))
				if encErr != nil {
					return status.Error(codes.Internal, "failed to encrypt OAuth client secret")
				}
				row.OAuthClientSecretCiphertext = ct
			}
		}
		if req.OidcEnabled != nil {
			row.OIDCEnabled = *req.OidcEnabled
		}
		if req.ScimEnabled != nil {
			row.ScimEnabled = *req.ScimEnabled
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
	secretConfigured := len(row.OAuthClientSecretCiphertext) > 0
	scimTokenConfigured := row.ScimBearerTokenHash != nil && *row.ScimBearerTokenHash != ""
	return &pb.OktaIdpSettings{
		OrganizationId:              row.OrganizationID.String(),
		Configured:                  true,
		IssuerBaseUrl:               row.IssuerBaseURL,
		OauthClientId:               row.OAuthClientID,
		OauthClientSecretConfigured: secretConfigured,
		OidcEnabled:                 row.OIDCEnabled,
		ScimEnabled:                 row.ScimEnabled,
		ScimBearerTokenConfigured:   scimTokenConfigured,
		CreatedAt:                   timestamppb.New(row.CreatedAt),
		UpdatedAt:                   timestamppb.New(row.UpdatedAt),
	}
}

func validateOktaIssuerBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return status.Error(codes.InvalidArgument, "issuer_base_url must be a valid https URL")
	}
	return nil
}
