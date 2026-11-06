package workers

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/claude"
	"github.com/superplanehq/superplane/pkg/llmcatalog"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestLLMModelCatalogRefreshContinuesAfterProviderAndKeyFailures(t *testing.T) {
	r := support.Setup(t)
	r.Registry.Integrations["claude"] = &claude.Claude{}
	db := database.DB(t.Context())
	now := time.Now()
	for index, state := range []string{models.IntegrationStateReady, models.IntegrationStateReady, models.IntegrationStatePending} {
		key := []string{"expired-key", "valid-key", "pending-key"}[index]
		integration := models.NewIntegration(uuid.New(), r.Organization.ID, "claude", key, map[string]any{
			"apiKey": base64.StdEncoding.EncodeToString([]byte(key)),
		})
		integration.State = state
		createdAt := now.Add(time.Duration(index) * time.Second)
		integration.CreatedAt = &createdAt
		require.NoError(t, db.Create(integration).Error)
	}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader(`{"error":"unavailable"}`))},
		{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"expired"}}`))},
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"claude-new-model","capabilities":{"effort":{"supported":true,"high":{"supported":true}}}}]}`))},
	}}
	worker := NewLLMModelCatalogWorker(llmcatalog.New(r.Encryptor, httpCtx), r.Registry)
	worker.refresh(t.Context())

	catalog, err := llmcatalog.LoadThinkingCatalog(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"high"}, catalog.ThinkingLevels(models.UsageProviderAnthropic, "claude-new-model", models.UsageFundingSourceBYOK))
	require.Len(t, httpCtx.Requests, 3)
	assert.Equal(t, "expired-key", httpCtx.Requests[1].Header.Get("x-api-key"))
	assert.Equal(t, "valid-key", httpCtx.Requests[2].Header.Get("x-api-key"))
}
