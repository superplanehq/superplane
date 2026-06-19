package grpc

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	sanitizedInternalMessage = "internal error"
	sanitizedNotFoundMessage = "resource not found"
)

func sanitizeError(err error) error {
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) || looksNotFound(err) {
		return status.Error(codes.NotFound, sanitizedNotFoundMessage)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return status.Error(codes.Internal, sanitizedInternalMessage)
	}

	return status.Error(codes.Internal, sanitizedInternalMessage)
}

func looksNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found")
}
