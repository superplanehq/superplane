package public

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/datadog"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

const (
	// rejectedDatadogReceiptLimit is the most rejected calls stored for one
	// integration during rejectedDatadogReceiptWindow. Later rejected calls
	// still return 403. They do not add a row.
	rejectedDatadogReceiptLimit  = 10
	rejectedDatadogReceiptWindow = time.Hour
)

var errRequestBodyTooLarge = errors.New("request body is too large")

// rejectedReceiptLimitCache remembers integrations that are already at the
// rejected-receipt limit. Later calls skip the database until a stored
// receipt leaves the window.
type rejectedReceiptLimitCache struct {
	mu        sync.Mutex
	fullUntil map[uuid.UUID]time.Time
}

func (c *rejectedReceiptLimitCache) blocked(id uuid.UUID, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	until, ok := c.fullUntil[id]
	if !ok {
		return false
	}
	if !now.Before(until) {
		delete(c.fullUntil, id)
		return false
	}
	return true
}

func (c *rejectedReceiptLimitCache) markFull(id uuid.UUID, until time.Time) {
	if until.IsZero() {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fullUntil == nil {
		c.fullUntil = map[uuid.UUID]time.Time{}
	}
	if current, ok := c.fullUntil[id]; ok && !current.Before(until) {
		return
	}
	c.fullUntil[id] = until
}

var rejectedDatadogReceiptLimits rejectedReceiptLimitCache

type statusCapture struct {
	http.ResponseWriter
	status int
}

func (c *statusCapture) WriteHeader(status int) {
	if c.status == 0 {
		c.status = status
	}
	c.ResponseWriter.WriteHeader(status)
}

func (c *statusCapture) Write(body []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	return c.ResponseWriter.Write(body)
}

// trackDatadogWebhook stores one receipt for a Datadog event call and returns
// the request that carries the receipt ID. deliver is false when this method
// already wrote the response. Other integration calls are unchanged.
func (s *Server) trackDatadogWebhook(r *http.Request, w http.ResponseWriter, integration *models.Integration) (*http.Request, http.ResponseWriter, func(), bool) {
	noop := func() {}
	if !datadogEventRequest(r, integration) {
		return r, w, noop, true
	}

	if !s.datadogWebhookAuthenticated(r, integration) {
		request, response, finish := s.trackRejectedDatadogWebhook(r, w, integration)
		return request, response, finish, true
	}

	body, err := readAndRestoreBody(r, MaxEventSize)
	if errors.Is(err, errRequestBodyTooLarge) {
		http.Error(
			w,
			fmt.Sprintf("Request body is too large - must be up to %d bytes", MaxEventSize),
			http.StatusRequestEntityTooLarge,
		)
		return r, w, noop, false
	}
	if err != nil {
		logging.ForIntegration(*integration).WithError(err).Error("failed to read Datadog webhook body for receipt")
		return r, w, noop, true
	}

	summary := datadog.SummarizeWebhook(body)
	receiptID, err := models.CreateDatadogWebhookReceipt(database.DB(r.Context()), models.DatadogWebhookReceipt{
		IntegrationID:   integration.ID,
		OrganizationID:  integration.OrganizationID,
		EventType:       summary.EventType,
		AlertTransition: summary.AlertTransition,
		AlertID:         summary.AlertID,
		Service:         summary.Service,
		IssueID:         summary.IssueID,
		Outcome:         models.DatadogWebhookOutcomePending,
	})
	if err != nil {
		logging.ForIntegration(*integration).WithError(err).Error("failed to store Datadog webhook receipt")
		return r, w, noop, true
	}

	request, response, finish := s.attachDatadogReceipt(r, w, integration, receiptID, "")
	return request, response, finish, true
}

func (s *Server) trackRejectedDatadogWebhook(r *http.Request, w http.ResponseWriter, integration *models.Integration) (*http.Request, http.ResponseWriter, func()) {
	noop := func() {}
	now := time.Now().UTC()
	if rejectedDatadogReceiptLimits.blocked(integration.ID, now) {
		return r, w, noop
	}

	since := now.Add(-rejectedDatadogReceiptWindow)
	receiptID, stored, oldest, err := models.CreateRejectedDatadogWebhookReceiptIfAllowed(
		database.DB(r.Context()),
		models.DatadogWebhookReceipt{
			IntegrationID:  integration.ID,
			OrganizationID: integration.OrganizationID,
			HTTPStatus:     http.StatusForbidden,
			Outcome:        models.DatadogWebhookOutcomeRejected,
		},
		since,
		rejectedDatadogReceiptLimit,
	)
	if err != nil {
		logging.ForIntegration(*integration).WithError(err).Error("failed to store rejected Datadog webhook receipt")
		return r, w, noop
	}
	if !stored {
		if !oldest.IsZero() {
			rejectedDatadogReceiptLimits.markFull(integration.ID, oldest.Add(rejectedDatadogReceiptWindow))
		}
		return r, w, noop
	}

	// The receipt slot is reserved. Read the body only now, and only up to the
	// webhook size limit, so a rejected call cannot fill memory.
	body, err := readAndRestoreBody(r, MaxEventSize)
	if err != nil && !errors.Is(err, errRequestBodyTooLarge) {
		logging.ForIntegration(*integration).WithError(err).Error("failed to read rejected Datadog webhook body for receipt")
	}
	if err == nil {
		summary := datadog.SummarizeWebhook(body)
		updateErr := models.UpdateDatadogWebhookReceiptSummary(database.DB(r.Context()), receiptID, models.DatadogWebhookReceipt{
			EventType:       summary.EventType,
			AlertTransition: summary.AlertTransition,
			AlertID:         summary.AlertID,
			Service:         summary.Service,
			IssueID:         summary.IssueID,
		})
		if updateErr != nil {
			logging.ForIntegration(*integration).WithError(updateErr).Error("failed to update rejected Datadog webhook receipt")
		}
	}

	return s.attachDatadogReceipt(r, w, integration, receiptID, models.DatadogWebhookOutcomeRejected)
}

func (s *Server) attachDatadogReceipt(r *http.Request, w http.ResponseWriter, integration *models.Integration, receiptID uuid.UUID, outcome string) (*http.Request, http.ResponseWriter, func()) {
	state := &datadog.WebhookReceiptState{ID: receiptID, Outcome: outcome}
	request := datadog.WithWebhookReceipt(r, state)
	capture := &statusCapture{ResponseWriter: w}
	finish := func() {
		status := capture.status
		if status == 0 {
			status = http.StatusOK
		}
		result := state.Outcome
		if result == "" {
			result = datadogOutcomeFromStatus(status)
		}
		err := models.UpdateDatadogWebhookReceiptResult(
			database.DB(request.Context()),
			state.ID,
			status,
			result,
			state.SubscriptionCount,
		)
		if err != nil {
			logging.ForIntegration(*integration).WithError(err).Error("failed to update Datadog webhook receipt")
		}
	}
	return request, capture, finish
}

func (s *Server) datadogWebhookAuthenticated(r *http.Request, integration *models.Integration) bool {
	if s == nil || s.encryptor == nil || r == nil || integration == nil {
		return false
	}
	appCtx := contexts.NewIntegrationContext(
		database.DB(r.Context()),
		nil,
		integration,
		s.encryptor,
		s.registry,
		nil,
	)
	return datadog.WebhookRequestAuthenticated(appCtx, r)
}

func datadogEventRequest(r *http.Request, integration *models.Integration) bool {
	if r == nil || integration == nil || integration.AppName != "datadog" {
		return false
	}
	return r.Method == http.MethodPost && r.URL != nil && strings.HasSuffix(r.URL.Path, "/events")
}

func datadogOutcomeFromStatus(status int) string {
	switch status {
	case http.StatusForbidden:
		return models.DatadogWebhookOutcomeRejected
	case http.StatusOK:
		return models.DatadogWebhookOutcomeAccepted
	default:
		return models.DatadogWebhookOutcomeFailed
	}
}

func readAndRestoreBody(r *http.Request, limit int64) ([]byte, error) {
	if r == nil || r.Body == nil {
		return nil, nil
	}

	original := r.Body
	body, err := io.ReadAll(io.LimitReader(original, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errRequestBodyTooLarge
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	_ = original.Close()
	return body, nil
}
