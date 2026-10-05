package public

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestAdminOrganizationSpendingReport(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	foreign, err := models.CreateOrganization(support.RandomName("foreign-org"), "")
	require.NoError(t, err)

	var members int64
	require.NoError(t, database.Conn().Model(&models.User{}).
		Where("organization_id = ? AND account_id = ?", foreign.ID, r.Account.ID).
		Count(&members).Error)
	require.Zero(t, members)

	now := time.Now()
	require.NoError(t, database.Conn().Create(&models.WorkspaceUsageEvent{
		ID:                 uuid.New(),
		OrganizationID:     foreign.ID,
		CanvasRunID:        uuid.New(),
		NodeExecutionID:    uuid.New(),
		NodeID:             "prompt",
		Provider:           models.UsageProviderOpenAI,
		Model:              "gpt-4o",
		UsageKind:          models.UsageKindModel,
		FundingSource:      models.UsageFundingSourceHosted,
		TotalTokens:        1_000_000,
		CostMicros:         2_000_000,
		ProviderCostMicros: 2_000_000,
		Currency:           "usd",
		PriceBookVersion:   "test",
		IdempotencyKey:     "admin-spending-report:" + foreign.ID.String(),
		OccurredAt:         now,
		CreatedAt:          now,
	}).Error)

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", support.RandomName("regular-spending")+"@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       spendingReportPath(foreign.ID.String(), now.Add(-24*time.Hour), now.Add(time.Minute), models.UsageKindModel),
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("admin who is not a member receives the report", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       spendingReportPath(foreign.ID.String(), now.Add(-24*time.Hour), now.Add(time.Minute), models.UsageKindModel),
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body map[string]any
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		kpi, ok := body["kpiTotals"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "1000000", kpi["totalTokens"])
		assert.Equal(t, "200", kpi["hostedCostCents"])

		explorer, ok := body["explorerTotals"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "1000000", explorer["totalTokens"])
		assert.Contains(t, body, "credit")
		assert.Contains(t, body, "catalogs")
	})

	t.Run("invalid time window returns 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       spendingReportPath(foreign.ID.String(), now, now.Add(-time.Hour), models.UsageKindModel),
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "end time must be after start time")
	})
}

func spendingReportPath(orgID string, start, end time.Time, usageKind string) string {
	query := url.Values{}
	query.Set("startTime", start.UTC().Format(time.RFC3339Nano))
	query.Set("endTime", end.UTC().Format(time.RFC3339Nano))
	query.Set("usageKind", usageKind)
	query.Set("groupBy", models.SpendingGroupByModel)
	query.Set("timeGrain", models.SpendingTimeGrainDay)
	return "/admin/api/organizations/" + orgID + "/spending-report?" + query.Encode()
}
