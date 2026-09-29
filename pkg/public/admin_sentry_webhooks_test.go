package public

import (
	"encoding/json"
	"net/http"
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
			ReceivedAt:       time.Date(2026, 9, 29, 11, 14, 0, 0, time.UTC),
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
}
