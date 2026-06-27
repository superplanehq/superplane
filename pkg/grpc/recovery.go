package grpc

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func sentryRecoveryHandler(p any) error {
	stack := debug.Stack()
	log.Errorf("recovered from panic in gRPC handler: %v. Stack: %s", p, stack)

	hub := sentry.CurrentHub()
	if hub != nil && hub.Client() != nil {
		/*
		 * Attach the raw panic stack as an extra so that, when Sentry's
		 * default attribution picks this recovery handler as the topmost
		 * in-app frame, we can still identify the original panic site
		 * from the event details.
		 */
		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetTag("recovered_panic", "true")
			scope.SetExtra("panic_value", fmt.Sprintf("%v", p))
			scope.SetExtra("panic_stack", string(stack))
			hub.Recover(p)
		})
		hub.Flush(2 * time.Second)
	}

	return status.Errorf(codes.Internal, "internal server error")
}
