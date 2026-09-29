package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
	"github.com/superplanehq/superplane/test/support"
)

func TestPublicFactoryBoardHidesPrivateWorkspace(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, _ := setupTestServer(r, t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))

	factory, line := openPublicLine(t, r, false)
	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   publicBoardPath(r, factory, line),
	})
	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestPublicFactoryBoardReturnsDisplayFieldsOnly(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, _ := setupTestServer(r, t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))

	factory, line := openPublicLine(t, r, true)
	order, err := factory.CreateWorkOrder(database.Conn(), "Ship the board", "secret-description-do-not-leak", nil, nil, nil)
	require.NoError(t, err)
	hidden, err := models.CreateFactory(database.Conn(), r.Organization.ID, "Hidden Workspace", "other", "HID")
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   publicBoardPath(r, factory, line),
	})
	require.Equal(t, http.StatusOK, response.Code)

	body := response.Body.String()
	assert.Contains(t, body, "Ship the board")
	assert.Contains(t, body, factory.Name)
	assert.NotContains(t, body, "secret-description-do-not-leak")
	assert.NotContains(t, body, "description")
	assert.NotContains(t, body, order.ID.String())
	assert.NotContains(t, body, "runId")
	assert.NotContains(t, body, "cost")
	assert.NotContains(t, body, "ctaUrl")
	assert.NotContains(t, body, hidden.Name)
	assert.NotContains(t, body, hidden.ID.String())

	var board struct {
		Columns []struct {
			Cards []struct {
				Title string `json:"title"`
			} `json:"cards"`
		} `json:"columns"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &board))
	require.NotEmpty(t, board.Columns)
	assert.Equal(t, "Ship the board", board.Columns[0].Cards[0].Title)
}

func TestAnonymousFactoryOrderRoutesStayUnauthorized(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		jwt.NewSigner("test"),
		support.NewOIDCProvider(),
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService,
		false,
	)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "desc", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(database.Conn(), "Private task", "secret", nil, nil, nil)
	require.NoError(t, err)

	paths := []string{
		"/api/v1/factories/" + factory.ID.String() + "/orders",
		"/api/v1/factories/" + factory.ID.String() + "/orders/" + order.ID.String(),
		"/api/v1/factories/" + factory.ID.String() + "/orders/" + order.ID.String() + "/events",
	}
	for _, path := range paths {
		response := execRequest(server, requestParams{
			method: http.MethodGet,
			path:   path,
			headers: map[string]string{
				"X-Organization-Id": r.Organization.ID.String(),
			},
		})
		assert.Equal(t, http.StatusUnauthorized, response.Code, path)
		assert.NotContains(t, response.Body.String(), "Private task", path)
		assert.NotContains(t, response.Body.String(), "secret", path)
	}
}

func TestPublicFactoryBoardSocketRejectsPrivateWorkspace(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, _ := setupTestServer(r, t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))

	factory, line := openPublicLine(t, r, false)
	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   publicBoardSocketPath(r, factory, line),
	})
	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestPublicFactoryBoardSocketSendsBoardChangedOnly(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, _ := setupTestServer(r, t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	server.WebsocketHub().Run()

	factory, line := openPublicLine(t, r, true)
	httpServer := httptest.NewServer(server.Router)
	defer httpServer.Close()

	parsed, err := url.Parse(httpServer.URL)
	require.NoError(t, err)
	header := http.Header{}
	header.Set("Origin", "http://localhost:8000")
	conn, resp, err := websocket.DefaultDialer.Dial("ws://"+parsed.Host+publicBoardSocketPath(r, factory, line), header)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()

	topic := eventdistributer.PublicLineTopic(line.ID.String())
	require.Eventually(t, func() bool {
		return server.WebsocketHub().WorkflowSubscriberCount(topic) == 1
	}, 2*time.Second, 5*time.Millisecond)

	orderID := "11111111-1111-1111-1111-111111111111"
	err = eventdistributer.BroadcastFactoryWorkOrderUpdated(server.WebsocketHub(), &pb.FactoryWorkOrderUpdatedMessage{
		FactoryId: factory.ID.String(),
		OrderId:   orderID,
		Reason:    "moved",
	})
	require.NoError(t, err)

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, payload, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.JSONEq(t, `{"event":"board_changed"}`, string(payload))
	assert.NotContains(t, string(payload), orderID)
	assert.NotContains(t, string(payload), "moved")
	assert.NotContains(t, string(payload), factory.ID.String())
}

func TestMemberFactorySocketStaysUnauthorizedWithoutSession(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, _ := setupTestServer(r, t)

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "desc", "")
	require.NoError(t, err)
	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   "/ws/factories/" + factory.ID.String(),
	})
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func openPublicLine(t *testing.T, r *support.ResourceRegistry, public bool) (*models.Factory, *models.FactoryLine) {
	t.Helper()
	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "desc", "")
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, database.Conn().Model(factory).Updates(map[string]any{
		"onboarding_completed_at": now,
		"public":                  public,
	}).Error)
	factory.OnboardingCompletedAt = &now
	factory.Public = public
	line, err := factory.CreateLine(database.Conn(), "Main", nil)
	require.NoError(t, err)
	return factory, line
}

func publicBoardPath(r *support.ResourceRegistry, factory *models.Factory, line *models.FactoryLine) string {
	return "/api/v1/public/organizations/" + r.Organization.Slug + "/workspaces/" + factory.Key + "/lines/" + line.ID.String() + "/board"
}

func publicBoardSocketPath(r *support.ResourceRegistry, factory *models.Factory, line *models.FactoryLine) string {
	return "/ws/public/organizations/" + r.Organization.Slug + "/workspaces/" + factory.Key + "/lines/" + line.ID.String()
}
