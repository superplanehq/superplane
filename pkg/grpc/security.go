package grpc

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

// securityChange runs after the service returns, including transaction commit.
// Only identifier paths are read. Configuration and credential fields are excluded.
func securityChange(ctx context.Context, action, targetType string, request, response proto.Message, err error) {
	outcome := "error"
	if err == nil && response != nil && response.ProtoReflect().IsValid() {
		outcome = "success"
	}
	if err != nil {
		outcome = "error"
		if grpcerrors.Code(err) == codes.PermissionDenied || grpcerrors.Code(err) == codes.Unauthenticated {
			outcome = "denied"
		}
	}
	paths := map[string][]string{
		"role":           {"role.metadata.name", "role_name"},
		"group":          {"group.metadata.name", "group_name"},
		"user":           {"user_id", "user_email"},
		"organization":   {"organization.metadata.id", "id"},
		"invitation":     {"invitation.metadata.id", "invitation.id", "invitation_id"},
		"api_key":        {"api_key.metadata.id", "api_key.id", "id"},
		"personal_token": {"token.id", "token_id", "id"},
		"secret":         {"secret.metadata.id", "id_or_name"},
		"integration":    {"integration.metadata.id", "integration.id", "integration_id"},
		"workspace":      {"factory.id", "factory.metadata.id", "id", "factory_id"},
		"automation":     {"automation.id", "custom_automation.automation.id", "automation.metadata.id", "automation_id", "intake_id", "handler_id", "canvas_id", "id"},
		"mcp_client":     {"client_id", "id", "factory_id"},
	}[targetType]
	targetID := ""
	for _, message := range []proto.Message{response, request} {
		for _, path := range paths {
			if value := securityIdentifier(message, path); value != "" {
				targetID = value
				break
			}
		}
		if targetID != "" {
			break
		}
	}
	fields := log.Fields{}
	if result, ok := response.(*structpb.Struct); ok && result != nil && targetType == "organization" {
		targetID = result.GetFields()["organization_id"].GetStringValue()
		fields["organization_id"] = targetID
	}
	for _, key := range []string{"group_name", "role_name", "factory_id"} {
		if value := securityIdentifier(request, key); value != "" {
			fields[key] = value
		}
	}
	if request != nil && request.ProtoReflect().IsValid() {
		message := request.ProtoReflect()
		for _, key := range []string{"is_owner", "visibility"} {
			field := message.Descriptor().Fields().ByName(protoreflect.Name(key))
			if field == nil {
				continue
			}
			if field.Kind() == protoreflect.BoolKind {
				fields[key] = message.Get(field).Bool()
			}
			if field.Kind() == protoreflect.EnumKind {
				fields[key] = message.Get(field).Enum()
			}
		}
	}
	for key, contextKey := range map[string]any{
		"organization_id": authorization.OrganizationContextKey,
		"domain_type":     authorization.DomainTypeContextKey,
		"domain_id":       authorization.DomainIdContextKey,
	} {
		if value, ok := ctx.Value(contextKey).(string); ok && value != "" {
			fields[key] = value
		}
	}
	if targetID == "" && targetType == "organization" {
		targetID, _ = ctx.Value(authorization.DomainIdContextKey).(string)
	}
	if targetID == "" && action == "personal_token.rotate" {
		targetID, _ = authentication.GetUserIdFromMetadata(ctx)
	}
	logging.SecurityEvent(ctx, action, outcome, targetType, targetID, fields)
}

func securityIdentifier(message proto.Message, path string) string {
	if message == nil {
		return ""
	}
	current := message.ProtoReflect()
	parts := strings.Split(path, ".")
	for i, part := range parts {
		if !current.IsValid() {
			return ""
		}
		field := current.Descriptor().Fields().ByName(protoreflect.Name(part))
		if field == nil || field.IsList() || field.IsMap() {
			return ""
		}
		if i == len(parts)-1 {
			if field.Kind() == protoreflect.StringKind {
				return current.Get(field).String()
			}
			return ""
		}
		if field.Kind() != protoreflect.MessageKind {
			return ""
		}
		current = current.Get(field).Message()
	}
	return ""
}
