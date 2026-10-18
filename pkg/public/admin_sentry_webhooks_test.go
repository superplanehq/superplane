package public

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

func findSentryReceipt(items []adminSentryWebhookReceipt, id string) *adminSentryWebhookReceipt {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

func TestAdminSentryWebhooks(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-sentry-webhooks@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/sentry/webhooks",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("lists stored receipts without a payload", func(t *testing.T) {
		receiptID, err := models.CreateSentryWebhookReceipt(database.Conn(), models.SentryWebhookReceipt{
			ID:               uuid.New(),
			ReceivedAt:       time.Now().UTC().Add(-time.Hour),
			HookResource:     "issue",
			Action:           "created",
			InstallationUUID: "install-1",
			ProjectSlug:      "javascript-react-f",
			IssueID:          "99",
			IssueShortID:     "JS-9",
			HTTPStatus:       http.StatusOK,
			Outcome:          models.SentryWebhookOutcomeAccepted,
			IntegrationCount: 1,
		})
		require.NoError(t, err)
		taskID := uuid.New()
		require.NoError(t, models.AppendSentryWebhookTask(database.Conn(), receiptID, taskID))
		require.NoError(t, models.AppendSentryWebhookTask(database.Conn(), receiptID, taskID))

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/sentry/webhooks",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)
		assert.NotContains(t, response.Body.String(), "accessToken")
		assert.NotContains(t, response.Body.String(), "payload")

		var body adminSentryWebhooksResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		receipt := findSentryReceipt(body.Items, receiptID.String())
		require.NotNil(t, receipt)
		assert.Equal(t, "issue", receipt.HookResource)
		assert.Equal(t, "created", receipt.Action)
		assert.Equal(t, "javascript-react-f", receipt.ProjectSlug)
		assert.Equal(t, models.SentryWebhookOutcomeAccepted, receipt.Outcome)
		assert.Equal(t, http.StatusOK, receipt.HTTPStatus)
		assert.Equal(t, []string{taskID.String()}, receipt.TaskIDs)
	})

	t.Run("filters receipts by project slug", func(t *testing.T) {
		slugToken := strings.ReplaceAll(uuid.New().String(), "-", "")
		productionSlug := "Production-" + slugToken
		otherSlug := "billing-" + slugToken
		literalSlug := "100%_" + slugToken
		wildcardSlug := "100X" + slugToken

		productionID := createAdminSentryReceipt(t, productionSlug)
		otherID := createAdminSentryReceipt(t, otherSlug)
		literalID := createAdminSentryReceipt(t, literalSlug)
		wildcardID := createAdminSentryReceipt(t, wildcardSlug)

		unfiltered := listAdminSentryWebhooks(t, server, token, "")
		assert.Equal(t, countSentryWebhookReceipts(t), unfiltered.Total)
		require.NotNil(t, findSentryReceipt(unfiltered.Items, productionID))
		require.NotNil(t, findSentryReceipt(unfiltered.Items, otherID))
		require.NotNil(t, findSentryReceipt(unfiltered.Items, literalID))
		require.NotNil(t, findSentryReceipt(unfiltered.Items, wildcardID))

		production := listAdminSentryWebhooks(t, server, token, "production-"+slugToken)
		assert.Equal(t, 1, production.Total)
		require.Len(t, production.Items, 1)
		assert.Equal(t, productionID, production.Items[0].ID)
		assert.Equal(t, productionSlug, production.Items[0].ProjectSlug)

		trimmed := listAdminSentryWebhooks(t, server, token, "  production-"+slugToken+"  ")
		assert.Equal(t, 1, trimmed.Total)
		require.Len(t, trimmed.Items, 1)
		assert.Equal(t, productionID, trimmed.Items[0].ID)

		missing := listAdminSentryWebhooks(t, server, token, "missing-"+slugToken)
		assert.Equal(t, 0, missing.Total)
		assert.Empty(t, missing.Items)

		literal := listAdminSentryWebhooks(t, server, token, literalSlug)
		assert.Equal(t, 1, literal.Total)
		require.Len(t, literal.Items, 1)
		assert.Equal(t, literalID, literal.Items[0].ID)
		assert.Nil(t, findSentryReceipt(literal.Items, wildcardID))
	})

	t.Run("keeps a multibyte project filter inside the field limit", func(t *testing.T) {
		project := strings.Repeat("a", 199) + "é"
		filtered := listAdminSentryWebhooks(t, server, token, project)
		assert.Equal(t, 0, filtered.Total)
		assert.Empty(t, filtered.Items)
	})
}

func createAdminSentryReceipt(t *testing.T, projectSlug string) string {
	t.Helper()
	receiptID, err := models.CreateSentryWebhookReceipt(database.Conn(), models.SentryWebhookReceipt{
		ID:           uuid.New(),
		ReceivedAt:   time.Now().UTC(),
		HookResource: "issue",
		Action:       "created",
		ProjectSlug:  projectSlug,
		HTTPStatus:   http.StatusOK,
		Outcome:      models.SentryWebhookOutcomeAccepted,
	})
	require.NoError(t, err)
	return receiptID.String()
}

func listAdminSentryWebhooks(t *testing.T, server *Server, token, project string) adminSentryWebhooksResponse {
	t.Helper()
	path := "/admin/api/sentry/webhooks"
	if project != "" {
		path += "?project=" + url.QueryEscape(project)
	}
	response := execRequest(server, requestParams{
		method:     "GET",
		path:       path,
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, response.Code)

	var body adminSentryWebhooksResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	return body
}

func countSentryWebhookReceipts(t *testing.T) int {
	t.Helper()
	var total int64
	require.NoError(t, database.Conn().Model(&models.SentryWebhookReceipt{}).Count(&total).Error)
	return int(total)
}
