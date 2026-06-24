package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	sanitizedInternalMessage = "internal error"
	sanitizedNotFoundMessage = "resource not found"
)

func SanitizeError(requestCtx context.Context, err error) error {
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	if ok, statusErr := grpcerrors.StatusFromContextError(requestCtx, err); ok {
		return statusErr
	}

	if code, msg, ok := grpcerrors.HandlerStatus(err); ok {
		return status.Error(code, msg)
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
