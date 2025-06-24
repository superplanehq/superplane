package authentication

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/metadata"
)

type contextKey string

const userContextKey contextKey = "user"

// SetUserInContext adds a user to the request context
func SetUserInContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetUserFromContext retrieves the authenticated user from context
func GetUserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userContextKey).(*models.User)
	return user, ok
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

// MustGetUserFromContext retrieves the user from context, panics if not found
func MustGetUserFromContext(ctx context.Context) *models.User {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		panic("user not found in context")
	}
	return user
}
