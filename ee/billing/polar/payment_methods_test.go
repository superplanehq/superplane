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
	now := time.Now().UTC()
	for _, test := range []struct {
		name      string
		status    int
		items     []map[string]any
		want      bool
		wantError bool
	}{
		{name: "customer without a card", status: 200, items: []map[string]any{}},
		{name: "missing customer", status: 404},
		{name: "confirmed saved card", status: 200, items: []map[string]any{{"id": "pm_card", "type": "card", "method_metadata": map[string]int{"exp_year": now.Year() + 1, "exp_month": 1}}}, want: true},
		{name: "expired card", status: 200, items: []map[string]any{{"id": "pm_old", "type": "card", "method_metadata": map[string]int{"exp_year": now.Year() - 1, "exp_month": 1}}}},
		{name: "expiration month stays usable", status: 200, items: []map[string]any{{"id": "pm_current", "type": "card", "method_metadata": map[string]int{"exp_year": now.Year(), "exp_month": int(now.Month())}}}, want: true},
		{name: "Korean card without expiration", status: 200, items: []map[string]any{{"id": "pm_kr", "type": "kr_card"}}, want: true},
		{name: "Korean card without identity", status: 200, items: []map[string]any{{"type": "kr_card"}}},
		{name: "missing expiration", status: 200, items: []map[string]any{{"id": "pm_invalid", "type": "card"}}},
		{name: "invalid expiration types", status: 200, items: []map[string]any{{"id": "pm_invalid", "type": "card", "method_metadata": map[string]any{"exp_month": "invalid", "exp_year": now.Year() + 1}}}},
		{name: "invalid month", status: 200, items: []map[string]any{{"id": "pm_invalid", "type": "card", "method_metadata": map[string]int{"exp_month": 13, "exp_year": now.Year() + 1}}}},
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

func TestHasPaymentMethodPagination(t *testing.T) {
	for _, tc := range []struct {
		name       string
		usablePage string
		wantPages  []string
		wantError  bool
	}{
		{name: "usable card on first page", usablePage: "1", wantPages: []string{"1"}},
		{name: "usable card on later page", usablePage: "2", wantPages: []string{"1", "2"}},
		{name: "failure before a usable card", wantPages: []string{"1", "2", "3"}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var pages []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				page := r.URL.Query().Get("page")
				pages = append(pages, page)
				w.Header().Set(apiVersionHeader, APIVersion())
				if page == "3" {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				methodType := "unknown"
				if page == tc.usablePage {
					methodType = "kr_card"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"items":      []map[string]any{{"id": "pm_" + page, "type": methodType}},
					"pagination": map[string]int{"max_page": 3},
				})
			}))
			defer server.Close()
			hasCard, err := NewClient(server.URL, "token", server.Client()).HasPaymentMethod(t.Context(), "org")
			require.Equal(t, tc.wantError, err != nil)
			require.Equal(t, tc.usablePage != "", hasCard)
			require.Equal(t, tc.wantPages, pages)
		})
	}
}

func TestPaymentMethodsRejectsMissingItems(t *testing.T) {
	for _, body := range []string{
		`{"pagination":{"max_page":1}}`,
		`{"items":null,"pagination":{"max_page":1}}`,
	} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(apiVersionHeader, APIVersion())
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			client := NewClient(server.URL, "token", server.Client())
			hasCard, err := client.HasPaymentMethod(t.Context(), "org")
			require.ErrorContains(t, err, "missing items")
			require.False(t, hasCard)
			snapshot, err := client.PaymentMethods(t.Context(), "org")
			require.ErrorContains(t, err, "missing items")
			require.False(t, snapshot.Complete, "malformed lists cannot establish card loss")
		})
	}
}

func TestPaymentMethodsRequiresCompletePagination(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		laterStatus             int
		wantComplete, wantError bool
	}{
		{"complete", 200, true, false}, {"later failure", 503, false, true}, {"later missing customer", 404, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pages := []string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				page := r.URL.Query().Get("page")
				pages = append(pages, page)
				w.Header().Set(apiVersionHeader, APIVersion())
				if page == "2" {
					w.WriteHeader(tc.laterStatus)
				}
				items := []map[string]any{{"id": "pm_" + page, "type": "kr_card"}}
				_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "pagination": map[string]int{"max_page": 2}})
			}))
			defer server.Close()
			snapshot, err := NewClient(server.URL, "token", server.Client()).PaymentMethods(t.Context(), "org")
			require.Equal(t, []string{"1", "2"}, pages, "even usable cards on page one require a complete read")
			require.Equal(t, tc.wantError, err != nil)
			require.Equal(t, tc.wantComplete, snapshot.Complete)
			if tc.wantComplete {
				require.Len(t, snapshot.Methods, 2)
				require.True(t, snapshot.HasUsableMethod())
			} else {
				require.Empty(t, snapshot.Methods)
			}
		})
	}
}
