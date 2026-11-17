package public

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type canvasReadCredential struct {
	name, token        string
	allowedA, allowedB bool
}

func canvasReadCredentials(t *testing.T, r *support.ResourceRegistry, signer *jwt.Signer, canvasA uuid.UUID) []canvasReadCredential {
	t.Helper()
	key, err := models.CreateAPIKey(database.DB(t.Context()), r.Organization.ID, "canvas-reader", nil, r.User, nil, []string{canvasA.String()})
	require.NoError(t, err)
	require.NoError(t, r.AuthService.AssignRole(key.ID.String(), models.RoleOrgOperator, r.Organization.ID.String(), models.DomainTypeOrganization))
	const keyToken = "dummy-canvas-reader-key"
	require.NoError(t, key.UpdateTokenHash(crypto.HashToken(keyToken)))
	unrestricted, err := models.CreateAPIKey(database.DB(t.Context()), r.Organization.ID, "unrestricted-reader", nil, r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, r.AuthService.AssignRole(unrestricted.ID.String(), models.RoleOrgOperator, r.Organization.ID.String(), models.DomainTypeOrganization))
	const unrestrictedToken = "dummy-unrestricted-reader-key"
	require.NoError(t, unrestricted.UpdateTokenHash(crypto.HashToken(unrestrictedToken)))
	account, err := models.CreateAccount(support.RandomName("no-read")+"@example.com", "No read permission")
	require.NoError(t, err)
	noRead, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	scoped := func(subject uuid.UUID, scopes ...string) string {
		token, err := signer.GenerateScopedToken(jwt.ScopedTokenClaims{Subject: subject.String(), OrgID: r.Organization.ID.String(), Purpose: "canvas-read-test", Scopes: scopes}, time.Hour)
		require.NoError(t, err)
		return token
	}
	return []canvasReadCredential{
		{"restricted API key", keyToken, true, false},
		{"restricted scoped token", scoped(r.User, "canvases:read:"+canvasA.String()), true, false},
		{"write-only scoped token", scoped(r.User, "canvases:write"), false, false},
		{"no organization read permission", scoped(noRead.ID, "canvases:read"), false, false},
		{"global claims cannot expand API key", scoped(key.ID, "canvases:read"), true, false},
		{"API key also requires scoped read", scoped(key.ID, "canvases:write"), false, false},
		{"global scoped read", scoped(r.User, "canvases:read"), true, true},
		{"unrestricted API key", unrestrictedToken, true, true},
	}
}

func canvasReadGET(server *Server, token, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	// Client-supplied grants must not replace the authenticated restrictions.
	request.Header.Set("x-token-scopes", `["canvases:read"]`)
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	return response
}

func observeCanvasReadQueries(t *testing.T) *[]string {
	t.Helper()
	queries := new([]string)
	const callback = "test:canvas-read-resource-queries"
	require.NoError(t, database.Conn().Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if slices.Contains([]string{"workflows", "workflow_versions", "workflow_node_executions", "runner_tasks", "runner_task_log_lifecycles", "organization_llm_settings", "installation_llm_settings"}, tx.Statement.Table) {
			*queries = append(*queries, tx.Statement.Table)
		}
	}))
	t.Cleanup(func() { require.NoError(t, database.Conn().Callback().Query().Remove(callback)) })
	return queries
}
