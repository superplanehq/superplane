package public

import (
	"net/http"
	"strconv"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	defaultSentryWebhookPage  = 1
	defaultSentryWebhookLimit = 50
	maxSentryWebhookLimit     = 100
)

type adminSentryWebhookReceipt struct {
	ID               string   `json:"id"`
	ReceivedAt       string   `json:"received_at"`
	HookResource     string   `json:"hook_resource"`
	Action           string   `json:"action"`
	InstallationUUID string   `json:"installation_uuid"`
	OrganizationSlug string   `json:"organization_slug"`
	ProjectSlug      string   `json:"project_slug"`
	IssueID          string   `json:"issue_id"`
	IssueShortID     string   `json:"issue_short_id"`
	HTTPStatus       int      `json:"http_status"`
	Outcome          string   `json:"outcome"`
	IntegrationCount int      `json:"integration_count"`
	TaskIDs          []string `json:"task_ids"`
}

type adminSentryWebhooksResponse struct {
	Items []adminSentryWebhookReceipt `json:"items"`
	Total int                         `json:"total"`
	Page  int                         `json:"page"`
	Limit int                         `json:"limit"`
}

func (s *Server) adminListSentryWebhooks(w http.ResponseWriter, r *http.Request) {
	page := defaultSentryWebhookPage
	if value, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && value > 0 {
		page = value
	}
	limit := defaultSentryWebhookLimit
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 {
		limit = value
	}
	if limit > maxSentryWebhookLimit {
		limit = maxSentryWebhookLimit
	}

	receipts, total, err := models.ListSentryWebhookReceipts(
		database.DB(r.Context()),
		limit,
		(page-1)*limit,
		r.URL.Query().Get("project"),
	)
	if err != nil {
		http.Error(w, "Failed to load Sentry webhooks", http.StatusInternalServerError)
		return
	}

	items := make([]adminSentryWebhookReceipt, 0, len(receipts))
	for _, receipt := range receipts {
		taskIDs := receipt.TaskIDList()
		if taskIDs == nil {
			taskIDs = []string{}
		}
		items = append(items, adminSentryWebhookReceipt{
			ID:               receipt.ID.String(),
			ReceivedAt:       receipt.ReceivedAt.UTC().Format(time.RFC3339),
			HookResource:     receipt.HookResource,
			Action:           receipt.Action,
			InstallationUUID: receipt.InstallationUUID,
			OrganizationSlug: receipt.OrganizationSlug,
			ProjectSlug:      receipt.ProjectSlug,
			IssueID:          receipt.IssueID,
			IssueShortID:     receipt.IssueShortID,
			HTTPStatus:       receipt.HTTPStatus,
			Outcome:          receipt.Outcome,
			IntegrationCount: receipt.IntegrationCount,
			TaskIDs:          taskIDs,
		})
	}

	respondJSON(w, adminSentryWebhooksResponse{
		Items: items,
		Total: int(total),
		Page:  page,
		Limit: limit,
	})
}
