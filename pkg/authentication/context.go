package authentication

import (
	"context"
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/jwt"
	"google.golang.org/grpc/metadata"
)

const ScopedTokenPermissionsMetadataKey = "x-scoped-token-permissions"

func SetUserIdInMetadata(ctx context.Context, userId string) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.Pairs("x-user-id", userId))
}

func GetUserIdFromMetadata(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	userMeta, ok := md["x-user-id"]
	if !ok || len(userMeta) == 0 {
		return "", false
	}

	return userMeta[0], true
}

func GetOrganizationIdFromMetadata(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	userMeta, ok := md["x-organization-id"]
	if !ok || len(userMeta) == 0 {
		return "", false
	}

	return userMeta[0], true
}

func GetScopedTokenPermissionsFromMetadata(ctx context.Context) ([]jwt.Permission, bool) {
	value, ok := getFirstMetadataValue(ctx, ScopedTokenPermissionsMetadataKey)
	if !ok {
		return nil, false
	}

	var permissions []jwt.Permission
	if err := json.Unmarshal([]byte(value), &permissions); err != nil {
		return nil, false
	}

	if len(permissions) == 0 {
		return nil, false
	}

	return permissions, true
}

func getFirstMetadataValue(ctx context.Context, key string) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	values := md.Get(key)
	if len(values) == 0 || values[0] == "" {
		return "", false
	}

	return values[0], true
}
