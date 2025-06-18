package authentication

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
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

// MustGetUserFromContext retrieves the user from context, panics if not found
func MustGetUserFromContext(ctx context.Context) *models.User {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		panic("user not found in context")
	}
	return user
}
