package public

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/datadog"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

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
// the request that carries the receipt ID. Other integration calls are unchanged.
func (s *Server) trackDatadogWebhook(r *http.Request, w http.ResponseWriter, integration *models.Integration) (*http.Request, http.ResponseWriter, func()) {
	noop := func() {}
	if !datadogEventRequest(r, integration) {
		return r, w, noop
	}

	body, err := readAndRestoreBody(r)
	if err != nil {
		logging.ForIntegration(*integration).WithError(err).Error("failed to read Datadog webhook body for receipt")
		return r, w, noop
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
		return r, w, noop
	}

	state := &datadog.WebhookReceiptState{ID: receiptID}
	request := datadog.WithWebhookReceipt(r, state)
	capture := &statusCapture{ResponseWriter: w}
	finish := func() {
		status := capture.status
		if status == 0 {
			status = http.StatusOK
		}
		outcome := state.Outcome
		if outcome == "" {
			outcome = datadogOutcomeFromStatus(status)
		}
		err := models.UpdateDatadogWebhookReceiptResult(
			database.DB(request.Context()),
			state.ID,
			status,
			outcome,
			state.SubscriptionCount,
		)
		if err != nil {
			logging.ForIntegration(*integration).WithError(err).Error("failed to update Datadog webhook receipt")
		}
	}
	return request, capture, finish
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

func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r == nil || r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, err
}
