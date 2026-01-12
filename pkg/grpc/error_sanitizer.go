package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	sanitizedInternalMessage = "internal error"
	sanitizedNotFoundMessage = "resource not found"
)

func sanitizeErrorUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			if s, ok := status.FromError(err); ok {
				if s.Code() == codes.Internal {
					log.WithError(err).Errorf("grpc internal error: %s", info.FullMethod)
				}
			} else {
				log.WithError(err).Errorf("grpc internal error: %s", info.FullMethod)
			}
			return nil, sanitizeError(err)
		}
		return resp, nil
	}
}

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
