package common

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__Client__ListAllOpenDependabotAlerts(t *testing.T) {
	var requests []*http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r)
		assert.Equal(t, "/repos/acme/payments/dependabot/alerts", r.URL.Path)
		assert.Equal(t, "open", r.URL.Query().Get("state"))
		assert.Equal(t, "100", r.URL.Query().Get("per_page"))

		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			_, _ = w.Write([]byte(`[{"number": 1, "state": "open"}]`))
			return
		}

		w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/acme/payments/dependabot/alerts?page=2>; rel="next"`, r.Host))
		_, _ = w.Write([]byte(`[{"number": 2, "state": "open"}]`))
	}))
	t.Cleanup(srv.Close)

	client := clientForTestServer(t, srv)
	alerts, err := client.ListAllOpenDependabotAlerts(context.Background(), "acme/payments")

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.Len(t, alerts, 2)
	assert.Equal(t, 2, alerts[0].GetNumber())
	assert.Equal(t, 1, alerts[1].GetNumber())
}
