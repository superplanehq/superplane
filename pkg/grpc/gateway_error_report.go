package grpc

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/requesterror"
	"google.golang.org/grpc/status"
)

// reportGatewayError logs the cause of a failed gateway request and keeps it
// available to the HTTP logging middleware, which sends it to the error
// tracker. The client receives the sanitized error only, so this is the last
// point at which the cause can be kept.
//
// Client errors (4xx) are caused by the request and are not reported.
func reportGatewayError(r *http.Request, original error, sanitized error) {
	if !isServerError(sanitized) {
		return
	}

	log.WithFields(log.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	}).Errorf("gateway request failed: %v", original)

	requesterror.Record(r.Context(), original)
}

// isServerError reports whether the sanitized error becomes a 5xx response.
func isServerError(sanitized error) bool {
	if sanitized == nil {
		return false
	}

	return runtime.HTTPStatusFromCode(status.Code(sanitized)) >= http.StatusInternalServerError
}
