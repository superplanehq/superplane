package integrations

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

func CreateIntegration(ctx context.Context, req *pb.CreateIntegrationRequest) (*pb.CreateIntegrationResponse, error) {
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

	integration, err := buildIntegration(canvas, req.Integration)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

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

func buildIntegration(canvas *models.Canvas, integration *pb.Integration) (*models.Integration, error) {
	t, err := validateType(integration.Spec.Type)
	if err != nil {
		return nil, err
	}

	if integration.Spec.Auth == nil {
		return nil, status.Error(codes.InvalidArgument, "auth is required")
	}

	auth, authType, err := validateAuth(integration.Spec.Auth)
	if err != nil {
		return nil, err
	}

	if integration.Spec.Oidc == nil {
		integration.Spec.Oidc = &pb.Integration_OIDC{
			Enabled: false,
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
		OIDC:       datatypes.NewJSONType(models.IntegrationOIDC{Enabled: integration.Spec.Oidc.Enabled}),
	}, nil
}

func validateType(t pb.Integration_Type) (string, error) {
	switch t {
	case pb.Integration_TYPE_SEMAPHORE:
		return models.IntegrationTypeSemaphore, nil
	case pb.Integration_TYPE_GITHUB:
		return models.IntegrationTypeGithub, nil
	default:
		return "", status.Error(codes.InvalidArgument, "invalid integration type")
	}
}

func validateAuth(auth *pb.Integration_Auth) (*models.IntegrationAuth, string, error) {
	switch auth.Use {
	case pb.Integration_AUTH_TYPE_TOKEN:
		if auth.Token == nil || auth.Token.ValueFrom == nil || auth.Token.ValueFrom.Secret == nil {
			return nil, "", fmt.Errorf("token is required")
		}

		// TODO: validate secret existence

		return &models.IntegrationAuth{
			Token: models.IntegrationAuthToken{
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
		return nil, "", status.Error(codes.InvalidArgument, "invalid auth type")
	}
}

func serializeIntegration(integration models.Integration) *pb.Integration {
	return &pb.Integration{
		Metadata: &pb.Integration_Metadata{
			Id:        integration.ID.String(),
			Name:      integration.Name,
			CreatedAt: timestamppb.New(*integration.CreatedAt),
		},
		Spec: &pb.Integration_Spec{
			Type: integrationTypeToProto(integration.Type),
			Url:  integration.URL,
			Auth: &pb.Integration_Auth{
				Use: integrationAuthTypeToProto(integration.AuthType),
			},
			Oidc: &pb.Integration_OIDC{
				Enabled: integration.OIDC.Data().Enabled,
			},
		},
	}
}

func integrationTypeToProto(integrationType string) pb.Integration_Type {
	switch integrationType {
	case "semaphore":
		return pb.Integration_TYPE_SEMAPHORE
	case "github":
		return pb.Integration_TYPE_GITHUB
	default:
		return pb.Integration_TYPE_NONE
	}
}

func integrationAuthTypeToProto(authType string) pb.Integration_AuthType {
	switch authType {
	case "token":
		return pb.Integration_AUTH_TYPE_TOKEN
	case "oidc":
		return pb.Integration_AUTH_TYPE_OIDC
	default:
		return pb.Integration_AUTH_TYPE_NONE
	}
}
