package public

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
)

func (s *Server) handlePolarWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxEventSize))
	if err != nil {
		http.Error(w, "unable to read webhook body", http.StatusBadRequest)
		return
	}

	event, err := polar.VerifyAndParseOrderEvent(r.Header, body, polar.WebhookSecret())
	if errors.Is(err, polar.ErrWebhookSecretMissing) || errors.Is(err, polar.ErrInvalidWebhookSignature) {
		http.Error(w, "invalid webhook signature", http.StatusForbidden)
		return
	}
	if errors.Is(err, polar.ErrUnsupportedWebhookEvent) {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if errors.Is(err, polar.ErrUnusableWebhookPayload) {
		log.WithError(err).Error("ignored polar webhook payload that cannot be applied")
		acceptPolarWebhook(w)
		return
	}
	if err != nil {
		log.WithError(err).Warn("rejected polar webhook payload")
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	var lookup polar.CreditPackLookup
	if polar.Configured() {
		lookup = polar.NewClientFromEnv()
	}
	if err := polar.ApplyOrderEvent(r.Context(), database.DB(r.Context()), event, lookup); err != nil {
		if polar.IsPermanentApplyError(err) {
			polar.LogPermanentApply(event, err)
			acceptPolarWebhook(w)
			return
		}
		log.WithError(err).WithField("polar_order_id", event.Data.ID).Error("failed to apply polar order")
		http.Error(w, "unable to apply order", http.StatusInternalServerError)
		return
	}

	acceptPolarWebhook(w)
}

func acceptPolarWebhook(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}
