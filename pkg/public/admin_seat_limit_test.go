package public

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAdminSeatLimit(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	t.Setenv(models.BillingEnabledEnv, "yes")
	planLimit := 10
	plan := models.BillingPlan{Key: "seat_" + uuid.NewString()[:8], Name: "Seats", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: &planLimit, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(database.Conn(), &plan))
	_, err := models.SetAdminOrganizationPlan(database.Conn(), r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	require.NoError(t, database.Conn().Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 3).Error)
	path := "/admin/api/organizations/" + r.Organization.ID.String() + "/seat-limit"
	regular, err := models.CreateAccount("Member", "seats@example.com")
	require.NoError(t, err)
	regularToken, err := authentication.GenerateAccountToken(jwt.NewSigner("test-client-secret"), regular.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	for _, method := range []string{"GET", "PUT", "DELETE"} {
		for _, auth := range []string{"", regularToken} {
			response := execRequest(server, requestParams{method: method, path: path, authCookie: auth, body: []byte(`{"seat_limit":2}`), contentType: "application/json"})
			require.NotEqual(t, http.StatusOK, response.Code)
			require.Contains(t, []int{http.StatusNotFound, http.StatusUnauthorized, http.StatusForbidden}, response.Code)
		}
	}
	for _, body := range []string{`{}`, `{"seat_limit":0}`, `{"seat_limit":-1}`, `{"seat_limit":1.5}`, `{"seat_limit":2147483648}`, `{"seat_limit":"2"}`, `{"seat_limit":true}`, `null`, `{"seat_limit":1} {}`} {
		response := execRequest(server, requestParams{method: "PUT", path: path, authCookie: token, body: []byte(body), contentType: "application/json"})
		require.Equal(t, http.StatusBadRequest, response.Code, body)
	}
	for _, tc := range []struct {
		method, body string
		override     any
		effective    any
	}{
		{"GET", "", nil, float64(10)},
		{"PUT", `{"seat_limit":2147483647}`, map[string]any{"seat_limit": float64(2147483647)}, float64(2147483647)},
		{"PUT", `{"seat_limit":null}`, map[string]any{"seat_limit": nil}, nil},
		{"DELETE", "", nil, float64(10)},
	} {
		response := execRequest(server, requestParams{method: tc.method, path: path, authCookie: token, body: []byte(tc.body), contentType: "application/json"})
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var state map[string]any
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &state))
		require.Equal(t, map[string]any{"plan_limit": float64(10), "override": tc.override, "effective_limit": tc.effective}, state)
	}
}
