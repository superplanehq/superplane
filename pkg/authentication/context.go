package authentication

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/metadata"
)

const TokenScopesMetadataKey = "x-token-scopes"
const SupportSessionMetadataKey = "x-support-session"

func SetSupportSessionInMetadata(ctx context.Context) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.Pairs(SupportSessionMetadataKey, "true"))
}

// IsSupportSessionFromMetadata reports whether the gateway marked the
// request as part of a read-only support session.
func IsSupportSessionFromMetadata(ctx context.Context) bool {
	value, ok := getFirstMetadataValue(ctx, SupportSessionMetadataKey)
	return ok && value == "true"
}

func SetAccountIDInMetadata(ctx context.Context, accountID string) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.Pairs("x-account-id", accountID))
}

func GetAccountIDFromMetadata(ctx context.Context) (string, bool) {
	return getFirstMetadataValue(ctx, "x-account-id")
}

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

func GetScopedTokenScopesFromMetadata(ctx context.Context) ([]string, bool) {
	value, ok := getFirstMetadataValue(ctx, TokenScopesMetadataKey)
	if !ok {
		return nil, false
	}

	var scopes []string
	if err := json.Unmarshal([]byte(value), &scopes); err != nil {
		return nil, false
	}

	if len(scopes) == 0 {
		return nil, false
	}

	return scopes, true
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
