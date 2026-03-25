package logfire

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueryLogfire struct{}

type QueryLogfireConfiguration struct {
	SQL          string `json:"sql" mapstructure:"sql"`
	ProjectID    string `json:"projectId" mapstructure:"projectId"`
	MinTimestamp string `json:"minTimestamp,omitempty" mapstructure:"minTimestamp"`
	MaxTimestamp string `json:"maxTimestamp,omitempty" mapstructure:"maxTimestamp"`
	Limit        int    `json:"limit,omitempty" mapstructure:"limit"`
	RowOriented  bool   `json:"rowOriented,omitempty" mapstructure:"rowOriented"`
}

// To ensure query is read-only.
var forbiddenWriteSQLPattern = regexp.MustCompile(`(?i)\b(insert|update|delete)\b`)

func (c *QueryLogfire) Name() string {
	return "logfire.queryLogfire"
}

func (c *QueryLogfire) Label() string {
	return "Query Logfire"
}

func (c *QueryLogfire) Description() string {
	return "Execute a read-only SQL query against Logfire Query API"
}

func (c *QueryLogfire) Documentation() string {
	return `The Query Logfire component executes a read-only SQL query against Logfire and returns query results for use in downstream steps.

## Use Cases

- **Investigate traces and spans**: Query recent records for errors, latency spikes, or specific services
- **Build reporting workflows**: Export Logfire data into Slack, email, dashboards, or data stores
- **Conditional automation**: Query for specific conditions, then branch workflow logic based on returned rows
- **Scheduled analytics**: Run recurring SQL queries to monitor usage and operational metrics

## Configuration

- **SQL**: Required SQL query (supports expressions). Example: ` + "`SELECT start_timestamp, message FROM records LIMIT 10`" + `
- **Min Timestamp**: Optional ISO 8601 lower bound. For ` + "`records`" + ` queries, filters by ` + "`start_timestamp`" + ` (` + "`>= min_timestamp`" + `)
- **Max Timestamp**: Optional ISO 8601 upper bound. For ` + "`records`" + ` queries, filters by ` + "`start_timestamp`" + ` (` + "`<= max_timestamp`" + `)
- **Limit**: Optional maximum rows to return. If omitted, Logfire defaults to ` + "`500`" + `; maximum is ` + "`10000`" + `
- **Row Oriented**: Optional JSON format toggle. ` + "`false`" + ` returns column-oriented JSON; ` + "`true`" + ` returns row-oriented JSON

## Output

Emits one ` + "`logfire.query`" + ` event containing the Logfire query response (for example ` + "`columns`" + ` and/or ` + "`rows`" + `, depending on format options).

Use this output to transform, filter, or route query results to other components.`
}

func (c *QueryLogfire) Icon() string {
	return "flame"
}

func (c *QueryLogfire) Color() string {
	return "gray"
}

func (c *QueryLogfire) ExampleOutput() map[string]any {
	return map[string]any{
		"columns": []any{
			map[string]any{
				"name": "start_timestamp",
				"type": "timestamp",
			},
			map[string]any{
				"name": "message",
				"type": "text",
			},
		},
		"rows": []any{
			[]any{"2026-01-01T00:00:00Z", "Example Logfire record"},
		},
	}
}

func (c *QueryLogfire) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *QueryLogfire) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "sql",
			Label:       "SQL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Read-only SQL query to execute against Logfire",
			Placeholder: "SELECT start_timestamp FROM records LIMIT 1",
		},
		{
			Name:        "projectId",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Logfire project to query (scopes the generated read token)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "project",
					UseNameAsValue: false,
				},
			},
		},
		{
			Name:        "minTimestamp",
			Label:       "Min Timestamp",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional minimum timestamp bound for query results",
			Placeholder: "2026-01-01T00:00:00Z",
		},
		{
			Name:        "maxTimestamp",
			Label:       "Max Timestamp",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional maximum timestamp bound for query results",
			Placeholder: "2026-01-02T00:00:00Z",
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Optional maximum number of rows to return",
			Placeholder: "100",
		},
		{
			Name:        "rowOriented",
			Label:       "Row Oriented",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Return row-oriented data",
			Default:     false,
		},
	}
}

func (c *QueryLogfire) Setup(ctx core.SetupContext) error {
	var config QueryLogfireConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeQueryLogfireConfiguration(config)

	if config.SQL == "" {
		return fmt.Errorf("sql is required")
	}
	if err := validateReadOnlySQL(config.SQL); err != nil {
		return err
	}

	if config.Limit < 0 {
		return fmt.Errorf("limit must be greater than or equal to 0")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Logfire client: %w", err)
	}

	if config.ProjectID != "" {
		readToken, err := client.CreateReadToken(config.ProjectID, defaultReadTokenName)
		if err != nil {
			return fmt.Errorf("failed to create Logfire read token for project %q: %w", config.ProjectID, err)
		}
		// Override the read token so ValidateCredentials() runs against the selected project.
		client.ReadToken = readToken
	}

	if err := client.ValidateCredentials(); err != nil {
		if config.ProjectID != "" {
			return fmt.Errorf("invalid Logfire project selection %q: %w", config.ProjectID, err)
		}
		return fmt.Errorf("invalid Logfire read token: %w", err)
	}

	return nil
}

func (c *QueryLogfire) Execute(ctx core.ExecutionContext) error {
	var config QueryLogfireConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeQueryLogfireConfiguration(config)
	if err := validateReadOnlySQL(config.SQL); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Logfire client: %w", err)
	}

	if config.ProjectID != "" {
		readToken, err := client.CreateReadToken(config.ProjectID, defaultReadTokenName)
		if err != nil {
			return fmt.Errorf("failed to create Logfire read token for project %q: %w", config.ProjectID, err)
		}
		client.ReadToken = readToken
	}

	response, err := client.ExecuteQuery(QueryRequest{
		SQL:          config.SQL,
		MinTimestamp: config.MinTimestamp,
		MaxTimestamp: config.MaxTimestamp,
		Limit:        config.Limit,
		RowOriented:  config.RowOriented,
	})
	if err != nil {
		return fmt.Errorf("failed to execute Logfire query: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"logfire.query",
		[]any{response},
	)
}

func sanitizeQueryLogfireConfiguration(config QueryLogfireConfiguration) QueryLogfireConfiguration {
	config.SQL = strings.TrimSpace(config.SQL)
	config.ProjectID = strings.TrimSpace(config.ProjectID)
	config.MinTimestamp = strings.TrimSpace(config.MinTimestamp)
	config.MaxTimestamp = strings.TrimSpace(config.MaxTimestamp)
	return config
}

func validateReadOnlySQL(sql string) error {
	if forbiddenWriteSQLPattern.MatchString(sql) {
		return fmt.Errorf("only read-only queries are allowed: INSERT, UPDATE, and DELETE are not permitted")
	}
	return nil
}

func (c *QueryLogfire) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *QueryLogfire) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *QueryLogfire) Actions() []core.Action {
	return []core.Action{}
}

func (c *QueryLogfire) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *QueryLogfire) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *QueryLogfire) Cleanup(ctx core.SetupContext) error {
	return nil
}
