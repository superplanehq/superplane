package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	configassistant "github.com/superplanehq/superplane/pkg/grpc/actions/configassistant"
	"github.com/superplanehq/superplane/pkg/jwt"
	pb "github.com/superplanehq/superplane/pkg/protos/config_assistant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ConfigAssistantService struct {
	authService authorization.Authorization
	jwtSigner   *jwt.Signer
}

func NewConfigAssistantService(authService authorization.Authorization, jwtSigner *jwt.Signer) *ConfigAssistantService {
	return &ConfigAssistantService{
		authService: authService,
		jwtSigner:   jwtSigner,
	}
}

func (s *ConfigAssistantService) SuggestConfigurationField(
	ctx context.Context,
	req *pb.SuggestConfigurationFieldRequest,
) (*pb.SuggestConfigurationFieldResponse, error) {
	configAssistantHTTPURL := config.ConfigAssistantHTTPURL()
	if configAssistantHTTPURL == "" {
		return nil, status.Error(codes.Unavailable, "CONFIG_ASSISTANT_HTTP_URL is not set")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return configassistant.SuggestConfigurationField(
		ctx,
		s.authService,
		s.jwtSigner,
		configAssistantHTTPURL,
		userID,
		organizationID,
		req,
	)
}
