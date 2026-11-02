package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
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
		{"boolean with fractional score", func(p *FactoryWorkOrderCheckParams) {
			p.Format = FactoryWorkOrderCheckFormatBoolean
			p.Score = 0.5
			p.MaxScore = 1
		}},
		{"boolean with max score above one", func(p *FactoryWorkOrderCheckParams) {
			p.Format = FactoryWorkOrderCheckFormatBoolean
			p.Score = 1
			p.MaxScore = 2
		}},
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
	assert.Equal(t, "Low risk overall.", payload.Check.Summary)
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

func TestFactoryWorkOrder_ReportCheck_TracksRecentScores(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "check-recent-scores")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Check target", "", &userID, nil, nil)
	require.NoError(t, err)

	report := func(score float64) *FactoryWorkOrderCheck {
		check, reportErr := order.ReportCheck(database.Conn(), FactoryWorkOrderCheckParams{
			Key:      "ci",
			Name:     "CI",
			Score:    score,
			MaxScore: 1,
			Format:   FactoryWorkOrderCheckFormatBoolean,
			Level:    FactoryWorkOrderCheckLevelPositive,
		})
		require.NoError(t, reportErr)
		return check
	}

	first := report(0)
	assert.Equal(t, []float64{0}, []float64(first.RecentScores))

	second := report(0)
	assert.Equal(t, []float64{0, 0}, []float64(second.RecentScores))

	third := report(1)
	assert.Equal(t, []float64{0, 0, 1}, []float64(third.RecentScores))
	assert.Equal(t, FactoryWorkOrderCheckFormatBoolean, third.Format)
}

func TestAppendRecentCheckScore_CapsAndSeedsHistory(t *testing.T) {
	// A row created before the history column existed seeds the history
	// with the score it held, so the strip does not start mid-stream.
	seeded := appendRecentCheckScore(nil, 7, 3)
	assert.Equal(t, []float64{7, 3}, []float64(seeded))

	long := make([]float64, MaxFactoryWorkOrderCheckRecentScores)
	for i := range long {
		long[i] = float64(i)
	}
	capped := appendRecentCheckScore(datatypes.NewJSONSlice(long), 0, 99)
	require.Len(t, []float64(capped), MaxFactoryWorkOrderCheckRecentScores)
	assert.Equal(t, float64(1), capped[0])
	assert.Equal(t, float64(99), capped[MaxFactoryWorkOrderCheckRecentScores-1])
}

// The retry in ReportCheck keys off this detection: a raced first report
// must be recognized by the unique index name, nothing else.
func TestIsFactoryWorkOrderCheckKeyConflict(t *testing.T) {
	conflict := &pgconn.PgError{ConstraintName: factoryWorkOrderCheckKeyUniqueConstraint}

	assert.True(t, isFactoryWorkOrderCheckKeyConflict(conflict))
	assert.True(t, isFactoryWorkOrderCheckKeyConflict(fmt.Errorf("insert: %w", conflict)))
	assert.False(t, isFactoryWorkOrderCheckKeyConflict(nil))
	assert.False(t, isFactoryWorkOrderCheckKeyConflict(errors.New("boom")))
	assert.False(t, isFactoryWorkOrderCheckKeyConflict(&pgconn.PgError{ConstraintName: "other_constraint"}))
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

	grouped, err := ListChecksForWorkOrders(database.Conn(), []uuid.UUID{order.ID})
	require.NoError(t, err)
	require.Len(t, grouped[order.ID], 3)
	assert.Equal(t, checks[0].Key, grouped[order.ID][0].Key)
	assert.Equal(t, "risk-review", checks[0].Key)
	assert.Equal(t, "code-coverage", checks[1].Key)
	assert.Equal(t, "confidence", checks[2].Key)
}

func TestFactoryWorkOrder_ReportAgentChecks(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())

	_, userID, factoryModel := setupFactoryWithUser(t, "agent-checks")
	order, err := factoryModel.CreateWorkOrder(db, "Check target", "", &userID, nil, nil)
	require.NoError(t, err)

	agentCheck := func(key string, score float64) FactoryWorkOrderCheckParams {
		return FactoryWorkOrderCheckParams{Key: key, Name: key, Score: score, MaxScore: 5, Level: FactoryWorkOrderCheckLevelCaution, Summary: "Reason."}
	}

	t.Run("saves agent checks and replaces an agent check with the same key", func(t *testing.T) {
		_, err := order.ReportAgentChecks(db, []FactoryWorkOrderCheckParams{agentCheck("api-latency", 2)})
		require.NoError(t, err)
		checks, err := order.ReportAgentChecks(db, []FactoryWorkOrderCheckParams{agentCheck("api-latency", 4)})
		require.NoError(t, err)

		require.Len(t, checks, 1)
		assert.Equal(t, FactoryWorkOrderCheckSourceAgent, checks[0].Source)
		assert.Equal(t, 4.0, checks[0].Score)
		require.NotNil(t, checks[0].PreviousScore)
		assert.Equal(t, 2.0, *checks[0].PreviousScore)
	})

	t.Run("does not replace a check that an agent did not add", func(t *testing.T) {
		_, err := order.ReportCheck(db, agentCheck("coverage", 5))
		require.NoError(t, err)

		_, err = order.ReportAgentChecks(db, []FactoryWorkOrderCheckParams{agentCheck("coverage", 1)})

		assert.ErrorIs(t, err, ErrFactoryWorkOrderCheckOwned)
		checks, err := order.ListChecks(db)
		require.NoError(t, err)
		coverage := findCheck(t, checks, "coverage")
		assert.Equal(t, 5.0, coverage.Score)
		assert.Equal(t, FactoryWorkOrderCheckSourceAutomation, coverage.Source)
	})

	t.Run("rejects plan score and merge confidence keys before the first report", func(t *testing.T) {
		for _, key := range []string{PlanningClarityCheckKey, PlanningConfidenceCheckKey, "risk-review", "custom-review"} {
			_, err := order.ReportAgentChecks(db, []FactoryWorkOrderCheckParams{agentCheck(key, 5)})
			assert.ErrorIs(t, err, ErrFactoryWorkOrderCheckOwned, key)
		}
	})

	t.Run("rejects plan score names under another key", func(t *testing.T) {
		for _, name := range []string{PlanningClarityCheckName, " confidence SCORE "} {
			check := agentCheck("agent-score", 5)
			check.Name = name
			_, err := order.ReportAgentChecks(db, []FactoryWorkOrderCheckParams{check})
			assert.ErrorIs(t, err, ErrFactoryWorkOrderCheckInvalid, name)
		}
	})

	t.Run("saves no check when one check fails", func(t *testing.T) {
		_, err := order.ReportAgentChecks(db, []FactoryWorkOrderCheckParams{
			agentCheck("first-new", 3),
			agentCheck(PlanningClarityCheckKey, 3),
		})
		require.ErrorIs(t, err, ErrFactoryWorkOrderCheckOwned)

		checks, err := order.ListChecks(db)
		require.NoError(t, err)
		for _, check := range checks {
			assert.NotEqual(t, "first-new", check.Key)
		}
	})

	t.Run("a plan or merge confidence report replaces an agent check", func(t *testing.T) {
		_, err := order.ReportAgentChecks(db, []FactoryWorkOrderCheckParams{agentCheck("handover", 1)})
		require.NoError(t, err)

		check, err := order.ReportCheck(db, agentCheck("handover", 3))

		require.NoError(t, err)
		assert.Equal(t, FactoryWorkOrderCheckSourceAutomation, check.Source)
	})
}

func TestFactoryWorkOrder_RemoveAgentChecks(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())

	_, userID, factoryModel := setupFactoryWithUser(t, "remove-agent-checks")
	order, err := factoryModel.CreateWorkOrder(db, "Check target", "", &userID, nil, nil)
	require.NoError(t, err)
	params := func(key string) FactoryWorkOrderCheckParams {
		return FactoryWorkOrderCheckParams{Key: key, Name: key, Score: 1, MaxScore: 5}
	}
	_, err = order.ReportAgentChecks(db, []FactoryWorkOrderCheckParams{params("api-latency")})
	require.NoError(t, err)
	_, err = order.ReportCheck(db, params("coverage"))
	require.NoError(t, err)

	removed, err := order.RemoveAgentChecks(db, []string{"api-latency", "coverage", "missing"})

	require.NoError(t, err)
	assert.Equal(t, []string{"api-latency"}, removed)
	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, "coverage", checks[0].Key)
}

func findCheck(t *testing.T, checks []FactoryWorkOrderCheck, key string) FactoryWorkOrderCheck {
	t.Helper()
	for _, check := range checks {
		if check.Key == key {
			return check
		}
	}
	t.Fatalf("check %q not found", key)
	return FactoryWorkOrderCheck{}
}
