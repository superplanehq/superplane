package organizations

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCommandRendersOrganizationText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			writeMeResponse(w)
		case "/api/v1/organizations/" + testOrgID:
			writeOrganizationResponse(w)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	require.NoError(t, (&getCommand{}).Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, "ID: "+testOrgID)
	require.Contains(t, out, "Name: Acme")
	require.Contains(t, out, "Description: Acme corp")
	require.Contains(t, out, "Created At: 2024-01-01T10:00:00Z")
	require.Contains(t, out, "Updated At: 2024-01-02T12:30:00Z")
}

func TestGetCommandAPIErrorPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			writeMeResponse(w)
		case "/api/v1/organizations/" + testOrgID:
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := newTestContext(t, server, "text")
	err := (&getCommand{}).Execute(ctx)
	require.Error(t, err)
}
