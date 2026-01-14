package actions

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	defaultInternalMessage = "internal error"
	defaultNotFoundMessage = "resource not found"
)

// ToStatus converts non-status errors into a sanitized gRPC status error.
func ToStatus(err error) error {
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) || looksNotFound(err) {
		return status.Error(codes.NotFound, defaultNotFoundMessage)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return status.Error(codes.Internal, defaultInternalMessage)
	}

	return status.Error(codes.Internal, defaultInternalMessage)
}

func looksNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found")
}
