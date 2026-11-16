package workers

import (
	"context"
	"encoding/base64"
	"fmt"
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
	"gorm.io/datatypes"
)

func TestLLMModelCatalogRefreshContinuesAfterProviderAndKeyFailures(t *testing.T) {
	r := support.Setup(t)
	r.Registry.Integrations["claude"] = &claude.Claude{}
	db := database.DB(t.Context())
	now := time.Now()
	for index, state := range []string{models.IntegrationStateReady, models.IntegrationStateReady, models.IntegrationStateReady, models.IntegrationStatePending} {
		key := []string{"unreadable-key", "expired-key", "valid-key", "pending-key"}[index]
		integration := models.NewIntegration(uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", index+1)), r.Organization.ID, "claude", key, map[string]any{
			"apiKey": base64.StdEncoding.EncodeToString([]byte(key)),
		})
		if index == 0 {
			integration.Configuration = datatypes.NewJSONType(map[string]any{"apiKey": "invalid-base64"})
		}
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

	rows, err := models.ListLLMModelCapabilities(db, models.UsageProviderAnthropic)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "claude-new-model", rows[0].Model)
	assert.Equal(t, []string{"high"}, []string(rows[0].ThinkingLevels))
	require.Len(t, httpCtx.Requests, 3)
	assert.Equal(t, "expired-key", httpCtx.Requests[1].Header.Get("x-api-key"))
	assert.Equal(t, "valid-key", httpCtx.Requests[2].Header.Get("x-api-key"))
}

// Cancel the scan after a bounded amount of provider work, without sleeping.
// The production five-minute deadline takes the same cancellation path.
type budgetCatalogHTTP struct {
	cancel    context.CancelFunc
	remaining int
	keys      []string
}

func (h *budgetCatalogHTTP) Do(req *http.Request) (*http.Response, error) {
	h.keys = append(h.keys, req.Header.Get("x-api-key"))
	h.remaining--
	if h.remaining == 0 {
		h.cancel()
		return nil, context.DeadlineExceeded
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"model"}]}`))}, nil
}

func TestLLMModelCatalogContinuesAcrossExhaustedScanWindows(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	const count = 205
	for i := range count {
		key := fmt.Sprintf("key-%03d", i)
		integration := models.NewIntegration(uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)), r.Organization.ID, "claude", key, map[string]any{"apiKey": base64.StdEncoding.EncodeToString([]byte(key))})
		integration.State = models.IntegrationStateReady
		require.NoError(t, db.Create(integration).Error)
	}
	httpCtx := &budgetCatalogHTTP{}
	worker := NewLLMModelCatalogWorker(llmcatalog.New(r.Encryptor, httpCtx), r.Registry)
	for range 9 {
		ctx, cancel := context.WithCancel(t.Context())
		httpCtx.cancel = cancel
		httpCtx.remaining = 25
		err := worker.refreshClaudeIntegrations(ctx, db.WithContext(ctx))
		cancel()
		if worker.after == uuid.Nil {
			require.NoError(t, err)
			break
		}
		require.Error(t, err)
	}
	require.Len(t, httpCtx.keys, count)
	for i, key := range httpCtx.keys {
		assert.Equal(t, fmt.Sprintf("key-%03d", i), key)
	}
	assert.Equal(t, uuid.Nil, worker.after)
}
