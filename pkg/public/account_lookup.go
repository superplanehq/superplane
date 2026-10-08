package public

import (
	"context"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

type accountLookupFailure struct {
	err            error
	detail         string
	missingAccount bool
}

func writeAccountLookupError(w http.ResponseWriter, r *http.Request, failure accountLookupFailure) {
	if accountLookupDisconnected(r, failure.err) {
		log.WithError(failure.err).Info("account request canceled")
		w.WriteHeader(statusClientClosedRequest)
		return
	}

	if failure.missingAccount {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	log.Errorf("%s: %v", failure.detail, failure.err)
	middleware.SetServerError(r.Context(), failure.err, nil)
	w.WriteHeader(accountLookupServerStatus(failure.err))
}

func accountLookupDisconnected(r *http.Request, err error) bool {
	if r == nil || err == nil {
		return false
	}
	return errors.Is(r.Context().Err(), context.Canceled)
}

func accountLookupServerStatus(err error) int {
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout
	}
	return http.StatusInternalServerError
}
