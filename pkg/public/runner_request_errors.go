package public

import (
	"context"
	"errors"
	"net/http"
)

// statusClientClosedRequest mirrors the codes.Canceled mapping described in
// pkg/grpc/errors/grpcerrors.go.
const statusClientClosedRequest = 499

func isRunnerRequestCanceled(r *http.Request, err error) bool {
	if r == nil {
		return false
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return errors.Is(r.Context().Err(), context.Canceled)
}
