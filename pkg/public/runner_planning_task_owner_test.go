package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
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
	channel, queueName := taskOwnerUpdateQueue(t)
	assertUpdate := func() {
		delivery, ok, err := channel.Get(queueName, true)
		require.NoError(t, err)
		require.True(t, ok, "ownership changes publish an update")
		var update pb.FactoryWorkOrderUpdatedMessage
		require.NoError(t, proto.Unmarshal(delivery.Body, &update))
		assert.Equal(t, session.DraftWorkOrderID.String(), update.OrderId)
		assert.Equal(t, factoryevents.EventTypeOrderAssigneesUpdated, update.Reason)
	}

	rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner",
		`{"user":"`+uuid.NewString()+`","reason":"Owns the code."}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, "a user outside the organization cannot own tasks")

	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner",
		`{"user":"payments-dev","reason":" https://evil.example/login "}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, "a reason that is only a link is empty after cleanup")

	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner",
		`{"user":"@Payments-Dev","reason":"Changed pkg/payments/charge.go 3 times in 30 days. See https://evil.example"}`)
	assertAssignedOwner(t, rec, owner.ID)
	assertUpdate()

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
	assertAssignedOwner(t, rec, other.ID)
	assertUpdate()
	order, err = factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{other.ID}, order.AssigneeIDs(), "a later call replaces the first owner")
	rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/suggest-owners", `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body = decodeSuggestOwners(t, rec)
	require.Len(t, body.CurrentOwners, 1)
	assert.Equal(t, other.ID, body.CurrentOwners[0].UserID)

	t.Run("same owner succeeds without publishing", func(t *testing.T) {
		rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner", `{"user":"other-dev","reason":"Owns the code."}`)
		assertAssignedOwner(t, rec, other.ID)
		_, published, err := channel.Get(queueName, true)
		require.NoError(t, err)
		assert.False(t, published, "same-owner assignment does not publish another update")
	})

	t.Run("invalid replacements preserve ownership and history", func(t *testing.T) {
		eventsBefore, err := order.ListEvents(db, 0, nil)
		require.NoError(t, err)
		for _, request := range []string{
			`{"user":"` + uuid.NewString() + `","reason":"Owns the code."}`,
			`{"user":"payments-dev","reason":"https://example.com"}`,
		} {
			rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner", request)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		}
		persisted, err := factoryModel.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{other.ID}, persisted.AssigneeIDs())
		eventsAfter, err := order.ListEvents(db, 0, nil)
		require.NoError(t, err)
		assert.Equal(t, eventsBefore, eventsAfter)
		_, published, err := channel.Get(queueName, true)
		require.NoError(t, err)
		assert.False(t, published)
	})

	for _, existing := range []struct {
		name   string
		owners []uuid.UUID
	}{
		{"manual owner", []uuid.UUID{owner.ID}},
		{"several owners", []uuid.UUID{owner.ID, r.User}},
	} {
		t.Run(existing.name, func(t *testing.T) {
			require.NoError(t, order.UpdateAssignees(db, existing.owners, r.User))
			rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner", `{"user":"other-dev","reason":"Owns the code."}`)
			assertAssignedOwner(t, rec, other.ID)
			assertUpdate()
			persisted, err := factoryModel.FindWorkOrder(db, order.ID)
			require.NoError(t, err)
			assert.Equal(t, []uuid.UUID{other.ID}, persisted.AssigneeIDs())
		})
	}
}

func assertAssignedOwner(t *testing.T, rec *httptest.ResponseRecorder, ownerID uuid.UUID) {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Status string          `json:"status"`
		Owner  suggestedMember `json:"owner"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "assigned", body.Status)
	assert.Equal(t, ownerID, body.Owner.UserID)
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

func TestRunnerPlanningOwnerRoutesRejectInactiveTarget(t *testing.T) {
	for _, state := range []string{"ended session", "closed task", "archived task"} {
		t.Run(state, func(t *testing.T) {
			r := support.Setup(t)
			server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
			db := database.DB(t.Context())
			mustTaskOwnerMember(t, r, "late-dev")
			order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
			require.NoError(t, err)
			require.NoError(t, order.UpdateAssignees(db, []uuid.UUID{r.User}, r.User))
			switch state {
			case "ended session":
				require.NoError(t, session.End(db))
			case "closed task":
				_, err := order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateClosed, Result: models.FactoryWorkOrderResultManualRejected})
				require.NoError(t, err)
			case "archived task":
				_, err := order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateClosed, Result: models.FactoryWorkOrderResultArchived})
				require.NoError(t, err)
			}
			eventsBefore, err := order.ListEvents(db, 0, nil)
			require.NoError(t, err)

			rec := postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/owner", `{"user":"late-dev","reason":"Owns the code."}`)
			require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
			rec = postPlanningRunner(t, server, token, "/api/v1/runner/planning-sessions/suggest-owners", `{"paths":["pkg"]}`)
			require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

			order, err = factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
			require.NoError(t, err)
			assert.Equal(t, []uuid.UUID{r.User}, order.AssigneeIDs())
			eventsAfter, err := order.ListEvents(db, 0, nil)
			require.NoError(t, err)
			assert.Equal(t, eventsBefore, eventsAfter)
		})
	}
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

func taskOwnerUpdateQueue(t *testing.T) (*amqp.Channel, string) {
	t.Helper()
	amqpURL, err := config.RabbitMQURL()
	require.NoError(t, err)
	connection, err := amqp.Dial(amqpURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	channel, err := connection.Channel()
	require.NoError(t, err)
	require.NoError(t, channel.ExchangeDeclare(messages.CanvasExchange, "direct", true, false, false, false, nil))
	queue, err := channel.QueueDeclare("", false, true, true, false, nil)
	require.NoError(t, err)
	require.NoError(t, channel.QueueBind(queue.Name, messages.FactoryWorkOrderUpdatedRoutingKey, messages.CanvasExchange, false, nil))
	return channel, queue.Name
}
