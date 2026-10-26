package polar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHasPaymentMethod(t *testing.T) {
	now := time.Now()
	for _, test := range []struct {
		name      string
		status    int
		items     []map[string]any
		want      bool
		wantError bool
	}{
		{name: "customer without a card", status: 200},
		{name: "missing customer", status: 404},
		{name: "confirmed saved card", status: 200, items: []map[string]any{{"id": "pm_card", "type": "card", "method_metadata": map[string]int{"exp_year": now.Year() + 1, "exp_month": 1}}}, want: true},
		{name: "expired card", status: 200, items: []map[string]any{{"id": "pm_old", "type": "card", "method_metadata": map[string]int{"exp_year": now.Year() - 1, "exp_month": 1}}}},
		{name: "unavailable provider", status: 503, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/customers/external/org_test/payment-methods", r.URL.Path)
				require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				w.Header().Set(apiVersionHeader, APIVersion())
				w.WriteHeader(test.status)
				_ = json.NewEncoder(w).Encode(map[string]any{"items": test.items, "pagination": map[string]int{"max_page": 1}})
			}))
			defer server.Close()
			got, err := NewClient(server.URL, "token", server.Client()).HasPaymentMethod(context.Background(), "org_test")
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got, test.name)
		})
	}
}
