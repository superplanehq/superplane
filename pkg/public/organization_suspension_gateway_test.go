package public

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

type suspensionChatProvider struct {
	agents.Provider
	createCalls int
	interrupted []string
}

func TestSuspendedExecutionCallbackFinishesWithoutStartingDownstreamWork(t *testing.T) {
	server, resource, cookie := setupAdminTestServer(t)
	server.InitRouter()
	callbackCalls := 0
	resource.Registry.Actions["suspensionCallback"] = impl.NewDummyAction(impl.DummyActionOptions{
		Name:  "suspensionCallback",
		Hooks: []core.Hook{{Name: "complete", Type: core.HookTypeUser}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			callbackCalls++
			return ctx.ExecutionState.Emit("passed", "completion", []any{map[string]any{"result": "done"}})
		},
	})
	registerTestGRPCGateway(t, server, resource.AuthService, resource.Registry, resource.Encryptor, support.NewOIDCProvider())
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, resource.Organization.ID, resource.User, []models.CanvasNode{
		{NodeID: "source", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "suspensionCallback"}})},
		{NodeID: "downstream", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
	}, []models.Edge{{SourceID: "source", TargetID: "downstream", Channel: "passed"}})
	root := support.EmitCanvasEventForNode(t, canvas.ID, "source", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "source", root.ID, root.ID)
	require.NoError(t, db.Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
	require.NoError(t, resource.Organization.Suspend(db, resource.Account.ID, "Stop downstream work"))
	rawToken := uuid.NewString()
	require.NoError(t, models.CreateUserAPIToken(db, models.NewUserAPIToken(resource.User, "Callback test", crypto.HashToken(rawToken))))
	for _, token := range []string{"", rawToken} {
		response := execRequest(server, requestParams{
			method: "POST", path: "/api/v1/canvases/" + canvas.ID.String() + "/executions/" + execution.ID.String() + "/hooks/complete",
			body: []byte(`{"parameters":{}}`), contentType: "application/json", authCookie: cookie, authToken: token,
			headers: map[string]string{"x-organization-id": resource.Organization.ID.String()},
		})
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	}
	require.Equal(t, 2, callbackCalls)
	stored, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeExecutionStateFinished, stored.State)
	require.Equal(t, models.CanvasNodeExecutionResultPassed, stored.Result)
	outputs, err := models.FindCanvasEventsForExecutions(db, []string{execution.ID.String()})
	require.NoError(t, err)
	require.Len(t, outputs, 1)
	router := workers.NewEventRouter("")
	require.NoError(t, router.LockAndProcessEvent(log.NewEntry(log.StandardLogger()), outputs[0], time.Now()))
	var queued int64
	require.NoError(t, db.Model(&models.CanvasNodeQueueItem{}).Where("workflow_id = ?", canvas.ID).Count(&queued).Error)
	require.Zero(t, queued)
	var downstream int64
	require.NoError(t, db.Model(&models.CanvasNodeExecution{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "downstream").Count(&downstream).Error)
	require.Zero(t, downstream)
}

func (p *suspensionChatProvider) Name() string { return "anthropic" }

func (p *suspensionChatProvider) ToolSchemaRevision() string { return "current" }

func (p *suspensionChatProvider) CreateSession(context.Context, agents.CreateSessionOptions) (*agents.CreateSessionResult, error) {
	p.createCalls++
	return nil, errors.New("suspended lookup must not provision or refresh a provider session")
}

func (p *suspensionChatProvider) InterruptSession(_ context.Context, sessionID string) error {
	p.interrupted = append(p.interrupted, sessionID)
	return nil
}

func TestSuspendedCanvasAgentChatLookup(t *testing.T) {
	server, resource, cookie := setupAdminTestServer(t)
	server.InitRouter()
	provider := &suspensionChatProvider{}
	services := testGRPCServices(t, resource.AuthService, resource.Registry, resource.Encryptor, support.NewOIDCProvider())
	services.Agents = grpc.NewAgentsService(agents.NewService(provider, resource.AuthService))
	require.NoError(t, server.RegisterGRPCGateway(services))
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(resource.Organization.ID, features.FeatureClaudeManagedAgents))
	canvas, _ := support.CreateCanvas(t, resource.Organization.ID, resource.User, nil, nil)
	emptyCanvas, _ := support.CreateCanvas(t, resource.Organization.ID, resource.User, nil, nil)
	session := &models.AgentSession{
		OrganizationID: resource.Organization.ID, UserID: resource.User, CanvasID: canvas.ID,
		Provider: "anthropic", ProviderSessionID: "existing-provider-session",
		AgentToolSchemaRevision: "outdated", Status: models.AgentSessionStatusStreaming,
	}
	require.NoError(t, models.CreateAgentSessionInTransaction(db, session))
	message := &models.AgentSessionMessage{SessionID: session.ID, Role: models.AgentMessageRoleUser, Content: "Prior conversation"}
	require.NoError(t, models.AppendAgentSessionMessage(message))
	require.NoError(t, resource.Organization.Suspend(db, resource.Account.ID, "Pause agent work"))
	rawToken := uuid.NewString()
	require.NoError(t, models.CreateUserAPIToken(db, models.NewUserAPIToken(resource.User, "Suspension chat test", crypto.HashToken(rawToken))))
	for _, token := range []string{"", rawToken} {
		request := func(method, path string) (int, string) {
			response := execRequest(server, requestParams{
				method: method, path: path, body: []byte(`{}`), contentType: "application/json",
				authCookie: cookie, authToken: token, headers: map[string]string{"x-organization-id": resource.Organization.ID.String()},
			})
			return response.Code, response.Body.String()
		}
		code, body := request("GET", "/api/v1/agents/canvases/"+canvas.ID.String()+"/chat")
		require.Equal(t, http.StatusOK, code, body)
		var lookup struct {
			Chat struct {
				ID string `json:"id"`
			} `json:"chat"`
		}
		require.NoError(t, json.Unmarshal([]byte(body), &lookup))
		require.Equal(t, session.ID.String(), lookup.Chat.ID)
		chatPath := "/api/v1/agents/chats/" + lookup.Chat.ID
		code, body = request("GET", chatPath+"/messages")
		require.Equal(t, http.StatusOK, code, body)
		require.Contains(t, body, message.Content)
		code, body = request("POST", chatPath+"/interrupt")
		require.Equal(t, http.StatusOK, code, body)
		code, body = request("GET", "/api/v1/agents/canvases/"+emptyCanvas.ID.String()+"/chat")
		require.Equal(t, http.StatusBadRequest, code, body)
		require.NotContains(t, body, "support session")
	}
	require.Zero(t, provider.createCalls)
	require.Equal(t, []string{session.ProviderSessionID, session.ProviderSessionID}, provider.interrupted)
	stored, err := models.FindAgentSessionForUser(resource.Organization.ID, resource.User, session.ID)
	require.NoError(t, err)
	require.Equal(t, session.ProviderSessionID, stored.ProviderSessionID)
	require.Equal(t, "outdated", stored.AgentToolSchemaRevision)
	require.Equal(t, models.AgentSessionStatusIdle, stored.Status)
	emptySession, err := agents.NewService(provider, resource.AuthService).FindCanvasSession(resource.Organization.ID, resource.User, emptyCanvas.ID)
	require.NoError(t, err)
	require.Nil(t, emptySession)
}

// The gateway owns transport authentication and privacy. Model lifecycle tests
// cannot detect a route registered outside the admin authentication boundary.
func TestOrganizationSuspensionGateway(t *testing.T) {
	server, resource, cookie := setupAdminTestServer(t)
	server.InitRouter()
	registerTestGRPCGateway(t, server, resource.AuthService, resource.Registry, resource.Encryptor, support.NewOIDCProvider())
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(resource.Organization.ID, features.FeatureClaudeManagedAgents))
	rawToken := uuid.NewString()
	require.NoError(t, models.CreateUserAPIToken(db, models.NewUserAPIToken(resource.User, "Suspension test", crypto.HashToken(rawToken))))
	base := "/admin/api/organizations/" + resource.Organization.ID.String()
	request := func(method, path, body, token string) int {
		response := execRequest(server, requestParams{method: method, path: path, body: []byte(body), contentType: "application/json", authCookie: cookie, authToken: token})
		return response.Code
	}
	require.Equal(t, http.StatusBadRequest, request("POST", base+"/suspend", `{"reason":"  "}`, ""))
	require.Equal(t, http.StatusBadRequest, request("POST", "/admin/api/organizations/invalid/suspend", `{"reason":"abuse"}`, ""))
	require.Equal(t, http.StatusNotFound, request("GET", "/admin/api/organizations/"+uuid.NewString()+"/suspension-history", "", ""))
	require.Equal(t, http.StatusOK, request("POST", base+"/suspend", `{"reason":"  Private abuse reason  "}`, ""))
	require.Equal(t, http.StatusOK, request("POST", base+"/suspend", `{"reason":"Replacement"}`, rawToken))
	events, err := models.ListOrganizationSuspensionEvents(db, resource.Organization.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "Private abuse reason", events[0].Reason)
	require.Equal(t, resource.Account.ID, events[0].AccountID)
	require.Equal(t, http.StatusConflict, request("POST", base+"/reactivate", `{}`, rawToken))
	for _, token := range []string{"", rawToken} {
		member := execRequest(server, requestParams{method: "GET", path: "/api/v1/organizations/" + resource.Organization.ID.String(), authCookie: cookie, authToken: token, headers: map[string]string{"x-organization-id": resource.Organization.ID.String()}})
		require.Equal(t, http.StatusOK, member.Code, member.Body.String())
		require.NotContains(t, member.Body.String(), events[0].Reason)
		require.NotContains(t, member.Body.String(), "accountId")
		var response struct {
			Organization struct {
				Suspended bool `json:"suspended"`
			} `json:"organization"`
		}
		require.NoError(t, json.Unmarshal(member.Body.Bytes(), &response))
		require.True(t, response.Organization.Suspended)
		write := execRequest(server, requestParams{method: "PATCH", path: "/api/v1/organizations/" + resource.Organization.ID.String(), body: []byte(`{"organization":{"metadata":{"name":"Changed"}}}`), contentType: "application/json", authCookie: cookie, authToken: token, headers: map[string]string{"x-organization-id": resource.Organization.ID.String()}})
		require.Equal(t, http.StatusForbidden, write.Code, write.Body.String())
		for _, operation := range []struct {
			method, path string
			status       int
		}{
			{"POST", "/api/v1/agents/chats/" + uuid.NewString() + "/messages", http.StatusForbidden},
			{"POST", "/api/v1/agents/canvases/" + uuid.NewString() + "/chat/reset", http.StatusForbidden},
			{"POST", "/api/v1/agents/chats/" + uuid.NewString() + "/outcome", http.StatusForbidden},
			// The test installation has no agent provider. Allowed operations reach
			// its availability check instead of failing suspension authorization.
			{"POST", "/api/v1/agents/chats/" + uuid.NewString() + "/interrupt", http.StatusServiceUnavailable},
			{"GET", "/api/v1/agents/chats/" + uuid.NewString() + "/messages", http.StatusServiceUnavailable},
		} {
			response := execRequest(server, requestParams{method: operation.method, path: operation.path, body: []byte(`{}`), contentType: "application/json", authCookie: cookie, authToken: token, headers: map[string]string{"x-organization-id": resource.Organization.ID.String()}})
			require.Equal(t, operation.status, response.Code, response.Body.String())
		}
	}
	require.NoError(t, workers.NewOrganizationSuspensionWorker().Process(t.Context()))
	require.Equal(t, http.StatusOK, request("POST", base+"/reactivate", `{}`, rawToken))
	require.Equal(t, http.StatusOK, request("POST", base+"/reactivate", `{}`, ""))
	events, err = models.ListOrganizationSuspensionEvents(db, resource.Organization.ID)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, models.OrganizationSuspensionActionReactivate, events[0].Action)
	require.NotNil(t, events[1].CancellationCompletedAt)
	// The same authenticated member loses private admin access without losing reads.
	require.NoError(t, db.Model(resource.Account).Update("installation_admin", false).Error)
	for _, suffix := range []string{"suspend", "reactivate", "suspension-history"} {
		method := "POST"
		if suffix == "suspension-history" {
			method = "GET"
		}
		for _, token := range []string{"", rawToken} {
			require.Equal(t, http.StatusNotFound, request(method, base+"/"+suffix, `{"reason":"abuse"}`, token))
		}
	}
}

func TestSuspensionAdminStatusFilterIncludesPinnedMatches(t *testing.T) {
	server, resource, cookie := setupAdminTestServer(t)
	db := database.DB(t.Context())
	require.NoError(t, resource.Organization.Suspend(db, resource.Account.ID, "Filter target"))
	require.NoError(t, models.PinOrganizationForAccount(db, resource.Account.ID, resource.Organization.ID))
	for _, status := range []string{"active", "suspended", "all"} {
		response := execRequest(server, requestParams{method: "GET", path: "/admin/api/organizations?status=" + status + "&search=" + resource.Organization.ID.String() + "&limit=1&offset=0&sort_by=name&sort_direction=asc", authCookie: cookie})
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var page adminOrganizationListResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &page))
		require.Empty(t, page.Items)
		require.Zero(t, page.Total)
		if status == "active" {
			require.Empty(t, page.Pinned)
			require.Zero(t, page.MatchTotal)
			continue
		}
		require.EqualValues(t, 1, page.MatchTotal)
		require.Len(t, page.Pinned, 1)
		require.True(t, page.Pinned[0].Suspended)
	}
}
