package grpc

import (
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	customFunc recovery.RecoveryHandlerFunc = sentryRecoveryHandler
)

func sentryRecoveryHandler(p any) error {
	log.Errorf("recovered from panic in gRPC handler: %v. Stack: %s", p, debug.Stack())

	hub := sentry.CurrentHub()
	if hub != nil && hub.Client() != nil {
		hub.Recover(p)
		hub.Flush(2 * time.Second)
	}

	return status.Errorf(codes.Internal, "internal server error")
}
