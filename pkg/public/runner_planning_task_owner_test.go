package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerPlanningSuggestOwnersWithoutCodeHistory(t *testing.T) {
	r := support.Setup(t)
	server, _, factoryModel, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	busy := mustTaskOwnerMember(t, r, "busy-dev")
	free := mustTaskOwnerMember(t, r, "free-dev")
	support.CreateUser(t, r, r.Organization.ID)
	_, err := factoryModel.CreateWorkOrder(db, "Other task", "", nil, []uuid.UUID{busy.ID}, nil)
	require.NoError(t, err)

	rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/suggest-owners", `{"paths":["./pkg/payments/","../secrets"]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var body struct {
		TaskHasOwner bool     `json:"task_has_owner"`
		Paths        []string `json:"paths"`
		CodeOwners   []any    `json:"code_owners"`
		HistoryNote  string   `json:"history_note"`
		Members      []struct {
			UserID      uuid.UUID `json:"user_id"`
			GitHubLogin string    `json:"github_login"`
			OpenTasks   int       `json:"open_tasks"`
		} `json:"members"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.False(t, body.TaskHasOwner)
	assert.Equal(t, []string{"pkg/payments"}, body.Paths)
	assert.Empty(t, body.CodeOwners)
	assert.NotEmpty(t, body.HistoryNote, "a factory without a GitHub integration has no code history")
	require.Len(t, body.Members, 2, "members without a linked GitHub account cannot own tasks")
	assert.Equal(t, free.ID, body.Members[0].UserID)
	assert.Equal(t, "free-dev", body.Members[0].GitHubLogin)
	assert.Equal(t, busy.ID, body.Members[1].UserID)
	assert.Equal(t, 1, body.Members[1].OpenTasks)
}

func TestRunnerPlanningAssignOwner(t *testing.T) {
	r := support.Setup(t)
	server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	owner := mustTaskOwnerMember(t, r, "payments-dev")
	other := mustTaskOwnerMember(t, r, "other-dev")
	unlinked := support.CreateUser(t, r, r.Organization.ID)

	published := []messages.FactoryWorkOrderNotificationMessage{}
	restore := messages.SetWorkOrderNotificationPublisherForTest(func(message messages.FactoryWorkOrderNotificationMessage) error {
		published = append(published, message)
		return nil
	})
	defer restore()

	rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner",
		`{"user":"`+unlinked.ID.String()+`","reason":"Owns the code."}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, "a member without a linked GitHub account cannot own tasks")

	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner",
		`{"user":"payments-dev","reason":" https://evil.example/login "}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, "a reason that is only a link is empty after cleanup")

	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner",
		`{"user":"@Payments-Dev","reason":"Changed pkg/payments/charge.go 3 times in 30 days. See https://evil.example"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"assigned"`)

	order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{owner.ID}, order.AssigneeIDs())

	require.Len(t, published, 1)
	assert.Equal(t, factoryevents.EventTypeOrderAssigneesUpdated, published[0].EventType)
	assert.Equal(t, []string{owner.ID.String()}, published[0].AssignedUserIDs)
	assert.Equal(t, "Changed pkg/payments/charge.go 3 times in 30 days. See", published[0].AssignmentReason)
	assert.Empty(t, published[0].ActorUserID)

	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/suggest-owners", `{"paths":["pkg/payments"]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"task_has_owner":true,"paths":["pkg/payments"]}`, rec.Body.String())

	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner", `{"user":"other-dev","reason":"Owns the code."}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"already_owned"`)
	order, err = factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{owner.ID}, order.AssigneeIDs(), "a later call keeps the first owner")
	assert.NotContains(t, order.AssigneeIDs(), other.ID)
	assert.Len(t, published, 1)
}

func TestRunnerPlanningOwnerRoutesRejectEndedSession(t *testing.T) {
	r := support.Setup(t)
	server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	mustTaskOwnerMember(t, r, "late-dev")
	require.NoError(t, session.End(db))

	rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner", `{"user":"late-dev","reason":"Owns the code."}`)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/suggest-owners", `{"paths":["pkg"]}`)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

	order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	assert.Empty(t, order.AssigneeIDs())
}

func TestRunnerPlanningUpdateAssignsFallbackOwner(t *testing.T) {
	scores := `"scores": {
		"clarity": {"score": 3, "summary": "Clear."},
		"complexity": {"score": 2, "summary": "One run."},
		"verifiability": {"score": 3, "summary": "Tests exist."}
	}`

	t.Run("first scored update assigns the least loaded member", func(t *testing.T) {
		r := support.Setup(t)
		server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
		db := database.DB(t.Context())
		busy := mustTaskOwnerMember(t, r, "busy-dev")
		free := mustTaskOwnerMember(t, r, "free-dev")
		_, err := factoryModel.CreateWorkOrder(db, "Other task", "", nil, []uuid.UUID{busy.ID}, nil)
		require.NoError(t, err)

		rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/updates", `{`+scores+`, "spec": "# Plan"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{free.ID}, order.AssigneeIDs())
		assert.Equal(t, models.TaskOwnerSourceFallback, latestOwnerEventSource(t, order))
	})

	t.Run("assigns a fallback owner when an intake check exists", func(t *testing.T) {
		r := support.Setup(t)
		server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
		db := database.DB(t.Context())
		free := mustTaskOwnerMember(t, r, "free-dev")
		order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
		require.NoError(t, err)
		_, err = order.ReportCheck(db, models.FactoryWorkOrderCheckParams{Key: "sentry-severity", Name: "Severity", Score: 2, MaxScore: 3})
		require.NoError(t, err)

		rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/updates", `{`+scores+`, "spec": "# Plan"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		order, err = factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{free.ID}, order.AssigneeIDs())
	})

	t.Run("keeps the owner the agent chose", func(t *testing.T) {
		r := support.Setup(t)
		server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
		db := database.DB(t.Context())
		chosen := mustTaskOwnerMember(t, r, "chosen-dev")
		mustTaskOwnerMember(t, r, "free-dev")
		_, err := factoryModel.CreateWorkOrder(db, "Other task", "", nil, []uuid.UUID{chosen.ID}, nil)
		require.NoError(t, err)

		rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner", `{"user":"chosen-dev","reason":"Owns the code."}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/updates", `{`+scores+`, "spec": "# Plan"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{chosen.ID}, order.AssigneeIDs())
	})

	t.Run("leaves the task without owner when nobody can own it", func(t *testing.T) {
		r := support.Setup(t)
		server, session, factoryModel, token := mustPlanningRunnerSession(t, r)

		rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/updates", `{`+scores+`, "spec": "# Plan"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		order, err := factoryModel.FindWorkOrder(database.DB(t.Context()), *session.DraftWorkOrderID)
		require.NoError(t, err)
		assert.Empty(t, order.AssigneeIDs())
	})
}

func mustTaskOwnerMember(t *testing.T, r *support.ResourceRegistry, login string) *models.User {
	t.Helper()
	user := support.CreateUser(t, r, r.Organization.ID)
	require.NotNil(t, user.AccountID)
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.DB(t.Context()),
		models.NewAccountLinkedAccount(*user.AccountID, models.ProviderGitHub, uuid.NewString(), login, "", ""),
	))
	return user
}

func postPlanningRunner(t *testing.T, server *Server, token, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}

func latestOwnerEventSource(t *testing.T, order *models.FactoryWorkOrder) string {
	t.Helper()
	events, err := order.ListEvents(database.DB(t.Context()), 20, nil)
	require.NoError(t, err)
	for _, event := range events {
		if event.Type != factoryevents.EventTypeOrderAssigneesUpdated {
			continue
		}
		var payload factoryevents.WorkOrderAssigneesUpdated
		require.NoError(t, json.Unmarshal(event.Data, &payload))
		return payload.Source
	}
	return ""
}
