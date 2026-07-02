package database

import (
	"context"

	gorm "gorm.io/gorm"
)

// DB returns a request-scoped GORM handle so SQL trace hooks can attach to the
// active OpenTelemetry span in ctx.
func DB(ctx context.Context) *gorm.DB {
	return Conn().WithContext(ctx)
}
