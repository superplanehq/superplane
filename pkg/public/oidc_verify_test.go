package public

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/git/inmemory"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestHandleVerifyOIDCToken(t *testing.T) {
	r := support.Setup(t)
	issuer := "http://superplane.test"
	provider, err := oidc.NewProviderFromKeyDir(issuer, filepath.Join("..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)

	canvas, nodes := createSemaphoreDeployCanvas(t, r)
	node := nodes[0]
	event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, node.NodeID, "default", nil, map[string]any{})
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node.NodeID, event.ID, event.ID)

	token, err := oidc.SignExecutionToken(provider, oidc.ExecutionTokenInput{
		OrganizationID: r.Organization.ID.String(),
		CanvasID:       canvas.ID.String(),
		NodeID:         node.NodeID,
		ExecutionID:    execution.ID.String(),
		Component:      "semaphore.runWorkflow",
		ProjectID:      "project-123",
		PipelineFile:   ".semaphore/deploy.yml",
	})
	require.NoError(t, err)

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

func TestAuthorizeExecutionToken(t *testing.T) {
	r := support.Setup(t)
	issuer := "http://superplane.test"
	provider, err := oidc.NewProviderFromKeyDir(issuer, filepath.Join("..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)

	canvas, nodes := createSemaphoreDeployCanvas(t, r)
	node := nodes[0]
	event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, node.NodeID, "default", nil, map[string]any{})
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node.NodeID, event.ID, event.ID)

	claims, err := oidc.ValidateExecutionToken(provider, mustSignToken(t, provider, oidc.ExecutionTokenInput{
		OrganizationID: r.Organization.ID.String(),
		CanvasID:       canvas.ID.String(),
		NodeID:         node.NodeID,
		ExecutionID:    execution.ID.String(),
		Component:      "semaphore.runWorkflow",
		ProjectID:      "project-123",
		PipelineFile:   ".semaphore/deploy.yml",
	}))
	require.NoError(t, err)
	require.NoError(t, authorizeExecutionToken(database.Conn(), claims))
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
}

func createSemaphoreDeployCanvas(t *testing.T, r *support.ResourceRegistry) (*models.Canvas, []models.CanvasNode) {
	t.Helper()

	return support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "deploy",
			Name:   "Deploy",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "semaphore.runWorkflow"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"project":      "project-123",
				"pipelineFile": ".semaphore/deploy.yml",
			}),
			Metadata: datatypes.NewJSONType(map[string]any{
				"project": map[string]any{"id": "project-123"},
			}),
		},
	}, nil)
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

func mustSignToken(t *testing.T, provider oidc.Provider, input oidc.ExecutionTokenInput) string {
	t.Helper()
	token, err := oidc.SignExecutionToken(provider, input)
	require.NoError(t, err)
	return token
}
