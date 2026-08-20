package posthog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__BuildQuery(t *testing.T) {
	t.Run("selected events are passed as values, never written into the query", func(t *testing.T) {
		spec := &RunQuerySpec{Events: []string{"signed up", "checkout completed"}}

		query, values, err := BuildQuery(spec)
		require.NoError(t, err)

		assert.Contains(t, query, "event IN {events}")
		assert.NotContains(t, query, "signed up")
		assert.Equal(t, []string{"signed up", "checkout completed"}, values["events"])
	})

	t.Run("an event name that looks like SQL stays out of the query text", func(t *testing.T) {
		spec := &RunQuerySpec{Events: []string{"' OR 1=1 --"}}

		query, values, err := BuildQuery(spec)
		require.NoError(t, err)

		assert.NotContains(t, query, "OR 1=1")
		assert.Equal(t, []string{"' OR 1=1 --"}, values["events"])
	})

	t.Run("no event selection means no event condition", func(t *testing.T) {
		query, values, err := BuildQuery(&RunQuerySpec{})
		require.NoError(t, err)

		assert.NotContains(t, query, "event IN")
		assert.NotContains(t, values, "events")
	})

	t.Run("each aggregation produces its own shape", func(t *testing.T) {
		for aggregation, expected := range map[string]string{
			AggregationEvents:       "SELECT event, distinct_id, timestamp, properties FROM events",
			AggregationCountByEvent: "SELECT event, count() AS count FROM events",
			AggregationTotalCount:   "SELECT count() AS count FROM events",
			AggregationUniqueUsers:  "SELECT count(DISTINCT distinct_id) AS uniqueUsers FROM events",
		} {
			query, _, err := BuildQuery(&RunQuerySpec{Aggregation: aggregation})
			require.NoError(t, err, aggregation)
			assert.True(t, strings.HasPrefix(query, expected), "%s: got %s", aggregation, query)
		}
	})

	t.Run("counting aggregations group and order, single-number ones do not", func(t *testing.T) {
		query, _, err := BuildQuery(&RunQuerySpec{Aggregation: AggregationCountByEvent})
		require.NoError(t, err)
		assert.Contains(t, query, "GROUP BY event")
		assert.Contains(t, query, "ORDER BY count DESC")

		query, _, err = BuildQuery(&RunQuerySpec{Aggregation: AggregationTotalCount})
		require.NoError(t, err)
		assert.NotContains(t, query, "GROUP BY")
		assert.NotContains(t, query, "ORDER BY")
	})

	t.Run("a row limit applies only where rows are returned", func(t *testing.T) {
		query, _, err := BuildQuery(&RunQuerySpec{Aggregation: AggregationEvents, Limit: 25})
		require.NoError(t, err)
		assert.Contains(t, query, "LIMIT 25")

		query, _, err = BuildQuery(&RunQuerySpec{Aggregation: AggregationTotalCount, Limit: 25})
		require.NoError(t, err)
		assert.NotContains(t, query, "LIMIT")
	})

	t.Run("limits fall back to the default and are capped", func(t *testing.T) {
		query, _, err := BuildQuery(&RunQuerySpec{})
		require.NoError(t, err)
		assert.Contains(t, query, "LIMIT 100")

		query, _, err = BuildQuery(&RunQuerySpec{Limit: MaxLimit + 5000})
		require.NoError(t, err)
		assert.Contains(t, query, "LIMIT 10000")
	})

	t.Run("each time range maps onto its interval", func(t *testing.T) {
		for timeRange, expected := range map[string]string{
			"1h":  "INTERVAL 1 HOUR",
			"24h": "INTERVAL 1 DAY",
			"7d":  "INTERVAL 7 DAY",
			"30d": "INTERVAL 30 DAY",
			"90d": "INTERVAL 90 DAY",
		} {
			query, _, err := BuildQuery(&RunQuerySpec{TimeRange: timeRange})
			require.NoError(t, err, timeRange)
			assert.Contains(t, query, "timestamp > now() - "+expected)
		}
	})

	//
	// Clock skew on clients leaves events dated in the future. Bounded only at
	// the bottom, a timestamp-ordered result returns those first and nothing
	// that actually just happened.
	//
	t.Run("the window is bounded at both ends, so future-dated events are excluded", func(t *testing.T) {
		for _, aggregation := range []string{
			AggregationEvents,
			AggregationCountByEvent,
			AggregationTotalCount,
			AggregationUniqueUsers,
		} {
			query, _, err := BuildQuery(&RunQuerySpec{Aggregation: aggregation})
			require.NoError(t, err, aggregation)
			assert.Contains(t, query, "timestamp <= now()", aggregation)
		}
	})

	t.Run("the default time range is used when none is set", func(t *testing.T) {
		query, _, err := BuildQuery(&RunQuerySpec{})
		require.NoError(t, err)
		assert.Contains(t, query, "INTERVAL 7 DAY")
	})

	//
	// PostHog resolves a query's placeholders before it substitutes filters, so a
	// query that carries {filters} alongside parameters is rejected outright with
	// "Global variable not found: filters". Parameters are what keep event names
	// out of the query text, so the filters placeholder is the one that goes.
	//
	t.Run("no filters placeholder is emitted, as it cannot coexist with parameters", func(t *testing.T) {
		query, _, err := BuildQuery(&RunQuerySpec{Events: []string{"signed up"}})
		require.NoError(t, err)
		assert.NotContains(t, query, "{filters}")
	})

	t.Run("unknown options are rejected rather than guessed at", func(t *testing.T) {
		_, _, err := BuildQuery(&RunQuerySpec{TimeRange: "all time"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown time range")

		_, _, err = BuildQuery(&RunQuerySpec{Aggregation: "median"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown aggregation")
	})

	t.Run("every time range option offered in the form can be built", func(t *testing.T) {
		for _, timeRange := range TimeRangeOptions() {
			_, _, err := BuildQuery(&RunQuerySpec{TimeRange: timeRange.Value})
			assert.NoError(t, err, timeRange.Value)
		}
	})
}

func Test__RunQuerySpec__resolve(t *testing.T) {
	t.Run("HogQL mode sends the query as written, with no values", func(t *testing.T) {
		spec := &RunQuerySpec{Mode: QueryModeHogQL, Query: "SELECT 1"}

		request, err := spec.resolve()
		require.NoError(t, err)

		assert.Equal(t, "SELECT 1", request.Query)
		assert.Empty(t, request.Values)
	})

	t.Run("builder mode sends the built query with its parameters", func(t *testing.T) {
		spec := &RunQuerySpec{Mode: QueryModeBuilder, Events: []string{"signed up"}}

		request, err := spec.resolve()
		require.NoError(t, err)

		assert.Contains(t, request.Query, "event IN {events}")
		assert.Equal(t, []string{"signed up"}, request.Values["events"])
	})

	t.Run("a spec saved before the builder existed still runs its query", func(t *testing.T) {
		spec := &RunQuerySpec{Query: "SELECT event FROM events"}

		require.Equal(t, QueryModeHogQL, spec.mode())

		request, err := spec.resolve()
		require.NoError(t, err)
		assert.Equal(t, "SELECT event FROM events", request.Query)
	})
}
