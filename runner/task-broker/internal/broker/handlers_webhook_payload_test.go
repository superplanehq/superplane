package broker

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/webhook"
)

func TestDeliverWebhookIncludesClaimedAndFinishedAt(t *testing.T) {
	received := make(chan []byte, 1)
	whSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read", http.StatusBadRequest)
			return
		}
		received <- b
		w.WriteHeader(http.StatusOK)
	}))
	defer whSrv.Close()

	srv := &Server{Webhook: &webhook.Sender{Client: whSrv.Client(), Retries: 1}}

	claimed := time.Date(2026, 5, 24, 20, 1, 2, 0, time.UTC)
	finished := claimed.Add(75 * time.Second)
	exitCode := 0
	srv.DeliverWebhook(&models.Task{
		ID:         "task-1",
		FleetID:    "fleet-1",
		WebhookURL: whSrv.URL,
		Status:     models.StatusSucceeded,
		ExitCode:   &exitCode,
		ClaimedAt:  &claimed,
		FinishedAt: &finished,
	})

	var body []byte
	select {
	case body = <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for webhook delivery")
	}

	var payload api.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("webhook payload json: %v", err)
	}
	if payload.ClaimedAt == nil || !payload.ClaimedAt.Equal(claimed) {
		t.Fatalf("claimed_at: %#v", payload.ClaimedAt)
	}
	if payload.FinishedAt == nil || !payload.FinishedAt.Equal(finished) {
		t.Fatalf("finished_at: %#v", payload.FinishedAt)
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"claimed_at", "finished_at"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("missing json field %q in %s", key, string(body))
		}
	}
}
