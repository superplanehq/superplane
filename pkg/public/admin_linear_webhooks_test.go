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

func TestAdminLinearWebhooks(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-linear-webhooks@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/linear/webhooks",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("lists stored receipts without a payload", func(t *testing.T) {
		receiptID, err := models.CreateLinearWebhookReceipt(database.Conn(), models.LinearWebhookReceipt{
			ID:                uuid.New(),
			ReceivedAt:        time.Now().UTC().Add(-time.Hour),
			IntegrationID:     uuid.New(),
			OrganizationID:    uuid.New(),
			WebhookID:         uuid.New(),
			EventType:         "Issue",
			Action:            "create",
			IssueIdentifier:   "ENG-142",
			IssueID:           "2174add1-f7c8-44e3-bbf3-2d60b5ea8bc9",
			TeamKey:           "ENG",
			WorkspaceKey:      "acme",
			HTTPStatus:        http.StatusOK,
			Outcome:           models.LinearWebhookOutcomeAccepted,
			SubscriptionCount: 1,
		})
		require.NoError(t, err)
		taskID := uuid.New()
		require.NoError(t, models.AppendLinearWebhookTask(database.Conn(), receiptID, taskID))
		require.NoError(t, models.AppendLinearWebhookTask(database.Conn(), receiptID, taskID))

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/linear/webhooks",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)
		assert.NotContains(t, response.Body.String(), "secret-description")
		assert.NotContains(t, response.Body.String(), "ada@example.com")

		var body adminLinearWebhooksResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		var receipt *adminLinearWebhookReceipt
		for i := range body.Items {
			if body.Items[i].ID == receiptID.String() {
				receipt = &body.Items[i]
			}
		}
		require.NotNil(t, receipt)
		assert.Equal(t, "Issue", receipt.EventType)
		assert.Equal(t, "create", receipt.Action)
		assert.Equal(t, "ENG-142", receipt.IssueIdentifier)
		assert.Equal(t, "ENG", receipt.TeamKey)
		assert.Equal(t, "acme", receipt.WorkspaceKey)
		assert.Equal(t, models.LinearWebhookOutcomeAccepted, receipt.Outcome)
		assert.Equal(t, http.StatusOK, receipt.HTTPStatus)
		assert.Equal(t, []string{taskID.String()}, receipt.TaskIDs)
	})
}
