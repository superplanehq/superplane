package models

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

func TestFactoryWorkOrder_CreateStartsAsDraft(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "create-draft")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "New order", "", &userID, nil)
	require.NoError(t, err)
	assert.Equal(t, FactoryWorkOrderStateDraft, order.State)
	assert.Equal(t, "", order.Result)
	assert.False(t, order.StateUpdatedAt.IsZero(), "state_updated_at must be set on create so the failure fence has a baseline")

	events, err := order.ListEvents(database.Conn(), 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, factory.EventTypeOrderStatusUpdated, events[0].Type)

	var payload factory.WorkOrderStatusUpdated
	require.NoError(t, json.Unmarshal(events[0].Data, &payload))
	assert.Equal(t, "", payload.FromState)
	assert.Equal(t, FactoryWorkOrderStateDraft, payload.ToState)
}

func TestFactoryWorkOrder_StateUpdatedAtOnlyBumpsOnLifecycleTransitions(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "state-updated-at")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Fenced", "", &userID, nil)
	require.NoError(t, err)
	createdFence := order.StateUpdatedAt

	//
	// Reload after an assignee update — updated_at should bump but the
	// lifecycle fence must stay put so a reassign never hides a
	// pre-existing failure in the UI.
	//
	time.Sleep(5 * time.Millisecond)
	require.NoError(t, order.UpdateAssignees(database.Conn(), []uuid.UUID{userID}, userID))

	reloaded, err := factoryModel.FindWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)
	assert.Equal(t, createdFence.UnixNano(), reloaded.StateUpdatedAt.UnixNano(),
		"UpdateAssignees must not bump state_updated_at")
	assert.True(t, reloaded.UpdatedAt.After(createdFence) || reloaded.UpdatedAt.Equal(createdFence),
		"UpdateAssignees is still expected to bump updated_at for cache invalidation")

	//
	// A real lifecycle transition must bump the fence.
	//
	time.Sleep(5 * time.Millisecond)
	require.NoError(t, reloaded.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateOpen,
		Actor:   &userID,
	}))
	assert.True(t, reloaded.StateUpdatedAt.After(createdFence),
		"UpdateStatus must bump state_updated_at")
}

func TestFactoryWorkOrder_UpdateStatusTransitions(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "transitions")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Lifecycle", "", &userID, nil)
	require.NoError(t, err)

	t.Run("draft to open is allowed and emits opened event", func(t *testing.T) {
		require.NoError(t, order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateOpen,
			Actor:   &userID,
		}))
		assert.Equal(t, FactoryWorkOrderStateOpen, order.State)

		events, err := order.ListEvents(database.Conn(), 10, nil)
		require.NoError(t, err)
		assert.Contains(t, eventTypes(events), factory.EventTypeOrderOpened)
	})

	t.Run("open to closed requires a result", func(t *testing.T) {
		err := order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateClosed,
			Actor:   &userID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderInvalidState)
	})

	t.Run("open to closed with failed result works", func(t *testing.T) {
		require.NoError(t, order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateClosed,
			Result:  FactoryWorkOrderResultFailed,
			Actor:   &userID,
		}))
		assert.Equal(t, FactoryWorkOrderStateClosed, order.State)
		assert.Equal(t, FactoryWorkOrderResultFailed, order.Result)

		events, err := order.ListEvents(database.Conn(), 10, nil)
		require.NoError(t, err)
		assert.Contains(t, eventTypes(events), factory.EventTypeOrderClosed)
	})

	t.Run("invalid transition draft to closed is rejected", func(t *testing.T) {
		other, err := factoryModel.CreateWorkOrder(database.Conn(), "Direct close", "", &userID, nil)
		require.NoError(t, err)

		err = other.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateClosed,
			Result:  FactoryWorkOrderResultCompleted,
			Actor:   &userID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderInvalidState)
	})

	t.Run("reopen from closed does not re-emit order.opened", func(t *testing.T) {
		reopened, err := factoryModel.CreateWorkOrder(database.Conn(), "Reopen", "", &userID, nil)
		require.NoError(t, err)

		for _, step := range []FactoryWorkOrderStatusUpdate{
			{ToState: FactoryWorkOrderStateOpen, Actor: &userID},
			{ToState: FactoryWorkOrderStateClosed, Result: FactoryWorkOrderResultCompleted, Actor: &userID},
		} {
			require.NoError(t, reopened.UpdateStatus(database.Conn(), step))
		}

		before, err := reopened.ListEvents(database.Conn(), 50, nil)
		require.NoError(t, err)
		openedBefore := countEventsOfType(before, factory.EventTypeOrderOpened)

		require.NoError(t, reopened.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateOpen,
			Actor:   &userID,
		}))

		after, err := reopened.ListEvents(database.Conn(), 50, nil)
		require.NoError(t, err)
		assert.Equal(t, openedBefore, countEventsOfType(after, factory.EventTypeOrderOpened),
			"reopen should not emit another order.opened event; the paired order.status.updated is authoritative")

		assert.Contains(t, eventTypes(after), factory.EventTypeOrderStatusUpdated,
			"reopen must still record an order.status.updated event")
	})
}

func TestFactoryWorkOrder_RecordCommentAdded(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "comment")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Comment target", "", &userID, nil)
	require.NoError(t, err)

	userIDStr := userID.String()
	require.NoError(t, order.RecordCommentAdded(database.Conn(), "Hello there", factory.WorkOrderCommentAuthor{
		Kind:   factory.CommentAuthorKindUser,
		UserID: &userIDStr,
	}, nil))

	events, err := order.ListEvents(database.Conn(), 10, nil)
	require.NoError(t, err)
	assert.Contains(t, eventTypes(events), factory.EventTypeOrderCommentAdded)

	var comment factory.WorkOrderCommentAdded
	for _, e := range events {
		if e.Type == factory.EventTypeOrderCommentAdded {
			require.NoError(t, json.Unmarshal(e.Data, &comment))
		}
	}
	assert.Equal(t, "Hello there", comment.Body)
	require.NotNil(t, comment.Author)
	assert.Equal(t, factory.CommentAuthorKindUser, comment.Author.Kind)
}

func TestFactoryWorkOrder_CreateArtifact(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "artifact")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Artifact target", "", &userID, nil)
	require.NoError(t, err)

	t.Run("pr requires a url", func(t *testing.T) {
		_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypePR,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
	})

	t.Run("markdown requires a body", func(t *testing.T) {
		_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeMarkdown,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
	})

	t.Run("pr rejects non-http(s) urls", func(t *testing.T) {
		cases := []string{
			"javascript:alert(1)",
			"data:text/html,<script>alert(1)</script>",
			"file:///etc/passwd",
			"mailto:someone@example.com",
			"//evil.example/pull/1",
			"not a url at all",
		}
		for _, url := range cases {
			t.Run(url, func(t *testing.T) {
				_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
					Type: FactoryWorkOrderArtifactTypePR,
					URL:  url,
				})
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
			})
		}
	})

	t.Run("markdown rejects non-http(s) urls when a url is provided", func(t *testing.T) {
		_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeMarkdown,
			URL:  "javascript:alert(1)",
			Body: "note body",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
	})

	t.Run("creates pr and emits event", func(t *testing.T) {
		artifact, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type:      FactoryWorkOrderArtifactTypePR,
			URL:       "https://github.com/example/repo/pull/1",
			Title:     "Draft PR",
			Data:      map[string]any{"number": 1},
			CreatedBy: &userID,
		})
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, FactoryWorkOrderArtifactTypePR, artifact.Type)

		artifacts, err := order.ListArtifacts(database.Conn())
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.Equal(t, artifact.ID, artifacts[0].ID)

		events, err := order.ListEvents(database.Conn(), 10, nil)
		require.NoError(t, err)
		assert.Contains(t, eventTypes(events), factory.EventTypeOrderArtifactAdded)
	})

	t.Run("markdown event payload carries the inline body", func(t *testing.T) {
		artifact, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type:      FactoryWorkOrderArtifactTypeMarkdown,
			Title:     "Design notes",
			Body:      "Investigation notes: retry policy exceeded idempotency window.",
			CreatedBy: &userID,
		})
		require.NoError(t, err)
		require.NotNil(t, artifact)

		events, err := order.ListEvents(database.Conn(), 10, nil)
		require.NoError(t, err)

		artifactEvent := findEventOfType(t, events, factory.EventTypeOrderArtifactAdded)
		var payload factory.WorkOrderArtifactAdded
		require.NoError(t, json.Unmarshal(artifactEvent.Data, &payload))
		require.NotNil(t, payload.Artifact)
		assert.Equal(t, FactoryWorkOrderArtifactTypeMarkdown, payload.Artifact.Type)
		assert.Equal(t, "Investigation notes: retry policy exceeded idempotency window.", payload.Artifact.Body,
			"markdown artifacts must include the body so the timeline can render it without a separate fetch")
	})
}

func setupFactoryWithUser(t *testing.T, prefix string) (org *Organization, userID uuid.UUID, factoryModel *Factory) {
	t.Helper()

	nonce := time.Now().UnixNano()
	orgName := fmt.Sprintf("factory-test-org-%s-%d", prefix, nonce)

	organization, err := CreateOrganization(orgName, "")
	require.NoError(t, err)

	account, err := CreateAccount(
		fmt.Sprintf("Factory User %d", nonce),
		fmt.Sprintf("factory-%s-%d@example.com", prefix, nonce),
	)
	require.NoError(t, err)

	user, err := CreateUser(organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)

	factoryModel, err = CreateFactory(
		database.Conn(),
		organization.ID,
		fmt.Sprintf("factory-%s-%d", prefix, nonce),
		"",
	)
	require.NoError(t, err)

	return organization, user.ID, factoryModel
}

func eventTypes(events []FactoryWorkOrderEvent) []string {
	result := make([]string, 0, len(events))
	for _, e := range events {
		result = append(result, e.Type)
	}
	return result
}

func countEventsOfType(events []FactoryWorkOrderEvent, eventType string) int {
	count := 0
	for _, e := range events {
		if e.Type == eventType {
			count++
		}
	}
	return count
}

func findEventOfType(t *testing.T, events []FactoryWorkOrderEvent, eventType string) FactoryWorkOrderEvent {
	t.Helper()
	for _, e := range events {
		if e.Type == eventType {
			return e
		}
	}
	t.Fatalf("expected event of type %q, got %v", eventType, eventTypes(events))
	return FactoryWorkOrderEvent{}
}
