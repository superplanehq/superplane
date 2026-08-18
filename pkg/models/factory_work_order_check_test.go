package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

func TestFactoryWorkOrder_ReportCheck_Validation(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "check-validation")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Check target", "", &userID, nil, nil)
	require.NoError(t, err)

	valid := FactoryWorkOrderCheckParams{
		Key:      "risk-review",
		Name:     "Risk review",
		Score:    3,
		MaxScore: 10,
	}

	cases := []struct {
		name   string
		mutate func(params *FactoryWorkOrderCheckParams)
	}{
		{"missing key", func(p *FactoryWorkOrderCheckParams) { p.Key = "  " }},
		{"missing name", func(p *FactoryWorkOrderCheckParams) { p.Name = "" }},
		{"negative score", func(p *FactoryWorkOrderCheckParams) { p.Score = -1 }},
		{"zero max score", func(p *FactoryWorkOrderCheckParams) { p.MaxScore = 0 }},
		{"unknown format", func(p *FactoryWorkOrderCheckParams) { p.Format = "ratio" }},
		{"unknown level", func(p *FactoryWorkOrderCheckParams) { p.Level = "bad" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := valid
			tc.mutate(&params)

			_, err := order.ReportCheck(database.Conn(), params)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrFactoryWorkOrderCheckInvalid)
		})
	}
}

func TestFactoryWorkOrder_ReportCheck_CreatesRowAndEvent(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "check-create")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Check target", "", &userID, nil, nil)
	require.NoError(t, err)

	runID := uuid.New()
	check, err := order.ReportCheck(database.Conn(), FactoryWorkOrderCheckParams{
		Key:      "risk-review",
		Name:     "Risk review",
		Score:    3,
		MaxScore: 10,
		Level:    FactoryWorkOrderCheckLevelPositive,
		Summary:  "Low risk overall.",
		Analysis: "### Findings\n\nNothing alarming.",
		Automation: &factory.AutomationRef{
			NodeID:  "node-1",
			AppID:   uuid.New(),
			AppName: "Risk review app",
		},
		Run: &factory.RunRef{ID: runID},
	})
	require.NoError(t, err)
	assert.Equal(t, FactoryWorkOrderCheckFormatFraction, check.Format)
	assert.Nil(t, check.PreviousScore)
	require.NotNil(t, check.RunID)
	assert.Equal(t, runID, *check.RunID)

	automation, err := check.AutomationRef()
	require.NoError(t, err)
	require.NotNil(t, automation)
	assert.Equal(t, "Risk review app", automation.AppName)

	checks, err := order.ListChecks(database.Conn())
	require.NoError(t, err)
	require.Len(t, checks, 1)

	events, err := order.ListEvents(database.Conn(), 10, nil)
	require.NoError(t, err)
	event := findEventOfType(t, events, factory.EventTypeOrderCheckReported)

	var payload factory.WorkOrderCheckReported
	require.NoError(t, json.Unmarshal(event.Data, &payload))
	require.NotNil(t, payload.Check)
	assert.Equal(t, "risk-review", payload.Check.Key)
	assert.Equal(t, float64(3), payload.Check.Score)
	assert.Equal(t, FactoryWorkOrderCheckLevelPositive, payload.Check.Level)
	assert.Nil(t, payload.Check.PreviousScore)
	require.NotNil(t, payload.Run)
	assert.Equal(t, runID, payload.Run.ID)
}

func TestFactoryWorkOrder_ReportCheck_ReReportUpdatesInPlace(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "check-rereport")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Check target", "", &userID, nil, nil)
	require.NoError(t, err)

	first, err := order.ReportCheck(database.Conn(), FactoryWorkOrderCheckParams{
		Key:      "risk-review",
		Name:     "Risk review",
		Score:    7,
		MaxScore: 10,
		Level:    FactoryWorkOrderCheckLevelCaution,
	})
	require.NoError(t, err)

	second, err := order.ReportCheck(database.Conn(), FactoryWorkOrderCheckParams{
		Key:      "risk-review",
		Name:     "Risk review",
		Score:    3,
		MaxScore: 10,
		Level:    FactoryWorkOrderCheckLevelPositive,
	})
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	require.NotNil(t, second.PreviousScore)
	assert.Equal(t, float64(7), *second.PreviousScore)
	assert.Equal(t, float64(3), second.Score)
	assert.Equal(t, FactoryWorkOrderCheckLevelPositive, second.Level)

	checks, err := order.ListChecks(database.Conn())
	require.NoError(t, err)
	require.Len(t, checks, 1)

	events, err := order.ListEvents(database.Conn(), 10, nil)
	require.NoError(t, err)

	var reported []factory.WorkOrderCheckReported
	for _, event := range events {
		if event.Type != factory.EventTypeOrderCheckReported {
			continue
		}
		var payload factory.WorkOrderCheckReported
		require.NoError(t, json.Unmarshal(event.Data, &payload))
		reported = append(reported, payload)
	}
	require.Len(t, reported, 2)

	// ListEvents returns newest first.
	require.NotNil(t, reported[0].Check.PreviousScore)
	assert.Equal(t, float64(7), *reported[0].Check.PreviousScore)
	assert.Nil(t, reported[1].Check.PreviousScore)
}

func TestFactoryWorkOrder_ListChecks_OrdersByFirstReport(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "check-list")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Check target", "", &userID, nil, nil)
	require.NoError(t, err)

	for _, key := range []string{"risk-review", "code-coverage", "confidence"} {
		_, err := order.ReportCheck(database.Conn(), FactoryWorkOrderCheckParams{
			Key:      key,
			Name:     key,
			Score:    1,
			MaxScore: 10,
		})
		require.NoError(t, err)
	}

	// Re-reporting the first check must not move it to the end.
	_, err = order.ReportCheck(database.Conn(), FactoryWorkOrderCheckParams{
		Key:      "risk-review",
		Name:     "risk-review",
		Score:    2,
		MaxScore: 10,
	})
	require.NoError(t, err)

	checks, err := order.ListChecks(database.Conn())
	require.NoError(t, err)
	require.Len(t, checks, 3)
	assert.Equal(t, "risk-review", checks[0].Key)
	assert.Equal(t, "code-coverage", checks[1].Key)
	assert.Equal(t, "confidence", checks[2].Key)
}
