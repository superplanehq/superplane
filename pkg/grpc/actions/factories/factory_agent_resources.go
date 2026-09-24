package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListFactoryAgentResources(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryAgentResourcesRequest,
) (*pb.ListFactoryAgentResourcesResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent resources")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent resources")
	}

	kind := protoKindToModel(req.GetKind())
	if err := requireAgentResourceKindFeature(orgID, kind); err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent resources")
	}
	resources, err := factory.ListAgentResources(db, kind)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent resources")
	}

	out := make([]*pb.FactoryAgentResource, 0, len(resources))
	for i := range resources {
		out = append(out, serializeFactoryAgentResource(&resources[i]))
	}
	return &pb.ListFactoryAgentResourcesResponse{Resources: out}, nil
}

func CreateFactoryAgentResource(
	ctx context.Context,
	organizationID string,
	req *pb.CreateFactoryAgentResourceRequest,
) (*pb.CreateFactoryAgentResourceResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent resource")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent resource")
	}
	kind := protoKindToModel(req.GetKind())
	if err := requireAgentResourceKindFeature(orgID, kind); err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent resource")
	}

	config, err := protoAgentResourceConfig(req.GetKind(), req.GetUrl(), req.GetAuth(), req.GetHeaders(), req.GetMarkdown())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent resource")
	}
	resource, err := factory.CreateAgentResource(
		db,
		protoKindToModel(req.GetKind()),
		req.GetName(),
		req.GetEnabled(),
		config,
	)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent resource")
	}
	return &pb.CreateFactoryAgentResourceResponse{Resource: serializeFactoryAgentResource(resource)}, nil
}

func UpdateFactoryAgentResource(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.UpdateFactoryAgentResourceRequest,
) (*pb.UpdateFactoryAgentResourceResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update agent resource")
	}
	resourceID, err := uuid.Parse(strings.TrimSpace(req.GetResourceId()))
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid resource id"), "failed to update agent resource")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update agent resource")
	}
	resource, err := factory.FindAgentResource(db, resourceID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update agent resource")
	}
	if err := requireAgentResourceKindFeature(orgID, resource.Kind); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update agent resource")
	}

	var name *string
	if req.Name != nil {
		value := req.GetName()
		name = &value
	}
	var enabled *bool
	if req.Enabled != nil {
		value := req.GetEnabled()
		enabled = &value
	}
	var config *models.FactoryAgentResourceConfig
	if resource.Kind == models.FactoryAgentResourceKindSkill {
		if req.Markdown != nil {
			merged := resource.Config.Data()
			merged.Markdown = req.GetMarkdown()
			merged.Source = models.FactoryAgentResourceSourceInline
			config = &merged
		}
	} else if req.Url != nil || req.Auth != nil || len(req.GetHeaders()) > 0 || req.GetReplaceDisabledTools() {
		merged := resource.Config.Data()
		if req.Url != nil {
			merged.URL = req.GetUrl()
		}
		if req.Auth != nil {
			merged.Auth = protoAuthToModel(req.GetAuth())
		}
		if len(req.GetHeaders()) > 0 || (req.Auth != nil && req.GetAuth() == pb.FactoryAgentResource_AUTH_OAUTH) {
			merged.Headers = protoHeadersToModel(req.GetHeaders())
		}
		if req.GetReplaceDisabledTools() {
			merged.DisabledTools = models.NormalizeDisabledTools(req.GetDisabledTools())
		}
		if err := validateMCPURL(merged.URL); err != nil {
			return nil, factoryErrorToStatus(err, "failed to update agent resource")
		}
		config = &merged
	}

	var revocation *oauthRevocation
	if config != nil && resource.Config.Data().InvalidatesOAuth(*config) {
		revocation = captureOAuthRevocation(ctx, deps, db, resource)
	}
	if err := resource.Update(db, name, enabled, config); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update agent resource")
	}
	revocation.run(ctx, deps)
	return &pb.UpdateFactoryAgentResourceResponse{Resource: serializeFactoryAgentResource(resource)}, nil
}

func DeleteFactoryAgentResource(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.DeleteFactoryAgentResourceRequest,
) (*pb.DeleteFactoryAgentResourceResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete agent resource")
	}
	resourceID, err := uuid.Parse(strings.TrimSpace(req.GetResourceId()))
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid resource id"), "failed to delete agent resource")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete agent resource")
	}
	resource, err := factory.FindAgentResource(db, resourceID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete agent resource")
	}
	if err := requireAgentResourceKindFeature(orgID, resource.Kind); err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete agent resource")
	}
	revocation := captureOAuthRevocation(ctx, deps, db, resource)
	if err := resource.Delete(db); err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete agent resource")
	}
	revocation.run(ctx, deps)
	return &pb.DeleteFactoryAgentResourceResponse{}, nil
}

func serializeFactoryAgentResource(resource *models.FactoryAgentResource) *pb.FactoryAgentResource {
	config := resource.Config.Data()
	out := &pb.FactoryAgentResource{
		Id:            resource.ID.String(),
		FactoryId:     resource.FactoryID.String(),
		Kind:          modelKindToProto(resource.Kind),
		Name:          resource.Name,
		Enabled:       resource.Enabled,
		Url:           config.URL,
		Auth:          modelAuthToProto(config.MCPAuth()),
		Headers:       modelHeadersToProto(config.Headers),
		DisabledTools: config.DisabledTools,
		OauthError:    resource.OAuthError,
		CreatedAt:     timestamppb.New(resource.CreatedAt),
		UpdatedAt:     timestamppb.New(resource.UpdatedAt),
	}
	if config.MCPAuth() == models.FactoryAgentResourceAuthOAuth {
		out.OauthStatus = modelOAuthStatusToProto(resource.OAuthState())
	}
	if resource.Kind == models.FactoryAgentResourceKindSkill {
		out.Repository = config.Repository
		out.Ref = config.Ref
		out.Path = config.Path
		out.Markdown = config.Markdown
	}
	if resource.OAuthConnectedBy != nil {
		out.OauthConnectedByUserId = resource.OAuthConnectedBy.String()
	}
	if resource.OAuthConnectedAt != nil {
		out.OauthConnectedAt = timestamppb.New(*resource.OAuthConnectedAt)
	}
	return out
}

func protoAgentResourceConfig(
	kind pb.FactoryAgentResource_Kind,
	rawURL string,
	auth pb.FactoryAgentResource_Auth,
	headers []*pb.FactoryAgentResource_Header,
	markdown string,
) (models.FactoryAgentResourceConfig, error) {
	if kind == pb.FactoryAgentResource_KIND_SKILL {
		return models.FactoryAgentResourceConfig{
			Source:   models.FactoryAgentResourceSourceInline,
			Markdown: markdown,
		}, nil
	}
	return protoMCPConfig(rawURL, auth, headers)
}

func protoMCPConfig(rawURL string, auth pb.FactoryAgentResource_Auth, headers []*pb.FactoryAgentResource_Header) (models.FactoryAgentResourceConfig, error) {
	config := models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       strings.TrimSpace(rawURL),
		Auth:      protoAuthToModel(auth),
		Headers:   protoHeadersToModel(headers),
	}
	if err := validateMCPURL(config.URL); err != nil {
		return models.FactoryAgentResourceConfig{}, err
	}
	return config, nil
}

func validateMCPURL(rawURL string) error {
	if err := mcp.ValidatePublicHTTPSURL(rawURL); err != nil {
		return invalidArgument(err.Error())
	}
	return nil
}

func protoKindToModel(kind pb.FactoryAgentResource_Kind) string {
	switch kind {
	case pb.FactoryAgentResource_KIND_SKILL:
		return models.FactoryAgentResourceKindSkill
	case pb.FactoryAgentResource_KIND_MCP_SERVER, pb.FactoryAgentResource_KIND_UNSPECIFIED:
		return models.FactoryAgentResourceKindMCPServer
	default:
		return models.FactoryAgentResourceKindMCPServer
	}
}

func modelKindToProto(kind string) pb.FactoryAgentResource_Kind {
	if kind == models.FactoryAgentResourceKindSkill {
		return pb.FactoryAgentResource_KIND_SKILL
	}
	return pb.FactoryAgentResource_KIND_MCP_SERVER
}

func requireAgentResourceKindFeature(orgID uuid.UUID, kind string) error {
	featureID := features.FeatureWorkspaceMCP
	if kind == models.FactoryAgentResourceKindSkill {
		featureID = features.FeatureWorkspaceSkills
	}
	enabled, err := models.HasExperimentalFeature(orgID, featureID)
	if err != nil {
		return err
	}
	if enabled {
		return nil
	}
	if kind == models.FactoryAgentResourceKindSkill {
		return errWorkspaceSkillsDisabled
	}
	return errWorkspaceMCPDisabled
}

func protoAuthToModel(auth pb.FactoryAgentResource_Auth) string {
	if auth == pb.FactoryAgentResource_AUTH_OAUTH {
		return models.FactoryAgentResourceAuthOAuth
	}
	return models.FactoryAgentResourceAuthHeaders
}

func modelAuthToProto(auth string) pb.FactoryAgentResource_Auth {
	if auth == models.FactoryAgentResourceAuthOAuth {
		return pb.FactoryAgentResource_AUTH_OAUTH
	}
	return pb.FactoryAgentResource_AUTH_HEADERS
}

func protoHeadersToModel(headers []*pb.FactoryAgentResource_Header) []models.FactoryAgentResourceHeader {
	out := make([]models.FactoryAgentResourceHeader, 0, len(headers))
	for _, header := range headers {
		if header == nil {
			continue
		}
		out = append(out, models.FactoryAgentResourceHeader{
			Name:       strings.TrimSpace(header.GetName()),
			SecretName: strings.TrimSpace(header.GetSecretName()),
			SecretKey:  strings.TrimSpace(header.GetSecretKey()),
		})
	}
	return out
}

func modelHeadersToProto(headers []models.FactoryAgentResourceHeader) []*pb.FactoryAgentResource_Header {
	out := make([]*pb.FactoryAgentResource_Header, 0, len(headers))
	for _, header := range headers {
		out = append(out, &pb.FactoryAgentResource_Header{
			Name:       header.Name,
			SecretName: header.SecretName,
			SecretKey:  header.SecretKey,
		})
	}
	return out
}

func modelOAuthStatusToProto(status string) pb.FactoryAgentResource_OAuthStatus {
	switch status {
	case models.FactoryAgentResourceOAuthConnected:
		return pb.FactoryAgentResource_OAUTH_STATUS_CONNECTED
	case models.FactoryAgentResourceOAuthNeedsReconnect:
		return pb.FactoryAgentResource_OAUTH_STATUS_NEEDS_RECONNECT
	case models.FactoryAgentResourceOAuthVendorRejected:
		return pb.FactoryAgentResource_OAUTH_STATUS_VENDOR_REJECTED
	case models.FactoryAgentResourceOAuthNotConnected:
		return pb.FactoryAgentResource_OAUTH_STATUS_NOT_CONNECTED
	default:
		return pb.FactoryAgentResource_OAUTH_STATUS_UNSPECIFIED
	}
}
