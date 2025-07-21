package integrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

func CreateIntegration(ctx context.Context, encryptor crypto.Encryptor, req *pb.CreateIntegrationRequest) (*pb.CreateIntegrationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	err := actions.ValidateUUIDs(req.CanvasIdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	//
	// Validate request
	//
	if req.Integration == nil || req.Integration.Metadata == nil || req.Integration.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "integration name is required")
	}

	integration, err := buildIntegration(ctx, encryptor, canvas, req.Integration)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	integration.CreatedBy = uuid.MustParse(userID)
	integration, err = models.CreateIntegration(integration)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating integration. Request: %v. Error: %v", req, err)
		return nil, err
	}

	response := &pb.CreateIntegrationResponse{
		Integration: serializeIntegration(*integration),
	}

	return response, nil
}

func buildIntegration(ctx context.Context, encryptor crypto.Encryptor, canvas *models.Canvas, integration *pb.Integration) (*models.Integration, error) {
	t, err := validateType(integration.Spec.Type)
	if err != nil {
		return nil, err
	}

	if integration.Spec.Auth == nil {
		return nil, fmt.Errorf("auth is required")
	}

	auth, authType, err := validateAuth(ctx, encryptor, canvas, integration.Spec.Auth)
	if err != nil {
		return nil, err
	}

	if integration.Spec.Oidc == nil {
		integration.Spec.Oidc = &pb.Integration_OIDC{
			Supported: false,
		}
	}

	return &models.Integration{
		Name:       integration.Metadata.Name,
		DomainType: authorization.DomainCanvas,
		DomainID:   canvas.ID,
		Type:       t,
		URL:        integration.Spec.Url,
		AuthType:   authType,
		Auth:       datatypes.NewJSONType(*auth),
		OIDC:       datatypes.NewJSONType(models.IntegrationOIDC{Supported: integration.Spec.Oidc.Supported}),
	}, nil
}

func validateType(t pb.Integration_Type) (string, error) {
	switch t {
	case pb.Integration_TYPE_SEMAPHORE:
		return models.IntegrationTypeSemaphore, nil
	case pb.Integration_TYPE_GITHUB:
		return models.IntegrationTypeGithub, nil
	default:
		return "", fmt.Errorf("invalid integration type")
	}
}

func validateAuth(ctx context.Context, encryptor crypto.Encryptor, canvas *models.Canvas, auth *pb.Integration_Auth) (*models.IntegrationAuth, string, error) {
	switch auth.Use {
	case pb.Integration_AUTH_TYPE_TOKEN:
		if auth.Token == nil || auth.Token.ValueFrom == nil || auth.Token.ValueFrom.Secret == nil {
			return nil, "", fmt.Errorf("secret is required")
		}

		name := auth.Token.ValueFrom.Secret.Name
		provider, err := secrets.NewProvider(encryptor, name, canvas.ID.String())
		if err != nil {
			return nil, "", err
		}

		values, err := provider.Load(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("error loading values for secret %s: %v", name, err)
		}

		key := auth.Token.ValueFrom.Secret.Key
		_, ok := values[key]
		if !ok {
			return nil, "", fmt.Errorf("key %s not found in secret %s", key, name)
		}

		return &models.IntegrationAuth{
			Token: &models.IntegrationAuthToken{
				ValueFrom: models.ValueDefinitionFrom{
					Secret: &models.ValueDefinitionFromSecret{
						Name: auth.Token.ValueFrom.Secret.Name,
						Key:  auth.Token.ValueFrom.Secret.Key,
					},
				},
			},
		}, models.IntegrationAuthTypeToken, nil

	case pb.Integration_AUTH_TYPE_OIDC:
		return nil, models.IntegrationAuthTypeOIDC, nil

	default:
		return nil, "", fmt.Errorf("invalid auth type")
	}
}

func serializeIntegration(integration models.Integration) *pb.Integration {
	return &pb.Integration{
		Metadata: &pb.Integration_Metadata{
			Id:         integration.ID.String(),
			Name:       integration.Name,
			DomainType: actions.DomainTypeToProto(integration.DomainType),
			DomainId:   integration.DomainID.String(),
			CreatedAt:  timestamppb.New(*integration.CreatedAt),
			CreatedBy:  integration.CreatedBy.String(),
		},
		Spec: &pb.Integration_Spec{
			Type: integrationTypeToProto(integration.Type),
			Url:  integration.URL,
			Auth: serializeIntegrationAuth(integration.AuthType, integration.Auth.Data()),
			Oidc: &pb.Integration_OIDC{
				Supported: integration.OIDC.Data().Supported,
			},
		},
	}
}

func serializeIntegrationAuth(authType string, auth models.IntegrationAuth) *pb.Integration_Auth {
	switch authType {
	case models.IntegrationAuthTypeToken:
		return &pb.Integration_Auth{
			Use: integrationAuthTypeToProto(authType),
			Token: &pb.Integration_Auth_Token{
				ValueFrom: &pb.ValueFrom{
					Secret: &pb.ValueFromSecret{
						Name: auth.Token.ValueFrom.Secret.Name,
						Key:  auth.Token.ValueFrom.Secret.Key,
					},
				},
			},
		}
	case models.IntegrationAuthTypeOIDC:
		return &pb.Integration_Auth{
			Use: pb.Integration_AUTH_TYPE_OIDC,
		}
	default:
		return nil
	}
}

func integrationTypeToProto(integrationType string) pb.Integration_Type {
	switch integrationType {
	case models.IntegrationTypeSemaphore:
		return pb.Integration_TYPE_SEMAPHORE
	case models.IntegrationTypeGithub:
		return pb.Integration_TYPE_GITHUB
	default:
		return pb.Integration_TYPE_NONE
	}
}

func integrationAuthTypeToProto(authType string) pb.Integration_AuthType {
	switch authType {
	case models.IntegrationAuthTypeToken:
		return pb.Integration_AUTH_TYPE_TOKEN
	case models.IntegrationAuthTypeOIDC:
		return pb.Integration_AUTH_TYPE_OIDC
	default:
		return pb.Integration_AUTH_TYPE_NONE
	}
}
