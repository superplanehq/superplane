package public

import (
	"net/http"
	"strconv"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	defaultDatadogWebhookPage  = 1
	defaultDatadogWebhookLimit = 50
	maxDatadogWebhookLimit     = 100
)

type adminDatadogWebhookReceipt struct {
	ID                string   `json:"id"`
	ReceivedAt        string   `json:"received_at"`
	IntegrationID     string   `json:"integration_id"`
	OrganizationID    string   `json:"organization_id"`
	EventType         string   `json:"event_type"`
	AlertTransition   string   `json:"alert_transition"`
	AlertID           string   `json:"alert_id"`
	Service           string   `json:"service"`
	IssueID           string   `json:"issue_id"`
	HTTPStatus        int      `json:"http_status"`
	Outcome           string   `json:"outcome"`
	SubscriptionCount int      `json:"subscription_count"`
	TaskIDs           []string `json:"task_ids"`
}

type adminDatadogWebhooksResponse struct {
	Items []adminDatadogWebhookReceipt `json:"items"`
	Total int                          `json:"total"`
	Page  int                          `json:"page"`
	Limit int                          `json:"limit"`
}

func (s *Server) adminListDatadogWebhooks(w http.ResponseWriter, r *http.Request) {
	page := defaultDatadogWebhookPage
	if value, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && value > 0 {
		page = value
	}
	limit := defaultDatadogWebhookLimit
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 {
		limit = value
	}
	if limit > maxDatadogWebhookLimit {
		limit = maxDatadogWebhookLimit
	}

	receipts, total, err := models.ListDatadogWebhookReceipts(database.DB(r.Context()), limit, (page-1)*limit)
	if err != nil {
		http.Error(w, "Failed to load Datadog webhooks", http.StatusInternalServerError)
		return
	}

	items := make([]adminDatadogWebhookReceipt, 0, len(receipts))
	for _, receipt := range receipts {
		taskIDs := receipt.TaskIDList()
		if taskIDs == nil {
			taskIDs = []string{}
		}
		items = append(items, adminDatadogWebhookReceipt{
			ID:                receipt.ID.String(),
			ReceivedAt:        receipt.ReceivedAt.UTC().Format(time.RFC3339),
			IntegrationID:     receipt.IntegrationID.String(),
			OrganizationID:    receipt.OrganizationID.String(),
			EventType:         receipt.EventType,
			AlertTransition:   receipt.AlertTransition,
			AlertID:           receipt.AlertID,
			Service:           receipt.Service,
			IssueID:           receipt.IssueID,
			HTTPStatus:        receipt.HTTPStatus,
			Outcome:           receipt.Outcome,
			SubscriptionCount: receipt.SubscriptionCount,
			TaskIDs:           taskIDs,
		})
	}

	respondJSON(w, adminDatadogWebhooksResponse{
		Items: items,
		Total: int(total),
		Page:  page,
		Limit: limit,
	})
}
