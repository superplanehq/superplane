package public

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/test/support"
)

func runnerTaskLogsGET(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	resource *support.ResourceRegistry,
	canvasID, executionID uuid.UUID,
	cursor string,
) *httptest.ResponseRecorder {
	t.Helper()
	return runnerTaskLogsGETAs(
		t,
		server,
		signer,
		resource.Account.ID,
		resource.Organization.ID,
		canvasID,
		executionID,
		cursor,
	)
}

func runnerTaskLogsGETAs(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	accountID, organizationID, canvasID, executionID uuid.UUID,
	cursor string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/api/v1/canvases/%s/node-executions/%s/runner-logs?after=%s",
			canvasID,
			executionID,
			url.QueryEscape(cursor),
		),
		nil,
	)
	request.Header.Set("x-organization-id", organizationID.String())
	token, err := authentication.GenerateAccountToken(
		signer,
		accountID.String(),
		time.Now(),
		time.Hour,
	)
	require.NoError(t, err)
	request.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	return response
}
