package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestFactoryWorkOrder_CreateStartsAsDraft(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "create-draft")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "New order", "", &userID, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, FactoryWorkOrderStateDraft, order.State)
	assert.Equal(t, "", order.Result)

	events, err := order.ListEvents(database.Conn(), 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, factory.EventTypeOrderStatusUpdated, events[0].Type)

	var payload factory.WorkOrderStatusUpdated
	require.NoError(t, json.Unmarshal(events[0].Data, &payload))
	assert.Equal(t, "", payload.FromState)
	assert.Equal(t, FactoryWorkOrderStateDraft, payload.ToState)
}

func TestResolveFactoryWorkOrderCreatorAutomations(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, userID, _ := setupFactoryWithUser(t, "creator-automations")
	now := time.Now()
	versionID := uuid.New()
	canvas := Canvas{
		ID:             uuid.New(),
		OrganizationID: organization.ID,
		LiveVersionID:  &versionID,
		Name:           "Release automation",
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	version := CanvasVersion{
		ID:         versionID,
		WorkflowID: canvas.ID,
		OwnerID:    &userID,
		Nodes:      datatypes.NewJSONSlice([]Node{}),
		Edges:      datatypes.NewJSONSlice([]Edge{}),
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	node := CanvasNode{
		WorkflowID:    canvas.ID,
		NodeID:        "create-order",
		Name:          "Create work order",
		State:         CanvasNodeStateReady,
		Type:          NodeTypeComponent,
		Ref:           datatypes.NewJSONType(NodeRef{Component: &ComponentRef{Name: "noop"}}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(Position{}),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	run := CanvasRun{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     node.NodeID,
		VersionID:  versionID,
		State:      CanvasRunStateFinished,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&canvas).Error; err != nil {
			return err
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		return tx.Create(&run).Error
	}))

	firstOrderID := uuid.New()
	secondOrderID := uuid.New()
	refs, err := ResolveFactoryWorkOrderCreatorAutomations(database.Conn(), []FactoryWorkOrder{
		{ID: firstOrderID, SourceRunID: &run.ID},
		{ID: secondOrderID, SourceRunID: &run.ID},
		{ID: uuid.New()},
	})
	require.NoError(t, err)
	require.Len(t, refs, 2)
	assert.Equal(t, &factory.AutomationRef{
		AppID:    canvas.ID,
		AppName:  canvas.Name,
		NodeID:   node.NodeID,
		NodeName: node.Name,
	}, refs[firstOrderID])
	assert.Same(t, refs[firstOrderID], refs[secondOrderID])

	require.NoError(t, canvas.SoftDeleteInTransaction(database.Conn()))
	refs, err = ResolveFactoryWorkOrderCreatorAutomations(database.Conn(), []FactoryWorkOrder{
		{ID: firstOrderID, SourceRunID: &run.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, node.Name, refs[firstOrderID].NodeName)
}

func TestFactoryWorkOrder_UpdateStatusTransitions(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "transitions")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Lifecycle", "", &userID, nil, nil)
	require.NoError(t, err)

	t.Run("draft to open emits status.updated", func(t *testing.T) {
		changed, err := order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateOpen,
			Actor:   &userID,
		})
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, FactoryWorkOrderStateOpen, order.State)

		events, err := order.ListEvents(database.Conn(), 10, nil)
		require.NoError(t, err)
		latest := findEventOfType(t, events, factory.EventTypeOrderStatusUpdated)
		var payload factory.WorkOrderStatusUpdated
		require.NoError(t, json.Unmarshal(latest.Data, &payload))
		assert.Equal(t, FactoryWorkOrderStateDraft, payload.FromState)
		assert.Equal(t, FactoryWorkOrderStateOpen, payload.ToState)
	})

	t.Run("open to closed requires a valid close result", func(t *testing.T) {
		changed, err := order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateClosed,
			Actor:   &userID,
		})
		require.Error(t, err)
		assert.False(t, changed)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderInvalidState)
	})

	t.Run("open to closed with failed result emits status.updated", func(t *testing.T) {
		changed, err := order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateClosed,
			Result:  FactoryWorkOrderResultFailed,
			Actor:   &userID,
		})
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, FactoryWorkOrderStateClosed, order.State)
		assert.Equal(t, FactoryWorkOrderResultFailed, order.Result)

		events, err := order.ListEvents(database.Conn(), 10, nil)
		require.NoError(t, err)
		latest := findEventOfType(t, events, factory.EventTypeOrderStatusUpdated)
		var payload factory.WorkOrderStatusUpdated
		require.NoError(t, json.Unmarshal(latest.Data, &payload))
		assert.Equal(t, FactoryWorkOrderStateOpen, payload.FromState)
		assert.Equal(t, FactoryWorkOrderStateClosed, payload.ToState)
		assert.Equal(t, FactoryWorkOrderResultFailed, payload.ToResult)
	})

	t.Run("draft to closed is allowed only with rejected", func(t *testing.T) {
		other, err := factoryModel.CreateWorkOrder(database.Conn(), "Direct close", "", &userID, nil, nil)
		require.NoError(t, err)

		// `completed` and `failed` imply the order actually ran; only `rejected`
		// (abandon-before-dispatch) makes sense for a never-opened draft.
		for _, invalid := range []string{FactoryWorkOrderResultCompleted, FactoryWorkOrderResultFailed} {
			_, err = other.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
				ToState: FactoryWorkOrderStateClosed,
				Result:  invalid,
				Actor:   &userID,
			})
			require.Error(t, err, "expected draft→closed with result=%q to be rejected", invalid)
			assert.ErrorIs(t, err, ErrFactoryWorkOrderInvalidState)
		}

		_, err = other.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateClosed,
			Result:  FactoryWorkOrderResultRejected,
			Actor:   &userID,
		})
		require.NoError(t, err)
		assert.Equal(t, FactoryWorkOrderStateClosed, other.State)
		assert.Equal(t, FactoryWorkOrderResultRejected, other.Result)

		events, err := other.ListEvents(database.Conn(), 10, nil)
		require.NoError(t, err)
		// Should have exactly two status.updated events: creation ("" → draft)
		// and abandonment (draft → closed). No coarse order.opened/closed.
		statusEvents := filterEventsOfType(events, factory.EventTypeOrderStatusUpdated)
		require.Len(t, statusEvents, 2)
		var latest factory.WorkOrderStatusUpdated
		require.NoError(t, json.Unmarshal(statusEvents[0].Data, &latest))
		assert.Equal(t, FactoryWorkOrderStateDraft, latest.FromState)
		assert.Equal(t, FactoryWorkOrderStateClosed, latest.ToState)
		assert.Equal(t, FactoryWorkOrderResultRejected, latest.ToResult)
	})

	t.Run("reopen from closed emits a fresh status.updated", func(t *testing.T) {
		reopened, err := factoryModel.CreateWorkOrder(database.Conn(), "Reopen", "", &userID, nil, nil)
		require.NoError(t, err)

		for _, step := range []FactoryWorkOrderStatusUpdate{
			{ToState: FactoryWorkOrderStateOpen, Actor: &userID},
			{ToState: FactoryWorkOrderStateClosed, Result: FactoryWorkOrderResultCompleted, Actor: &userID},
		} {
			_, err := reopened.UpdateStatus(database.Conn(), step)
			require.NoError(t, err)
		}

		before, err := reopened.ListEvents(database.Conn(), 50, nil)
		require.NoError(t, err)
		statusBefore := countEventsOfType(before, factory.EventTypeOrderStatusUpdated)

		_, err = reopened.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateOpen,
			Actor:   &userID,
		})
		require.NoError(t, err)

		after, err := reopened.ListEvents(database.Conn(), 50, nil)
		require.NoError(t, err)
		assert.Equal(t, statusBefore+1, countEventsOfType(after, factory.EventTypeOrderStatusUpdated),
			"reopen must record exactly one new order.status.updated event")

		latest := findEventOfType(t, after, factory.EventTypeOrderStatusUpdated)
		var payload factory.WorkOrderStatusUpdated
		require.NoError(t, json.Unmarshal(latest.Data, &payload))
		assert.Equal(t, FactoryWorkOrderStateClosed, payload.FromState)
		assert.Equal(t, FactoryWorkOrderStateOpen, payload.ToState)
	})
}

func TestFactoryWorkOrder_UpdateStatusForwardsAutomation(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "automation-forward")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Automation", "", &userID, nil, nil)
	require.NoError(t, err)

	stepIndex := 0
	automation := &factory.AutomationRef{
		NodeID:    "node-comment",
		NodeName:  "node-comment",
		AppName:   "Factory-App",
		LineName:  "Plan",
		StepIndex: &stepIndex,
		StepName:  "step-01",
	}

	_, err = order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
		ToState:    FactoryWorkOrderStateOpen,
		Automation: automation,
	})
	require.NoError(t, err)
	_, err = order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
		ToState:    FactoryWorkOrderStateClosed,
		Result:     FactoryWorkOrderResultCompleted,
		Automation: automation,
	})
	require.NoError(t, err)

	events, err := order.ListEvents(database.Conn(), 50, nil)
	require.NoError(t, err)

	// Every transition (creation, open, close) is a status.updated event
	// carrying the automation ref. Assert on the two non-creation ones.
	statusEvents := filterEventsOfType(events, factory.EventTypeOrderStatusUpdated)
	require.GreaterOrEqual(t, len(statusEvents), 2)
	for _, ev := range statusEvents[:2] {
		var payload factory.WorkOrderStatusUpdated
		require.NoError(t, json.Unmarshal(ev.Data, &payload))
		require.NotNil(t, payload.Automation, "%s → %s missing automation", payload.FromState, payload.ToState)
		assert.Equal(t, "Plan", payload.Automation.LineName)
		assert.Equal(t, "step-01", payload.Automation.StepName)
		assert.Equal(t, "node-comment", payload.Automation.NodeName)
	}
}

func TestFactoryWorkOrder_RecordCommentAdded(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "comment")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Comment target", "", &userID, nil, nil)
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

func TestFactoryWorkOrder_ListComments(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "list-comments")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "List comments target", "", &userID, nil, nil)
	require.NoError(t, err)

	t.Run("no comments returns empty slice, not nil", func(t *testing.T) {
		comments, err := order.ListComments(database.Conn())
		require.NoError(t, err)
		assert.NotNil(t, comments)
		assert.Empty(t, comments)
	})

	userIDStr := userID.String()
	require.NoError(t, order.RecordCommentAdded(database.Conn(), "First comment", factory.WorkOrderCommentAuthor{
		Kind:   factory.CommentAuthorKindUser,
		UserID: &userIDStr,
	}, nil))

	require.NoError(t, order.RecordCommentAdded(database.Conn(), "Second comment", factory.WorkOrderCommentAuthor{
		Kind: factory.CommentAuthorKindAutomation,
		Automation: &factory.AutomationRef{
			NodeID:   "node-1",
			NodeName: "Node One",
		},
	}, &factory.RunRef{ID: uuid.New(), State: "finished"}))

	// A status update event should never show up in the comment thread.
	require.NoError(t, order.RecordStatusUpdated(database.Conn(), statusUpdatedRecord{
		FromState: FactoryWorkOrderStateDraft,
		ToState:   FactoryWorkOrderStateOpen,
	}))

	comments, err := order.ListComments(database.Conn())
	require.NoError(t, err)
	require.Len(t, comments, 2)

	for _, e := range comments {
		assert.Equal(t, factory.EventTypeOrderCommentAdded, e.Type)
	}

	var first, second factory.WorkOrderCommentAdded
	require.NoError(t, json.Unmarshal(comments[0].Data, &first))
	require.NoError(t, json.Unmarshal(comments[1].Data, &second))

	// Oldest first — chronological reading order.
	assert.Equal(t, "First comment", first.Body)
	require.NotNil(t, first.Author)
	assert.Equal(t, factory.CommentAuthorKindUser, first.Author.Kind)

	assert.Equal(t, "Second comment", second.Body)
	require.NotNil(t, second.Author)
	assert.Equal(t, factory.CommentAuthorKindAutomation, second.Author.Kind)
	require.NotNil(t, second.Author.Automation)
	assert.Equal(t, "node-1", second.Author.Automation.NodeID)
	require.NotNil(t, second.Run)
}

func TestFactoryWorkOrder_CreateArtifact(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "artifact")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Artifact target", "", &userID, nil, nil)
	require.NoError(t, err)

	t.Run("pr requires data.url", func(t *testing.T) {
		_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypePR,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)

		_, err = order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypePR,
			Data: map[string]any{"url": "   "},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
	})

	t.Run("pr rejects non-http(s) data.url", func(t *testing.T) {
		cases := []string{
			"javascript:alert(1)",
			"data:text/html,<script>alert(1)</script>",
			"file:///etc/passwd",
			"mailto:someone@example.com",
			"//evil.example/pull/1",
			"not a url at all",
		}
		for _, prURL := range cases {
			t.Run(prURL, func(t *testing.T) {
				_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
					Type: FactoryWorkOrderArtifactTypePR,
					Data: map[string]any{"url": prURL},
				})
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
			})
		}
	})

	t.Run("markdown requires data.body", func(t *testing.T) {
		_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeMarkdown,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)

		_, err = order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeMarkdown,
			Data: map[string]any{"body": "   "},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
	})

	t.Run("rejects artifact data over 64KiB", func(t *testing.T) {
		body := strings.Repeat("a", MaxFactoryWorkOrderArtifactDataBytes)
		_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeMarkdown,
			Data: map[string]any{"body": body},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
		assert.Contains(t, err.Error(), "exceeds")
	})

	t.Run("branch requires data.name", func(t *testing.T) {
		_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeBranch,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)

		_, err = order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeBranch,
			Data: map[string]any{"name": "   "},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
	})

	t.Run("markdown rejects non-http(s) data.url", func(t *testing.T) {
		cases := []string{
			"javascript:alert(1)",
			"data:text/html,<script>alert(1)</script>",
			"file:///etc/passwd",
			"mailto:someone@example.com",
			"//evil.example/notes",
		}
		for _, badURL := range cases {
			t.Run(badURL, func(t *testing.T) {
				_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
					Type: FactoryWorkOrderArtifactTypeMarkdown,
					Data: map[string]any{
						"body": "note body",
						"url":  badURL,
					},
				})
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
			})
		}
	})

	t.Run("creates pr and emits event", func(t *testing.T) {
		artifact, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypePR,
			Data: map[string]any{
				"url":    "https://github.com/example/repo/pull/1",
				"title":  "Draft PR",
				"number": "1",
			},
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

	t.Run("markdown persists free-form data", func(t *testing.T) {
		artifact, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeMarkdown,
			Data: map[string]any{
				"title": "Design notes",
				"body":  "Investigation notes: retry policy exceeded idempotency window.",
			},
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
		assert.Equal(t, "Design notes", payload.Artifact.Data["title"])
	})

	t.Run("creates branch and emits event", func(t *testing.T) {
		artifact, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeBranch,
			Data: map[string]any{
				"name": "feature/refund-retry",
			},
			CreatedBy: &userID,
		})
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, FactoryWorkOrderArtifactTypeBranch, artifact.Type)

		events, err := order.ListEvents(database.Conn(), 10, nil)
		require.NoError(t, err)

		artifactEvent := findEventOfType(t, events, factory.EventTypeOrderArtifactAdded)
		var payload factory.WorkOrderArtifactAdded
		require.NoError(t, json.Unmarshal(artifactEvent.Data, &payload))
		require.NotNil(t, payload.Artifact)
		assert.Equal(t, "feature/refund-retry", payload.Artifact.Data["name"])
	})

	// A markdown artifact with no data.url must succeed — the http(s)
	// guard only kicks in when url is actually present. Runs last so
	// earlier subtests that count artifacts stay accurate.
	t.Run("markdown accepts absent data.url", func(t *testing.T) {
		artifact, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeMarkdown,
			Data: map[string]any{
				"body": "note without a url",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, artifact)
	})
}

func TestFactoryWorkOrder_CreateArtifact_Key(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "artifact-key")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Key target", "", &userID, nil, nil)
	require.NoError(t, err)

	t.Run("creates artifact with a key", func(t *testing.T) {
		artifact, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypePR,
			Data: map[string]any{"url": "https://github.com/example/repo/pull/1"},
			Key:  "https://github.com/example/repo/pull/1",
		})
		require.NoError(t, err)
		require.NotNil(t, artifact.Key)
		assert.Equal(t, "https://github.com/example/repo/pull/1", *artifact.Key)
	})

	t.Run("rejects a duplicate key in the same factory", func(t *testing.T) {
		_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypePR,
			Data: map[string]any{"url": "https://github.com/example/repo/pull/2"},
			Key:  "https://github.com/example/repo/pull/1",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactKeyAlreadyExists)
	})

	t.Run("allows the same key across two different factories", func(t *testing.T) {
		_, otherUserID, otherFactory := setupFactoryWithUser(t, "artifact-key-other")

		otherOrder, err := otherFactory.CreateWorkOrder(database.Conn(), "Other factory target", "", &otherUserID, nil, nil)
		require.NoError(t, err)

		_, err = otherOrder.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypePR,
			Data: map[string]any{"url": "https://github.com/example/repo/pull/1"},
			Key:  "https://github.com/example/repo/pull/1",
		})
		require.NoError(t, err)
	})

	t.Run("trims the key and treats blank as no key", func(t *testing.T) {
		artifact, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeBranch,
			Data: map[string]any{"name": "feature/trimmed-key"},
			Key:  "  padded-key  ",
		})
		require.NoError(t, err)
		require.NotNil(t, artifact.Key)
		assert.Equal(t, "padded-key", *artifact.Key)

		blank, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeBranch,
			Data: map[string]any{"name": "feature/blank-key-1"},
			Key:  "   ",
		})
		require.NoError(t, err)
		assert.Nil(t, blank.Key)

		// A second artifact with an empty key must not collide with the
		// first — the partial unique index excludes NULLs.
		blankTwo, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeBranch,
			Data: map[string]any{"name": "feature/blank-key-2"},
		})
		require.NoError(t, err)
		assert.Nil(t, blankTwo.Key)
	})

	t.Run("rejects a key over 512 bytes", func(t *testing.T) {
		_, err := order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
			Type: FactoryWorkOrderArtifactTypeBranch,
			Data: map[string]any{"name": "feature/long-key"},
			Key:  strings.Repeat("a", MaxFactoryWorkOrderArtifactKeyBytes+1),
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderArtifactInvalid)
	})
}

func TestFactory_FindWorkOrderByArtifactKey(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "find-by-key")
	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Find target", "", &userID, nil, nil)
	require.NoError(t, err)

	_, err = order.CreateArtifact(database.Conn(), FactoryWorkOrderArtifactParams{
		Type: FactoryWorkOrderArtifactTypePR,
		Data: map[string]any{"url": "https://github.com/example/repo/pull/42"},
		Key:  "https://github.com/example/repo/pull/42",
	})
	require.NoError(t, err)

	t.Run("finds the work order by its artifact key", func(t *testing.T) {
		found, err := factoryModel.FindWorkOrderByArtifactKey(database.Conn(), "https://github.com/example/repo/pull/42")
		require.NoError(t, err)
		assert.Equal(t, order.ID, found.ID)
	})

	t.Run("trims the lookup key", func(t *testing.T) {
		found, err := factoryModel.FindWorkOrderByArtifactKey(database.Conn(), "  https://github.com/example/repo/pull/42  ")
		require.NoError(t, err)
		assert.Equal(t, order.ID, found.ID)
	})

	t.Run("returns not-found for an unknown key", func(t *testing.T) {
		_, err := factoryModel.FindWorkOrderByArtifactKey(database.Conn(), "no-such-key")
		assert.ErrorIs(t, err, ErrFactoryWorkOrderNotFound)
	})

	t.Run("returns not-found for a blank key", func(t *testing.T) {
		_, err := factoryModel.FindWorkOrderByArtifactKey(database.Conn(), "   ")
		assert.ErrorIs(t, err, ErrFactoryWorkOrderNotFound)
	})

	t.Run("does not find a key belonging to a different factory", func(t *testing.T) {
		_, _, otherFactory := setupFactoryWithUser(t, "find-by-key-other")

		_, err := otherFactory.FindWorkOrderByArtifactKey(database.Conn(), "https://github.com/example/repo/pull/42")
		assert.ErrorIs(t, err, ErrFactoryWorkOrderNotFound)
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

func filterEventsOfType(events []FactoryWorkOrderEvent, eventType string) []FactoryWorkOrderEvent {
	result := make([]FactoryWorkOrderEvent, 0)
	for _, e := range events {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result
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
