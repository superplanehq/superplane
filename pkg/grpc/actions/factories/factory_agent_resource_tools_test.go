package factories

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__ListFactoryAgentResourceToolsReturnsNames(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceAgentResources(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	server := httptest.NewServer(mcpToolsHandler(t, []map[string]any{
		{"name": "search", "description": "Search the catalog.", "annotations": map[string]any{"readOnlyHint": true}},
		{"name": "create_issue", "description": "Create an issue."},
	}))
	t.Cleanup(server.Close)

	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       server.URL,
		Auth:      models.FactoryAgentResourceAuthHeaders,
	})
	require.NoError(t, err)

	response, err := ListFactoryAgentResourceTools(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.ListFactoryAgentResourceToolsRequest{
		FactoryId:  factory.ID.String(),
		ResourceId: resource.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, response.GetTools(), 2)
	assert.Equal(t, "search", response.GetTools()[0].GetName())
	assert.Equal(t, "Search the catalog.", response.GetTools()[0].GetDescription())
	assert.True(t, response.GetTools()[0].GetReadOnly())
	assert.Equal(t, "create_issue", response.GetTools()[1].GetName())
	assert.False(t, response.GetTools()[1].GetReadOnly())
}

func Test__ListFactoryAgentResourceToolsRejectsSkills(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceAgentResources(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "review-copy", true, models.FactoryAgentResourceConfig{
		Source:   models.FactoryAgentResourceSourceInline,
		Markdown: "# Review copy",
	})
	require.NoError(t, err)

	_, err = ListFactoryAgentResourceTools(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.ListFactoryAgentResourceToolsRequest{
		FactoryId:  factory.ID.String(),
		ResourceId: resource.ID.String(),
	})
	require.Error(t, err)
	code, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
	assert.Contains(t, message, "not an MCP server")
}

func Test__ListFactoryAgentResourceToolsRequiresOAuthConnection(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceAgentResources(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mobbin", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://api.mobbin.com/mcp",
		Auth:      models.FactoryAgentResourceAuthOAuth,
	})
	require.NoError(t, err)

	_, err = ListFactoryAgentResourceTools(t.Context(), IntakeDependencies{Encryptor: r.Encryptor}, r.Organization.ID.String(), &pb.ListFactoryAgentResourceToolsRequest{
		FactoryId:  factory.ID.String(),
		ResourceId: resource.ID.String(),
	})
	require.Error(t, err)
	code, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Equal(t, "Connect this MCP server first.", message)
}

func Test__ListFactoryAgentResourceToolsRejectsNonJSON(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceAgentResources(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "<html>not json</html>")
	}))
	t.Cleanup(server.Close)

	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       server.URL,
		Auth:      models.FactoryAgentResourceAuthHeaders,
	})
	require.NoError(t, err)

	_, err = ListFactoryAgentResourceTools(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.ListFactoryAgentResourceToolsRequest{
		FactoryId:  factory.ID.String(),
		ResourceId: resource.ID.String(),
	})
	require.Error(t, err)
	code, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Equal(t, "SuperPlane could not load the tools. Try again.", message)
}

func mcpToolsHandler(t *testing.T, tools []map[string]any) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Method string `json:"method"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		switch payload.Method {
		case "initialize":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{}})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      2,
				"result":  map[string]any{"tools": tools},
			})
		default:
			t.Fatalf("unexpected method %s", payload.Method)
		}
	})
}
