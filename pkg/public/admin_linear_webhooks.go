package public

import (
	"net/http"
	"strconv"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	defaultLinearWebhookPage  = 1
	defaultLinearWebhookLimit = 50
	maxLinearWebhookLimit     = 100
)

type adminLinearWebhookReceipt struct {
	ID                string   `json:"id"`
	ReceivedAt        string   `json:"received_at"`
	IntegrationID     string   `json:"integration_id"`
	OrganizationID    string   `json:"organization_id"`
	WebhookID         string   `json:"webhook_id"`
	EventType         string   `json:"event_type"`
	Action            string   `json:"action"`
	IssueIdentifier   string   `json:"issue_identifier"`
	IssueID           string   `json:"issue_id"`
	TeamKey           string   `json:"team_key"`
	WorkspaceKey      string   `json:"workspace_key"`
	HTTPStatus        int      `json:"http_status"`
	Outcome           string   `json:"outcome"`
	SubscriptionCount int      `json:"subscription_count"`
	TaskIDs           []string `json:"task_ids"`
}

type adminLinearWebhooksResponse struct {
	Items []adminLinearWebhookReceipt `json:"items"`
	Total int                         `json:"total"`
	Page  int                         `json:"page"`
	Limit int                         `json:"limit"`
}

func (s *Server) adminListLinearWebhooks(w http.ResponseWriter, r *http.Request) {
	page := defaultLinearWebhookPage
	if value, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && value > 0 {
		page = value
	}
	limit := defaultLinearWebhookLimit
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 {
		limit = value
	}
	if limit > maxLinearWebhookLimit {
		limit = maxLinearWebhookLimit
	}

	receipts, total, err := models.ListLinearWebhookReceipts(database.DB(r.Context()), limit, (page-1)*limit)
	if err != nil {
		http.Error(w, "Failed to load Linear webhooks", http.StatusInternalServerError)
		return
	}

	items := make([]adminLinearWebhookReceipt, 0, len(receipts))
	for _, receipt := range receipts {
		taskIDs := receipt.TaskIDList()
		if taskIDs == nil {
			taskIDs = []string{}
		}
		items = append(items, adminLinearWebhookReceipt{
			ID:                receipt.ID.String(),
			ReceivedAt:        receipt.ReceivedAt.UTC().Format(time.RFC3339),
			IntegrationID:     receipt.IntegrationID.String(),
			OrganizationID:    receipt.OrganizationID.String(),
			WebhookID:         receipt.WebhookID.String(),
			EventType:         receipt.EventType,
			Action:            receipt.Action,
			IssueIdentifier:   receipt.IssueIdentifier,
			IssueID:           receipt.IssueID,
			TeamKey:           receipt.TeamKey,
			WorkspaceKey:      receipt.WorkspaceKey,
			HTTPStatus:        receipt.HTTPStatus,
			Outcome:           receipt.Outcome,
			SubscriptionCount: receipt.SubscriptionCount,
			TaskIDs:           taskIDs,
		})
	}

	respondJSON(w, adminLinearWebhooksResponse{
		Items: items,
		Total: int(total),
		Page:  page,
		Limit: limit,
	})
}
