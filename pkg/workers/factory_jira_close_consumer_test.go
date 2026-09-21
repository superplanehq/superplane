package workers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__FactoryJiraCloseConsumer(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	completedMessage := func(order *models.FactoryWorkOrder) messages.FactoryWorkOrderNotificationMessage {
		return messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: r.Organization.ID.String(),
			FactoryID:      factoryModel.ID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			FromState:      models.FactoryWorkOrderStateOpen,
			ToState:        models.FactoryWorkOrderStateClosed,
			Result:         models.FactoryWorkOrderResultCompleted,
		}
	}

	t.Run("moves the originating jira issue to done on completed", func(t *testing.T) {
		order, httpCtx := seedJiraWorkOrder(t, r, factoryModel, "ENG-42", nil, jiraMockResponses(
			http.StatusOK, `{"key":"ENG-42","fields":{"status":{"name":"In Progress","statusCategory":{"key":"indeterminate"}}}}`,
			http.StatusOK, `{"transitions":[{"id":"31","name":"Resolve","to":{"id":"10003","name":"Done","statusCategory":{"key":"done"}},"fields":{"resolution":{"required":true},"comment":{"required":false}}}]}`,
			http.StatusOK, `[{"id":"10000","name":"Done"}]`,
			http.StatusNoContent, ``,
		))

		require.NoError(t, newJiraCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 4)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/issue/ENG-42")
		assert.Contains(t, httpCtx.Requests[1].URL.Path, "/transitions")
		assert.Contains(t, httpCtx.Requests[3].URL.Path, "/transitions")
		body, err := io.ReadAll(httpCtx.Requests[3].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "31", payload["transition"].(map[string]any)["id"])
		assert.Contains(t, payload["update"].(map[string]any), "comment")
	})

	t.Run("uses the chosen column", func(t *testing.T) {
		order, httpCtx := seedJiraWorkOrder(t, r, factoryModel, "ENG-7", map[string]any{
			"jiraMoveOnComplete":   true,
			"jiraCompletionColumn": "QA",
		}, jiraMockResponses(
			http.StatusOK, `{"key":"ENG-7","fields":{"status":{"name":"In Progress","statusCategory":{"key":"indeterminate"}}}}`,
			http.StatusOK, `{"transitions":[{"id":"21","name":"Start QA","to":{"id":"10004","name":"QA","statusCategory":{"key":"indeterminate"}}}]}`,
			http.StatusNoContent, ``,
		))

		require.NoError(t, newJiraCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 3)
		body, err := io.ReadAll(httpCtx.Requests[2].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "21", payload["transition"].(map[string]any)["id"])
	})

	t.Run("skips when the issue is already in the column", func(t *testing.T) {
		order, httpCtx := seedJiraWorkOrder(t, r, factoryModel, "ENG-1", nil, jiraMockResponses(
			http.StatusOK, `{"key":"ENG-1","fields":{"status":{"name":"Done","statusCategory":{"key":"done"}}}}`,
		))

		require.NoError(t, newJiraCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 1)
	})

	t.Run("skips an unreachable column without failing complete", func(t *testing.T) {
		order, httpCtx := seedJiraWorkOrder(t, r, factoryModel, "ENG-8", map[string]any{
			"jiraMoveOnComplete":   true,
			"jiraCompletionColumn": "Done",
		}, jiraMockResponses(
			http.StatusOK, `{"key":"ENG-8","fields":{"status":{"name":"In Progress","statusCategory":{"key":"indeterminate"}}}}`,
			http.StatusOK, `{"transitions":[{"id":"21","name":"Start","to":{"id":"10002","name":"In Progress","statusCategory":{"key":"indeterminate"}}}]}`,
		))

		require.NoError(t, newJiraCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 2)
		reloaded, err := models.FindUnscopedWorkOrder(db, order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
		assert.Equal(t, models.FactoryWorkOrderResultCompleted, reloaded.Result)
	})

	t.Run("does not call jira when move is off", func(t *testing.T) {
		order, httpCtx := seedJiraWorkOrder(t, r, factoryModel, "ENG-3", map[string]any{
			"jiraMoveOnComplete": false,
		}, nil)

		require.NoError(t, newJiraCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("skips rejected closes", func(t *testing.T) {
		order, httpCtx := seedJiraWorkOrder(t, r, factoryModel, "ENG-4", nil, nil)
		message := completedMessage(order)
		message.Result = models.FactoryWorkOrderResultRejected

		require.NoError(t, newJiraCloseConsumer(r, httpCtx).process(db, message))
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("skips a different project", func(t *testing.T) {
		order, httpCtx := seedJiraWorkOrder(t, r, factoryModel, "OPS-1", nil, nil)

		require.NoError(t, newJiraCloseConsumer(r, httpCtx).process(db, completedMessage(order)))
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("retries a transient jira error", func(t *testing.T) {
		order, httpCtx := seedJiraWorkOrder(t, r, factoryModel, "ENG-5", nil, jiraMockResponses(
			http.StatusInternalServerError, `{"error":"unavailable"}`,
		))

		err := newJiraCloseConsumer(r, httpCtx).process(db, completedMessage(order))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close Jira issue ENG-5")
	})

	t.Run("consume rejects invalid json", func(t *testing.T) {
		err := newJiraCloseConsumer(r, &supportcontexts.HTTPContext{}).
			Consume(tackle.NewFakeDelivery([]byte("{")))
		require.Error(t, err)
	})
}

func newJiraCloseConsumer(r *support.ResourceRegistry, httpCtx *supportcontexts.HTTPContext) *FactoryJiraCloseConsumer {
	consumer := NewFactoryJiraCloseConsumer("amqp://localhost:5672", r.Encryptor, r.Registry, "http://localhost:8000")
	consumer.httpContext = httpCtx
	return consumer
}

func seedJiraWorkOrder(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryModel *models.Factory,
	issueKey string,
	metadata map[string]any,
	responses []*http.Response,
) (*models.FactoryWorkOrder, *supportcontexts.HTTPContext) {
	t.Helper()

	db := database.Conn()
	integration := createReadyJiraIntegration(t, r)
	const triggerNodeID = "trigger"
	integrationID := integration.ID.String()
	if metadata == nil {
		metadata = map[string]any{"jiraMoveOnComplete": true}
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
				Trigger: &models.TriggerRef{Name: "jira.onIssue"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"project": "ENG",
				"events":  []any{"created", "updated"},
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
		nodes[i].Ref = models.NodeRef{Trigger: &models.TriggerRef{Name: "jira.onIssue"}}
		nodes[i].Configuration = map[string]any{"project": "ENG", "events": []any{"created", "updated"}}
		nodes[i].Metadata = metadata
	}
	require.NoError(t, db.Model(liveVersion).Update("nodes", datatypes.NewJSONSlice(nodes)).Error)

	_, err = factoryModel.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceJiraIssues)
	require.NoError(t, err)

	originURL := "https://acme.atlassian.net/browse/" + issueKey
	order, err := factoryModel.CreateWorkOrderWithOrigin(db, "Fix "+issueKey, "", nil, nil, nil, models.WorkOrderOrigin{
		URL:   originURL,
		Label: issueKey,
	})
	require.NoError(t, err)
	completeWorkOrder(t, order)

	return order, &supportcontexts.HTTPContext{Responses: responses}
}

func createReadyJiraIntegration(t *testing.T, r *support.ResourceRegistry) *models.Integration {
	t.Helper()

	integrationID := uuid.New()
	integration, err := models.CreateIntegration(
		integrationID,
		r.Organization.ID,
		"jira",
		support.RandomName("jira"),
		map[string]any{},
	)
	require.NoError(t, err)
	integration.State = models.IntegrationStateReady
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"cloudId":              "jira-cloud-1",
		"siteUrl":              "https://acme.atlassian.net",
		"accessTokenExpiresAt": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
	})
	require.NoError(t, database.Conn().Save(integration).Error)
	saveEncryptedIntegrationSecret(t, r, integration, jira.SecretOAuthAccessToken, "jira-token")
	saveEncryptedIntegrationSecret(t, r, integration, jira.SecretOAuthRefreshToken, "jira-refresh")
	return integration
}

func jiraMockResponses(pairs ...any) []*http.Response {
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
