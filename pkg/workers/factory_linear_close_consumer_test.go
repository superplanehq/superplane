package workers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__FactoryLinearCloseConsumer(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	completedMessage := func(order *models.FactoryWorkOrder) messages.FactoryWorkOrderNotificationMessage {
		return messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: r.Organization.ID.String(),
			FactoryID:      order.FactoryID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			FromState:      models.FactoryWorkOrderStateOpen,
			ToState:        models.FactoryWorkOrderStateClosed,
			Result:         models.FactoryWorkOrderResultCompleted,
		}
	}

	t.Run("moves the originating linear issue to done on completed", func(t *testing.T) {
		order, httpCtx := seedLinearWorkOrder(t, r, "ENG-42", "project-1", []string{"project-1"}, nil,
			linearMockResponses(
				http.StatusOK, linearIssueResponse("ENG-42", "project-1", "In Progress", "started"),
				http.StatusOK, linearStatesResponse(),
				http.StatusOK, `{"data":{"issueUpdate":{"success":true,"issue":{"id":"issue-1"}}}}`,
			))

		require.NoError(t, newLinearCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 3)
		assert.Equal(t, "state-done", linearUpdateStateID(t, httpCtx.Requests[2]))
	})

	t.Run("uses the chosen state", func(t *testing.T) {
		order, httpCtx := seedLinearWorkOrder(t, r, "ENG-7", "project-1", []string{"project-1"}, map[string]any{
			"linearMoveOnComplete":  true,
			"linearCompletionState": "In Review",
		}, linearMockResponses(
			http.StatusOK, linearIssueResponse("ENG-7", "project-1", "In Progress", "started"),
			http.StatusOK, linearStatesResponse(),
			http.StatusOK, `{"data":{"issueUpdate":{"success":true,"issue":{"id":"issue-1"}}}}`,
		))

		require.NoError(t, newLinearCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 3)
		assert.Equal(t, "state-review", linearUpdateStateID(t, httpCtx.Requests[2]))
	})

	t.Run("moves an issue without a project filter", func(t *testing.T) {
		order, httpCtx := seedLinearWorkOrder(t, r, "ENG-9", "project-9", nil, nil,
			linearMockResponses(
				http.StatusOK, linearIssueResponse("ENG-9", "project-9", "Todo", "unstarted"),
				http.StatusOK, linearStatesResponse(),
				http.StatusOK, `{"data":{"issueUpdate":{"success":true,"issue":{"id":"issue-1"}}}}`,
			))

		require.NoError(t, newLinearCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 3)
	})

	t.Run("skips when the issue is already completed", func(t *testing.T) {
		order, httpCtx := seedLinearWorkOrder(t, r, "ENG-1", "project-1", []string{"project-1"}, nil,
			linearMockResponses(
				http.StatusOK, linearIssueResponse("ENG-1", "project-1", "Done", "completed"),
			))

		require.NoError(t, newLinearCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 1)
	})

	t.Run("skips an unknown state without failing complete", func(t *testing.T) {
		order, httpCtx := seedLinearWorkOrder(t, r, "ENG-8", "project-1", []string{"project-1"}, map[string]any{
			"linearMoveOnComplete":  true,
			"linearCompletionState": "Shipped",
		}, linearMockResponses(
			http.StatusOK, linearIssueResponse("ENG-8", "project-1", "In Progress", "started"),
			http.StatusOK, linearStatesResponse(),
		))

		require.NoError(t, newLinearCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 2)
		reloaded, err := models.FindUnscopedWorkOrder(db, order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
		assert.Equal(t, models.FactoryWorkOrderResultCompleted, reloaded.Result)
	})

	t.Run("does not update linear when move is off", func(t *testing.T) {
		order, httpCtx := seedLinearWorkOrder(t, r, "ENG-3", "project-1", []string{"project-1"}, map[string]any{
			"linearMoveOnComplete": false,
		}, linearMockResponses(
			http.StatusOK, linearIssueResponse("ENG-3", "project-1", "In Progress", "started"),
		))

		require.NoError(t, newLinearCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 1)
	})

	t.Run("skips rejected closes", func(t *testing.T) {
		order, httpCtx := seedLinearWorkOrder(t, r, "ENG-4", "project-1", []string{"project-1"}, nil, nil)
		message := completedMessage(order)
		message.Result = models.FactoryWorkOrderResultRejected

		require.NoError(t, newLinearCloseConsumer(r, httpCtx).process(db, message))
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("skips a different project", func(t *testing.T) {
		order, httpCtx := seedLinearWorkOrder(t, r, "OPS-1", "project-2", []string{"project-1"}, nil,
			linearMockResponses(
				http.StatusOK, linearIssueResponse("OPS-1", "project-2", "In Progress", "started"),
			))

		require.NoError(t, newLinearCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 1)
	})

	t.Run("retries a transient linear error", func(t *testing.T) {
		order, httpCtx := seedLinearWorkOrder(t, r, "ENG-5", "project-1", []string{"project-1"}, nil,
			linearMockResponses(
				http.StatusInternalServerError, `{"error":"unavailable"}`,
			))

		err := newLinearCloseConsumer(r, httpCtx).process(db, completedMessage(order))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close Linear issue ENG-5")
	})

	t.Run("consume rejects invalid json", func(t *testing.T) {
		err := newLinearCloseConsumer(r, &supportcontexts.HTTPContext{}).
			Consume(tackle.NewFakeDelivery([]byte("{")))
		require.Error(t, err)
	})
}

func linearIssueResponse(identifier, projectID, stateName, stateType string) string {
	return `{"data":{"issue":{` +
		`"id":"issue-1","identifier":"` + identifier + `","title":"Fix it","url":"https://linear.app/acme/issue/` + identifier + `",` +
		`"state":{"id":"state-open","name":"` + stateName + `","type":"` + stateType + `"},` +
		`"team":{"id":"team-1","key":"ENG","name":"Engineering"},` +
		`"project":{"id":"` + projectID + `","name":"Platform"},` +
		`"labels":{"nodes":[]}}}}`
}

func linearStatesResponse() string {
	return `{"data":{"workflowStates":{"nodes":[` +
		`{"id":"state-todo","name":"Todo","type":"unstarted"},` +
		`{"id":"state-review","name":"In Review","type":"started"},` +
		`{"id":"state-done","name":"Done","type":"completed"}` +
		`],"pageInfo":{"hasNextPage":false}}}}`
}

func newLinearCloseConsumer(r *support.ResourceRegistry, httpCtx *supportcontexts.HTTPContext) *FactoryLinearCloseConsumer {
	consumer := NewFactoryLinearCloseConsumer("amqp://localhost:5672", r.Encryptor, r.Registry, "http://localhost:8000")
	consumer.httpContext = httpCtx
	return consumer
}

func seedLinearWorkOrder(
	t *testing.T,
	r *support.ResourceRegistry,
	identifier string,
	projectID string,
	projects []string,
	metadata map[string]any,
	responses []*http.Response,
) (*models.FactoryWorkOrder, *supportcontexts.HTTPContext) {
	t.Helper()

	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	integration := createReadyLinearIntegration(t, r)
	const triggerNodeID = "trigger"
	integrationID := integration.ID.String()
	if metadata == nil {
		metadata = map[string]any{"linearMoveOnComplete": true}
	}
	projectIDs := make([]any, 0, len(projects))
	for _, project := range projects {
		projectIDs = append(projectIDs, project)
	}

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{
			NodeID: triggerNodeID,
			Type:   models.NodeTypeTrigger,
			Name:   "On Issue",
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "linear.onIssue"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"projects": projectIDs,
				"actions":  []any{"create", "update"},
			}),
			Metadata: datatypes.NewJSONType(metadata),
		}},
		nil,
	)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	bindCanvasNodeInstallation(t, canvas.ID, triggerNodeID, integration.ID)

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	require.NoError(t, err)
	nodes := append([]models.Node(nil), liveVersion.Nodes...)
	for i := range nodes {
		if nodes[i].ID != triggerNodeID {
			continue
		}
		nodes[i].IntegrationID = &integrationID
		nodes[i].Ref = models.NodeRef{Trigger: &models.TriggerRef{Name: "linear.onIssue"}}
		nodes[i].Configuration = map[string]any{"projects": projectIDs, "actions": []any{"create", "update"}}
		nodes[i].Metadata = metadata
	}
	require.NoError(t, db.Model(liveVersion).Update("nodes", datatypes.NewJSONSlice(nodes)).Error)

	_, err = factoryModel.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceLinearIssues)
	require.NoError(t, err)

	originURL := "https://linear.app/acme/issue/" + identifier
	order, err := factoryModel.CreateWorkOrderWithOrigin(db, "Fix "+identifier, "", nil, nil, nil, models.WorkOrderOrigin{
		URL:   originURL,
		Label: identifier,
	})
	require.NoError(t, err)
	completeWorkOrder(t, order)

	return order, &supportcontexts.HTTPContext{Responses: responses}
}

func createReadyLinearIntegration(t *testing.T, r *support.ResourceRegistry) *models.Integration {
	t.Helper()

	integrationID := uuid.New()
	integration, err := models.CreateIntegration(
		integrationID,
		r.Organization.ID,
		"linear",
		support.RandomName("linear"),
		map[string]any{},
	)
	require.NoError(t, err)
	integration.State = models.IntegrationStateReady
	require.NoError(t, database.Conn().Save(integration).Error)
	saveEncryptedIntegrationSecret(t, r, integration, linear.OAuthAccessToken, "linear-token")
	return integration
}

func linearUpdateStateID(t *testing.T, request *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	variables, _ := payload["variables"].(map[string]any)
	input, _ := variables["input"].(map[string]any)
	stateID, _ := input["stateId"].(string)
	return stateID
}

func linearMockResponses(pairs ...any) []*http.Response {
	responses := make([]*http.Response, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		status, _ := pairs[i].(int)
		body, _ := pairs[i+1].(string)
		responses = append(responses, &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		})
	}
	return responses
}
