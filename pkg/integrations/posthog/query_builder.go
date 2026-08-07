package posthog

import (
	"fmt"
	"strings"
)

const (
	QueryModeBuilder = "builder"
	QueryModeHogQL   = "hogql"

	AggregationEvents       = "events"
	AggregationCountByEvent = "countByEvent"
	AggregationTotalCount   = "totalCount"
	AggregationUniqueUsers  = "uniqueUsers"

	DefaultTimeRange = "7d"
	DefaultLimit     = 100
	MaxLimit         = 10000
)

// timeRangeIntervals maps the closed set of time range options onto the HogQL
// interval each one stands for. The interval is looked up here instead of being
// interpolated, so nothing the user typed ever reaches the query text.
var timeRangeIntervals = map[string]string{
	"1h":  "INTERVAL 1 HOUR",
	"24h": "INTERVAL 1 DAY",
	"7d":  "INTERVAL 7 DAY",
	"30d": "INTERVAL 30 DAY",
	"90d": "INTERVAL 90 DAY",
}

// aggregationSelects maps each aggregation onto the part of the query that
// varies: what to select, how to group, and whether a row limit applies.
type aggregationSelect struct {
	columns  string
	groupBy  string
	orderBy  string
	rowBound bool
}

var aggregationSelects = map[string]aggregationSelect{
	AggregationEvents: {
		columns:  "event, distinct_id, timestamp, properties",
		orderBy:  "timestamp DESC",
		rowBound: true,
	},
	AggregationCountByEvent: {
		columns:  "event, count() AS count",
		groupBy:  "event",
		orderBy:  "count DESC",
		rowBound: true,
	},
	AggregationTotalCount: {
		columns: "count() AS count",
	},
	AggregationUniqueUsers: {
		columns: "count(DISTINCT distinct_id) AS uniqueUsers",
	},
}

// TimeRangeOptions returns the supported time ranges with the labels shown in
// the component form.
func TimeRangeOptions() []struct{ Value, Label string } {
	return []struct{ Value, Label string }{
		{"1h", "Last hour"},
		{"24h", "Last 24 hours"},
		{"7d", "Last 7 days"},
		{"30d", "Last 30 days"},
		{"90d", "Last 90 days"},
	}
}

// BuildQuery turns the builder fields into a HogQL query. Event names are
// returned separately as placeholder values rather than being written into the
// query, so a name containing quotes cannot change what the query does.
func BuildQuery(spec *RunQuerySpec) (string, map[string]any, error) {
	interval, ok := timeRangeIntervals[spec.timeRange()]
	if !ok {
		return "", nil, fmt.Errorf("unknown time range %q", spec.TimeRange)
	}

	selection, ok := aggregationSelects[spec.aggregation()]
	if !ok {
		return "", nil, fmt.Errorf("unknown aggregation %q", spec.Aggregation)
	}

	values := map[string]any{}

	//
	// The window is bounded at both ends. "Last 7 days" means up to now, and
	// client clock skew leaves a steady trickle of events dated in the future -
	// without the upper bound those sort to the top of a timestamp-ordered
	// result and crowd out the events that actually just happened.
	//
	conditions := []string{
		"timestamp > now() - " + interval,
		"timestamp <= now()",
	}

	if len(spec.Events) > 0 {
		conditions = append(conditions, "event IN {events}")
		values["events"] = spec.Events
	}

	query := strings.Builder{}
	fmt.Fprintf(&query, "SELECT %s FROM events WHERE %s", selection.columns, strings.Join(conditions, " AND "))

	if selection.groupBy != "" {
		fmt.Fprintf(&query, " GROUP BY %s", selection.groupBy)
	}

	if selection.orderBy != "" {
		fmt.Fprintf(&query, " ORDER BY %s", selection.orderBy)
	}

	//
	// The limit is an integer we have already bounded, so it is safe to write
	// into the query - and unlike a placeholder it is guaranteed to parse.
	//
	if selection.rowBound {
		fmt.Fprintf(&query, " LIMIT %d", spec.limit())
	}

	return query.String(), values, nil
}
