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
	unlinked := support.CreateUser(t, r, r.Organization.ID)
	_, err := factoryModel.CreateWorkOrder(db, "Other task", "", nil, []uuid.UUID{busy.ID, r.User}, nil)
	require.NoError(t, err)
	_, err = factoryModel.CreateWorkOrder(db, "Second task", "", nil, []uuid.UUID{r.User}, nil)
	require.NoError(t, err)

	rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/suggest-owners", `{"paths":["./pkg/payments/","../secrets"]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	body := decodeSuggestOwners(t, rec)
	assert.Equal(t, []string{"pkg/payments"}, body.Paths)
	assert.Empty(t, body.CurrentOwners)
	assert.Empty(t, body.CodeOwners)
	assert.NotEmpty(t, body.HistoryNote, "a factory without a GitHub integration has no code history")

	members := map[uuid.UUID]suggestedMember{}
	for _, member := range body.Members {
		members[member.UserID] = member
	}
	require.Len(t, body.Members, 4, "every human member is listed, with or without GitHub")
	assert.Equal(t, 2, body.Members[3].OpenTasks, "members are sorted by open tasks, fewest first")
	assert.Equal(t, "free-dev", members[free.ID].GitHubLogin)
	assert.Equal(t, 1, members[busy.ID].OpenTasks)
	assert.Empty(t, members[unlinked.ID].GitHubLogin)
	assert.Equal(t, []string{"Operator"}, members[unlinked.ID].Roles)
	assert.Equal(t, []string{"Owner", "Admin"}, members[r.User].Roles)

	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/suggest-owners", `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body = decodeSuggestOwners(t, rec)
	assert.Empty(t, body.Paths)
	assert.Empty(t, body.CodeOwners)
	assert.Empty(t, body.HistoryNote, "without paths SuperPlane does not read code history")
	assert.Len(t, body.Members, 4)
}

type suggestedMember struct {
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	GitHubLogin string    `json:"github_login"`
	Roles       []string  `json:"roles"`
	OpenTasks   int       `json:"open_tasks"`
}

type suggestOwnersBody struct {
	Paths         []string          `json:"paths"`
	CurrentOwners []suggestedMember `json:"current_owners"`
	CodeOwners    []any             `json:"code_owners"`
	HistoryNote   string            `json:"history_note"`
	Members       []suggestedMember `json:"members"`
}

func decodeSuggestOwners(t *testing.T, rec *httptest.ResponseRecorder) suggestOwnersBody {
	t.Helper()
	var body suggestOwnersBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func TestRunnerPlanningAssignOwner(t *testing.T) {
	r := support.Setup(t)
	server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	owner := mustTaskOwnerMember(t, r, "payments-dev")
	other := mustTaskOwnerMember(t, r, "other-dev")

	rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner",
		`{"user":"`+uuid.NewString()+`","reason":"Owns the code."}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, "a user outside the organization cannot own tasks")

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
	source, reason := latestOwnerEvent(t, order)
	assert.Equal(t, models.TaskOwnerSourceAgent, source)
	assert.Equal(t, "Changed pkg/payments/charge.go 3 times in 30 days. See", reason)

	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/suggest-owners", `{"paths":["pkg/payments"]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := decodeSuggestOwners(t, rec)
	require.Len(t, body.CurrentOwners, 1, "an owned task still lists reviewers and members")
	assert.Equal(t, owner.ID, body.CurrentOwners[0].UserID)
	assert.Equal(t, "payments-dev", body.CurrentOwners[0].GitHubLogin)
	assert.NotEmpty(t, body.Members)

	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner", `{"user":"other-dev","reason":"Owns the code."}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"already_owned"`)
	order, err = factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{owner.ID}, order.AssigneeIDs(), "a later call keeps the first owner")
	assert.NotContains(t, order.AssigneeIDs(), other.ID)
}

func TestRunnerPlanningAssignOwnerWithoutGitHub(t *testing.T) {
	r := support.Setup(t)
	server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
	unlinked := support.CreateUser(t, r, r.Organization.ID)

	rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner",
		`{"user":"`+unlinked.ID.String()+`","reason":"The team rules route payment problems to this person."}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	order, err := factoryModel.FindWorkOrder(database.DB(t.Context()), *session.DraftWorkOrderID)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{unlinked.ID}, order.AssigneeIDs())
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

func latestOwnerEvent(t *testing.T, order *models.FactoryWorkOrder) (string, string) {
	t.Helper()
	events, err := order.ListEvents(database.DB(t.Context()), 20, nil)
	require.NoError(t, err)
	for _, event := range events {
		if event.Type != factoryevents.EventTypeOrderAssigneesUpdated {
			continue
		}
		var payload factoryevents.WorkOrderAssigneesUpdated
		require.NoError(t, json.Unmarshal(event.Data, &payload))
		return payload.Source, payload.Reason
	}
	t.Fatal("no assignees updated event")
	return "", ""
}
