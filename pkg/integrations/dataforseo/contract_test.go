// pkg/integrations/dataforseo/contract_test.go
package dataforseo

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// realHTTPContext adapts the real net/http client to core.HTTPContext, so
// this test hits the real DataForSEO API instead of a mocked response.
type realHTTPContext struct{ client *http.Client }

func (r realHTTPContext) Do(req *http.Request) (*http.Response, error) {
	return r.client.Do(req)
}

func TestContract_DataForSEO(t *testing.T) {
	apiKey := os.Getenv("DATAFORSEO_TEST_API_KEY")
	if apiKey == "" {
		t.Skip("DATAFORSEO_TEST_API_KEY not set, skipping contract test")
	}

	httpCtx := realHTTPContext{client: &http.Client{Timeout: 30 * time.Second}}
	client, err := NewClient(httpCtx, &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": apiKey},
	})
	require.NoError(t, err)

	t.Run("Verify", func(t *testing.T) {
		err := client.Verify()
		require.NoError(t, err)
	})

	t.Run("PostAudit and GetSummary shape", func(t *testing.T) {
		taskID, err := client.PostAudit("freehire.me", 5)
		require.NoError(t, err)
		require.NotEmpty(t, taskID)

		// Real crawls take time; this only asserts the response shape parses,
		// not that the crawl has finished within the test's lifetime.
		_, err = client.GetSummary(taskID)
		require.NoError(t, err)
	})
}
