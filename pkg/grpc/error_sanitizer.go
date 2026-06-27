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

	// pgQueryCanceledCode is PostgreSQL's SQLSTATE for query_canceled (57014).
	// pgx returns this when an in-flight query is interrupted — typically
	// because the request context was canceled (client disconnect) or the
	// server-side statement_timeout fired.
	pgQueryCanceledCode = "57014"
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

	if ok, statusErr := statusFromCanceledQuery(requestCtx, err); ok {
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

// statusFromCanceledQuery maps a PostgreSQL query_canceled error (SQLSTATE
// 57014) to the appropriate gRPC status when the cancellation was driven by
// the request context. This commonly happens when the client disconnects
// mid-query: pgx forwards ctx cancellation to the server, which responds with
// query_canceled. Without this mapping we surface a confusing HTTP 500 (and a
// Sentry alert) for what is really a client-side abort.
//
// If the request context is not canceled or expired, the error is left to the
// generic pgError path so genuine server-side statement_timeout failures keep
// surfacing as internal errors.
func statusFromCanceledQuery(requestCtx context.Context, err error) (bool, error) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != pgQueryCanceledCode {
		return false, nil
	}

	ctxErr := requestCtx.Err()
	if ctxErr == nil {
		return false, nil
	}

	if errors.Is(ctxErr, context.DeadlineExceeded) {
		return true, status.Error(codes.DeadlineExceeded, grpcerrors.RequestDeadlineExceededMessage)
	}

	if errors.Is(ctxErr, context.Canceled) {
		return true, status.Error(codes.Canceled, grpcerrors.RequestCanceledMessage)
	}

	return false, nil
}

func looksNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found")
}
