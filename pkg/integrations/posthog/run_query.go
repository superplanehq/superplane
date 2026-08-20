package posthog

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type RunQuery struct{}

type RunQuerySpec struct {
	ProjectID   string   `json:"projectId" mapstructure:"projectId"`
	Mode        string   `json:"mode" mapstructure:"mode"`
	Events      []string `json:"events" mapstructure:"events"`
	Aggregation string   `json:"aggregation" mapstructure:"aggregation"`
	TimeRange   string   `json:"timeRange" mapstructure:"timeRange"`
	Limit       int      `json:"limit" mapstructure:"limit"`
	Query       string   `json:"query" mapstructure:"query"`
}

// The mode decides which half of the form is in play. A spec saved before the
// builder existed carries no mode, so a bare query still means HogQL.
func (s *RunQuerySpec) mode() string {
	if s.Mode != "" {
		return s.Mode
	}

	if strings.TrimSpace(s.Query) != "" {
		return QueryModeHogQL
	}

	return QueryModeBuilder
}

func (s *RunQuerySpec) aggregation() string {
	if s.Aggregation == "" {
		return AggregationEvents
	}

	return s.Aggregation
}

func (s *RunQuerySpec) timeRange() string {
	if s.TimeRange == "" {
		return DefaultTimeRange
	}

	return s.TimeRange
}

func (s *RunQuerySpec) limit() int {
	if s.Limit <= 0 {
		return DefaultLimit
	}

	if s.Limit > MaxLimit {
		return MaxLimit
	}

	return s.Limit
}

func (c *RunQuery) Name() string {
	return "posthog.runQuery"
}

func (c *RunQuery) Label() string {
	return "Run Query"
}

func (c *RunQuery) Description() string {
	return "Run a HogQL query against a PostHog project"
}

func (c *RunQuery) Documentation() string {
	return `The Run Query component reads product analytics data from a PostHog project and returns
the rows it produced. You can either build the query from a few fields or write HogQL yourself.

## Use Cases

- **Pull events on demand**: Fetch the events matching a condition at the moment the workflow runs
- **Enrichment**: Look up product usage for a person or account before acting on it
- **Reporting**: Aggregate product metrics and forward them to another system
- **Guardrails**: Check a metric before a workflow continues to a deployment or rollout step

## Configuration

- **Project**: The PostHog project to query.
- **Query**: Whether to build the query from fields or write HogQL yourself.

### Build a query

- **Events**: The events to include, picked from the event names in the project. Leave empty to
  include every event.
- **Return**: What you get back.
  - *Matching events* — one row per event, with its properties
  - *Count per event* — how many times each event was captured
  - *Total count* — a single number: how many events matched
  - *Unique users* — a single number: how many distinct users matched
- **Time range**: How far back to look, from the last hour up to the last 90 days.
- **Row limit**: The most rows to return, up to 10000. Ignored by the two single-number options.

Event names you pick are sent to PostHog as query parameters rather than pasted into the query
text, so a name containing quotes cannot change what the query does.

The builder does not apply the project's "internal and test users" filter. PostHog applies that
filter only to queries carrying its ` + "`{filters}`" + ` placeholder, and a query using that
placeholder cannot also use query parameters - so the builder keeps the parameters. Exclude
internal traffic with a condition in **Write HogQL** mode when you need it.

### Write HogQL

HogQL is PostHog's SQL dialect. It reads the same tables you see in the PostHog UI, including
` + "`events`" + `, ` + "`persons`" + `, and ` + "`sessions`" + `. Use this mode for anything the builder
cannot express - joins, window functions, or querying tables other than ` + "`events`" + `.

` + "```sql" + `
SELECT event, distinct_id, timestamp
FROM events
WHERE timestamp > now() - INTERVAL 1 DAY
ORDER BY timestamp DESC
LIMIT 100
` + "```" + `

## Output

Returns a single object with the query results:

- ` + "`rows`" + ` — the result rows, each keyed by column name so expressions can read ` + "`row.event`" + `
- ` + "`columns`" + ` — the column names, in the order PostHog returned them
- ` + "`rowCount`" + ` — how many rows were returned
- ` + "`projectId`" + ` — the project the query ran against

## Notes

- The personal API key needs the **Query: Read** scope.
- Results are returned in full. The builder caps rows for you; in HogQL mode add a
  ` + "`LIMIT`" + ` yourself to keep executions small.`
}

func (c *RunQuery) Icon() string {
	return "posthog"
}

func (c *RunQuery) Color() string {
	return "gray"
}

func (c *RunQuery) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RunQuery) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectId",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The PostHog project to query",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "mode",
			Label:       "Query",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     QueryModeBuilder,
			Description: "Pick the events to look at, or write the HogQL yourself",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label:       "Build a query",
							Value:       QueryModeBuilder,
							Description: "Choose events, a time range, and what to return",
						},
						{
							Label:       "Write HogQL",
							Value:       QueryModeHogQL,
							Description: "Write the query yourself for anything the builder cannot express",
						},
					},
				},
			},
		},
		{
			Name:        "events",
			Label:       "Events",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The events to include. Leave empty to include every event in the project.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "event",
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "projectId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "projectId"},
						},
					},
				},
			},
			VisibilityConditions: builderFields,
		},
		{
			Name:        "aggregation",
			Label:       "Return",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     AggregationEvents,
			Description: "What the query should return",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label:       "Matching events",
							Value:       AggregationEvents,
							Description: "One row per event, with its properties",
						},
						{
							Label:       "Count per event",
							Value:       AggregationCountByEvent,
							Description: "How many times each event was captured",
						},
						{
							Label:       "Total count",
							Value:       AggregationTotalCount,
							Description: "A single number: how many events matched",
						},
						{
							Label:       "Unique users",
							Value:       AggregationUniqueUsers,
							Description: "A single number: how many distinct users matched",
						},
					},
				},
			},
			VisibilityConditions: builderFields,
		},
		{
			Name:                 "timeRange",
			Label:                "Time range",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			Default:              DefaultTimeRange,
			Description:          "How far back to look",
			TypeOptions:          &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: timeRangeFieldOptions()}},
			VisibilityConditions: builderFields,
		},
		{
			Name:        "limit",
			Label:       "Row limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     DefaultLimit,
			Description: "The most rows to return. Ignored when the query returns a single number.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: &minLimit,
					Max: &maxLimit,
				},
			},
			VisibilityConditions: builderFields,
		},
		{
			Name:        "query",
			Label:       "HogQL Query",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "The HogQL query to run, for example: SELECT event, timestamp FROM events LIMIT 100",
			TypeOptions: &configuration.TypeOptions{
				Text: &configuration.TextTypeOptions{
					Language: "sql",
				},
			},
			VisibilityConditions: hogQLFields,
			RequiredConditions:   []configuration.RequiredCondition{{Field: "mode", Values: []string{QueryModeHogQL}}},
		},
	}
}

var (
	minLimit = 1
	maxLimit = MaxLimit

	builderFields = []configuration.VisibilityCondition{{Field: "mode", Values: []string{QueryModeBuilder}}}
	hogQLFields   = []configuration.VisibilityCondition{{Field: "mode", Values: []string{QueryModeHogQL}}}
)

func timeRangeFieldOptions() []configuration.FieldOption {
	options := make([]configuration.FieldOption, 0, len(TimeRangeOptions()))
	for _, timeRange := range TimeRangeOptions() {
		options = append(options, configuration.FieldOption{Label: timeRange.Label, Value: timeRange.Value})
	}

	return options
}

func (c *RunQuery) Setup(ctx core.SetupContext) error {
	spec := RunQuerySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return spec.validate()
}

func (c *RunQuery) Execute(ctx core.ExecutionContext) error {
	spec := RunQuerySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := spec.validate(); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create PostHog client: %w", err)
	}

	request, err := spec.resolve()
	if err != nil {
		return err
	}

	response, err := client.Query(spec.ProjectID, request)
	if err != nil {
		return fmt.Errorf("failed to run query: %w", err)
	}

	rows := RowsToMaps(response)
	result := map[string]any{
		"projectId": spec.ProjectID,
		"rows":      rows,
		"columns":   response.Columns,
		"rowCount":  len(rows),
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"posthog.queryResult",
		[]any{result},
	)
}

func (c *RunQuery) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunQuery) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RunQuery) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RunQuery) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RunQuery) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RunQuery) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (s *RunQuerySpec) validate() error {
	if strings.TrimSpace(s.ProjectID) == "" {
		return errors.New("project is required")
	}

	if s.mode() == QueryModeHogQL {
		if strings.TrimSpace(s.Query) == "" {
			return errors.New("query is required")
		}

		return nil
	}

	if _, ok := timeRangeIntervals[s.timeRange()]; !ok {
		return fmt.Errorf("unknown time range %q", s.TimeRange)
	}

	if _, ok := aggregationSelects[s.aggregation()]; !ok {
		return fmt.Errorf("unknown aggregation %q", s.Aggregation)
	}

	return nil
}

// resolve turns the spec into the request to send, whichever mode it is in.
func (s *RunQuerySpec) resolve() (QueryRequest, error) {
	if s.mode() == QueryModeHogQL {
		return QueryRequest{Query: s.Query}, nil
	}

	query, values, err := BuildQuery(s)
	if err != nil {
		return QueryRequest{}, err
	}

	return QueryRequest{Query: query, Values: values}, nil
}
