package public

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/ciauth"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/git/inmemory"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/oidc"
	"github.com/superplanehq/superplane/pkg/registry"
)

func TestHandleVerifyOIDCToken(t *testing.T) {
	issuer := "http://superplane.test"
	provider, err := oidc.NewProviderFromKeyDir(issuer, filepath.Join("..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)

	token := signTestExecutionToken(t, provider, map[string]any{
		ciauth.ClaimOrgID:                uuid.NewString(),
		ciauth.ClaimCanvasID:             uuid.NewString(),
		ciauth.ClaimNodeID:               "deploy",
		ciauth.ClaimExecutionID:          uuid.NewString(),
		ciauth.ClaimComponent:            "semaphore.runWorkflow",
		"pipeline_file":                  ".semaphore/deploy.yml",
	})

	server := newVerifyTestServer(t, provider, issuer)

	body, err := json.Marshal(map[string]any{
		"token": token,
		"expected": map[string]any{
			"pipeline_file": ".semaphore/deploy.yml",
		},
	})
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method:      "POST",
		path:        "/api/v1/oidc/verify",
		body:        body,
		contentType: "application/json",
	})

	require.Equal(t, http.StatusOK, response.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, true, payload["valid"])
}

func TestHandleVerifyOIDCTokenRejectsInvalidToken(t *testing.T) {
	provider, err := oidc.NewProviderFromKeyDir("http://superplane.test", filepath.Join("..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)

	server := newVerifyTestServer(t, provider, "http://superplane.test")

	body, err := json.Marshal(map[string]any{"token": "not-a-token"})
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method:      "POST",
		path:        "/api/v1/oidc/verify",
		body:        body,
		contentType: "application/json",
	})

	require.Equal(t, http.StatusUnauthorized, response.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, false, payload["valid"])
	require.Equal(t, verifyOIDCTokenFailedMessage, payload["error"])
}

func TestHandleVerifyOIDCTokenRejectsExpectedClaimMismatch(t *testing.T) {
	issuer := "http://superplane.test"
	provider, err := oidc.NewProviderFromKeyDir(issuer, filepath.Join("..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)

	token := signTestExecutionToken(t, provider, map[string]any{
		ciauth.ClaimOrgID:       uuid.NewString(),
		ciauth.ClaimCanvasID:    uuid.NewString(),
		ciauth.ClaimNodeID:      "deploy",
		ciauth.ClaimExecutionID: uuid.NewString(),
		ciauth.ClaimComponent:   "semaphore.runWorkflow",
	})

	server := newVerifyTestServer(t, provider, issuer)

	body, err := json.Marshal(map[string]any{
		"token": token,
		"expected": map[string]any{
			"canvas_id": uuid.NewString(),
		},
	})
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method:      "POST",
		path:        "/api/v1/oidc/verify",
		body:        body,
		contentType: "application/json",
	})

	require.Equal(t, http.StatusForbidden, response.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, false, payload["valid"])
	require.Equal(t, verifyOIDCTokenFailedMessage, payload["error"])
}

func newVerifyTestServer(t *testing.T, provider oidc.Provider, issuer string) *Server {
	t.Helper()

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	server, err := NewServer(
		&crypto.NoOpEncryptor{},
		reg,
		jwt.NewSigner("test"),
		provider,
		inmemory.NewProvider(),
		"/api/v1",
		"http://localhost",
		issuer,
		"test",
		"/app/templates",
		authService,
		nil,
		false,
	)
	require.NoError(t, err)
	return server
}

func signTestExecutionToken(t *testing.T, provider oidc.Provider, claims map[string]any) string {
	t.Helper()

	executionID, _ := claims[ciauth.ClaimExecutionID].(string)
	if executionID == "" {
		executionID = uuid.NewString()
		claims[ciauth.ClaimExecutionID] = executionID
	}

	token, err := provider.Sign(
		fmt.Sprintf("execution:%s", executionID),
		time.Hour,
		ciauth.ExecutionTokenAudience,
		claims,
	)
	require.NoError(t, err)
	return token
}
