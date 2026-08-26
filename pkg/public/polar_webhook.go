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

	event, err := polar.VerifyAndParseOrderPaid(r.Header, body, polar.WebhookSecret())
	if errors.Is(err, polar.ErrWebhookSecretMissing) || errors.Is(err, polar.ErrInvalidWebhookSignature) {
		http.Error(w, "invalid webhook signature", http.StatusForbidden)
		return
	}
	if errors.Is(err, polar.ErrUnsupportedWebhookEvent) {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if err != nil {
		log.WithError(err).Warn("rejected polar webhook payload")
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	if err := polar.ApplyOrderPaid(database.DB(r.Context()), event); err != nil {
		log.WithError(err).WithField("polar_order_id", event.Data.ID).Error("failed to apply polar order")
		http.Error(w, "unable to apply order", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}
