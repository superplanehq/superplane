package models

import (
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
		Headline: "Review the pull request",
		Body:     "Merging PR #42 completes this work order.",
		CtaLabel: "Review PR #42",
		CtaURL:   "https://github.com/acme/app/pull/42",
	}

	cases := []struct {
		name   string
		mutate func(params *FactoryWorkOrderStatusNoteParams)
	}{
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

	_, err = order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{Headline: "Waiting"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFactoryWorkOrderStatusNoteInvalid)
}

func TestFactoryWorkOrder_SetStatusNote_StoresAndReplaces(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, _ := openWorkOrderForStatusNote(t, "note-store")

	appID := uuid.New()
	note, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
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
	assert.Equal(t, FactoryWorkOrderStatusNoteKindInfo, note.Kind)
	assert.False(t, note.UpdatedAt.IsZero())

	reloaded, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	stored, err := reloaded.StatusNoteRef()
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "Review the pull request", stored.Headline)
	assert.Equal(t, "Review PR #42", stored.CtaLabel)
	require.NotNil(t, stored.Automation)
	assert.Equal(t, appID, stored.Automation.AppID)

	// Latest wins: a second set replaces the note wholesale.
	_, err = reloaded.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Headline: "Waiting on CI",
	})
	require.NoError(t, err)

	replaced, err := reloaded.StatusNoteRef()
	require.NoError(t, err)
	require.NotNil(t, replaced)
	assert.Equal(t, "Waiting on CI", replaced.Headline)
	assert.Empty(t, replaced.CtaLabel)
	assert.Nil(t, replaced.Automation)
}

func TestFactoryWorkOrder_StatusNote_ClearedOnTransition(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, userID := openWorkOrderForStatusNote(t, "note-transition")

	_, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
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

	note, err := reloaded.StatusNoteRef()
	require.NoError(t, err)
	assert.Nil(t, note)
}

func TestFactoryWorkOrder_ClearStatusNote(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	order, _ := openWorkOrderForStatusNote(t, "note-clear")

	_, err := order.SetStatusNote(database.Conn(), FactoryWorkOrderStatusNoteParams{
		Headline: "Review the pull request",
	})
	require.NoError(t, err)

	require.NoError(t, order.ClearStatusNote(database.Conn()))

	reloaded, err := FindUnscopedWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)

	note, err := reloaded.StatusNoteRef()
	require.NoError(t, err)
	assert.Nil(t, note)
}
