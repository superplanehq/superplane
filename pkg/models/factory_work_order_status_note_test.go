package models

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

func openWorkOrderForStatusNote(t *testing.T, suffix string) (*FactoryWorkOrder, uuid.UUID) {
	t.Helper()

	_, userID, factoryModel := setupFactoryWithUser(t, suffix)

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Note target", "", &userID, nil, nil)
	require.NoError(t, err)

	_, err = order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateOpen,
		Actor:   &userID,
	})
	require.NoError(t, err)

	return order, userID
}

func TestFactoryWorkOrder_SetStatusNote_Validation(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, _ := openWorkOrderForStatusNote(t, "note-validation")

	valid := FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Review the pull request",
		Body:     "Merging PR #42 completes this work order.",
		CtaLabel: "Review PR #42",
		CtaURL:   "https://github.com/acme/app/pull/42",
	}

	cases := []struct {
		name   string
		mutate func(params *FactoryWorkOrderStatusNoteParams)
	}{
		{"missing key", func(p *FactoryWorkOrderStatusNoteParams) { p.Key = "  " }},
		{"missing headline", func(p *FactoryWorkOrderStatusNoteParams) { p.Headline = "  " }},
		{"unknown kind", func(p *FactoryWorkOrderStatusNoteParams) { p.Kind = "decision" }},
		{"cta label without url", func(p *FactoryWorkOrderStatusNoteParams) { p.CtaURL = "" }},
		{"cta url without label", func(p *FactoryWorkOrderStatusNoteParams) { p.CtaLabel = "" }},
		{"relative cta url", func(p *FactoryWorkOrderStatusNoteParams) { p.CtaURL = "/pull/42" }},
		{"non-http cta url", func(p *FactoryWorkOrderStatusNoteParams) { p.CtaURL = "ftp://example.com/x" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := valid
			tc.mutate(&params)

			_, err := order.SetStatusNote(database.Conn(), params)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrFactoryWorkOrderStatusNoteInvalid)
		})
	}
}

func TestFactoryWorkOrder_SetStatusNote_RequiresOpenOrder(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "note-draft")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Draft order", "", &userID, nil, nil)
	require.NoError(t, err)

	_, err = order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Waiting",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFactoryWorkOrderStatusNoteInvalid)
}

func TestFactoryWorkOrder_SetStatusNote_UpsertsByKey(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, _ := openWorkOrderForStatusNote(t, "note-store")

	appID := uuid.New()
	note, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Review the pull request",
		Body:     "Merging PR #42 completes this work order.",
		CtaLabel: "Review PR #42",
		CtaURL:   "https://github.com/acme/app/pull/42",
		Automation: &factory.AutomationRef{
			AppID:   appID,
			AppName: "PR Closure",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "pr-closure", note.Key)
	assert.Equal(t, FactoryWorkOrderStatusNoteKindInfo, note.Kind)
	assert.False(t, note.UpdatedAt.IsZero())

	reloaded, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	stored, err := reloaded.StatusNotes()
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, "pr-closure", stored[0].Key)
	assert.Equal(t, "Review the pull request", stored[0].Headline)
	require.NotNil(t, stored[0].Automation)
	assert.Equal(t, appID, stored[0].Automation.AppID)

	_, err = reloaded.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Review PR #43",
	})
	require.NoError(t, err)

	replaced, err := reloaded.StatusNotes()
	require.NoError(t, err)
	require.Len(t, replaced, 1)
	assert.Equal(t, "Review PR #43", replaced[0].Headline)
	assert.Empty(t, replaced[0].CtaLabel)
	assert.Nil(t, replaced[0].Automation)
}

func TestFactoryWorkOrder_SetStatusNote_KeepsDistinctKeys(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, _ := openWorkOrderForStatusNote(t, "note-multi")

	_, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Review the pull request",
	})
	require.NoError(t, err)

	_, err = order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "deploy-window",
		Headline: "Waiting on the deploy window",
	})
	require.NoError(t, err)

	reloaded, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	notes, err := reloaded.StatusNotes()
	require.NoError(t, err)
	require.Len(t, notes, 2)
	assert.Equal(t, "pr-closure", notes[0].Key)
	assert.Equal(t, "deploy-window", notes[1].Key)
}

func TestFactoryWorkOrder_SetStatusNote_RejectsOverCap(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, _ := openWorkOrderForStatusNote(t, "note-cap")

	for i := range MaxFactoryWorkOrderStatusNotes {
		_, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
			Key:      fmt.Sprintf("note-%d", i),
			Headline: "Waiting",
		})
		require.NoError(t, err)
	}

	_, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "overflow",
		Headline: "One more",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFactoryWorkOrderStatusNoteInvalid)
}

func TestFactoryWorkOrder_StatusNotes_ClearedOnTransition(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, userID := openWorkOrderForStatusNote(t, "note-transition")

	_, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Review the pull request",
	})
	require.NoError(t, err)

	_, err = order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateClosed,
		Result:  FactoryWorkOrderResultCompleted,
		Actor:   &userID,
	})
	require.NoError(t, err)

	reloaded, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	notes, err := reloaded.StatusNotes()
	require.NoError(t, err)
	assert.Empty(t, notes)
}

func TestFactoryWorkOrder_ClearStatusNote_RemovesOneKey(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, _ := openWorkOrderForStatusNote(t, "note-clear-one")

	_, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Review the pull request",
	})
	require.NoError(t, err)
	_, err = order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "deploy-window",
		Headline: "Waiting on the deploy window",
	})
	require.NoError(t, err)

	require.NoError(t, order.ClearStatusNote(database.Conn(), "pr-closure"))

	reloaded, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	notes, err := reloaded.StatusNotes()
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "deploy-window", notes[0].Key)
}

func TestFactoryWorkOrder_SetStatusNote_MergesKeysFromStaleSnapshot(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, _ := openWorkOrderForStatusNote(t, "note-stale-merge")

	first, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)
	second, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	_, err = first.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Review the pull request",
	})
	require.NoError(t, err)

	_, err = second.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "deploy-window",
		Headline: "Waiting on the deploy window",
	})
	require.NoError(t, err)

	reloaded, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	notes, err := reloaded.StatusNotes()
	require.NoError(t, err)
	require.Len(t, notes, 2)
	assert.ElementsMatch(t, []string{"pr-closure", "deploy-window"}, []string{notes[0].Key, notes[1].Key})
}

func TestFactoryWorkOrder_SetStatusNote_RejectsAfterCloseOnStaleSnapshot(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, userID := openWorkOrderForStatusNote(t, "note-stale-close")

	stale, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	_, err = order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateClosed,
		Result:  FactoryWorkOrderResultCompleted,
		Actor:   &userID,
	})
	require.NoError(t, err)

	_, err = stale.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Review the pull request",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFactoryWorkOrderStatusNoteInvalid)

	reloaded, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	notes, err := reloaded.StatusNotes()
	require.NoError(t, err)
	assert.Empty(t, notes)
	assert.Equal(t, FactoryWorkOrderStateClosed, reloaded.State)
}

func TestFactoryWorkOrder_SetStatusNote_PersistsShowOnlyWhenWaiting(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, _ := openWorkOrderForStatusNote(t, "note-waiting-only")

	_, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:                 "queue-slot",
		Headline:            "Waiting for a slot",
		ShowOnlyWhenWaiting: true,
	})
	require.NoError(t, err)

	reloaded, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	notes, err := reloaded.StatusNotes()
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.True(t, notes[0].ShowOnlyWhenWaiting)

	_, err = reloaded.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Review the pull request",
	})
	require.NoError(t, err)

	notes, err = reloaded.StatusNotes()
	require.NoError(t, err)
	require.Len(t, notes, 2)
	assert.False(t, notes[1].ShowOnlyWhenWaiting)
}

func TestFactoryWorkOrder_ClearStatusNotes(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, _ := openWorkOrderForStatusNote(t, "note-clear-all")

	_, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Key:      "pr-closure",
		Headline: "Review the pull request",
	})
	require.NoError(t, err)

	require.NoError(t, order.ClearStatusNotes(database.Conn()))

	reloaded, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	notes, err := reloaded.StatusNotes()
	require.NoError(t, err)
	assert.Empty(t, notes)
}
