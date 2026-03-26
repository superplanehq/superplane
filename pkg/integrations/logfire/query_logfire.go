package logfire

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueryLogfire struct{}

type ProjectMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

type QueryLogfireNodeMetadata struct {
	Project *ProjectMetadata `json:"project,omitempty" mapstructure:"project"`
}

type QueryLogfireConfiguration struct {
	SQL          string `json:"sql" mapstructure:"sql"`
	ProjectID    string `json:"projectId" mapstructure:"projectId"`
	TimeWindow   string `json:"timeWindow,omitempty" mapstructure:"timeWindow"`
	MinTimestamp string `json:"minTimestamp,omitempty" mapstructure:"minTimestamp"`
	MaxTimestamp string `json:"maxTimestamp,omitempty" mapstructure:"maxTimestamp"`
	Limit        int    `json:"limit,omitempty" mapstructure:"limit"`
	RowOriented  bool   `json:"rowOriented,omitempty" mapstructure:"rowOriented"`
}

const (
	timeWindowNone   = "none"
	timeWindow5m     = "5m"
	timeWindow15m    = "15m"
	timeWindow1h     = "1h"
	timeWindow6h     = "6h"
	timeWindow24h    = "24h"
	timeWindow7d     = "7d"
	timeWindowCustom = "custom"
)

// To ensure query is read-only.
var forbiddenWriteSQLPattern = regexp.MustCompile(`(?i)\b(insert|update|delete|drop|alter|truncate|create|grant)\b`)

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

- **Project**: Required Logfire project to query (scopes the generated read token)
- **SQL**: Required SQL query (supports expressions). Example: ` + "`SELECT start_timestamp, message FROM records LIMIT 10`" + `
- **Time Window**: Optional preset time window (e.g., Last 5 minutes, Last 1 hour). Select "Custom" to specify exact timestamps
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
			Name:        "sql",
			Label:       "SQL",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Read-only SQL query to execute against Logfire",
			Placeholder: "SELECT start_timestamp, message FROM records LIMIT 10",
		},
		{
			Name:        "timeWindow",
			Label:       "Time Window",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Limit query results to a recent time window",
			Default:     timeWindowNone,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "No limit", Value: timeWindowNone},
						{Label: "Last 5 minutes", Value: timeWindow5m},
						{Label: "Last 15 minutes", Value: timeWindow15m},
						{Label: "Last 1 hour", Value: timeWindow1h},
						{Label: "Last 6 hours", Value: timeWindow6h},
						{Label: "Last 24 hours", Value: timeWindow24h},
						{Label: "Last 7 days", Value: timeWindow7d},
						{Label: "Custom", Value: timeWindowCustom},
					},
				},
			},
		},
		{
			Name:        "minTimestamp",
			Label:       "Min Timestamp",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Custom minimum timestamp bound (ISO 8601)",
			Placeholder: "2026-01-01T00:00:00Z",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "timeWindow", Values: []string{timeWindowCustom}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "timeWindow", Values: []string{timeWindowCustom}},
			},
		},
		{
			Name:        "maxTimestamp",
			Label:       "Max Timestamp",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Custom maximum timestamp bound (ISO 8601)",
			Placeholder: "2026-01-02T00:00:00Z",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "timeWindow", Values: []string{timeWindowCustom}},
			},
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

	if config.Limit > 10000 {
		return fmt.Errorf("limit must not exceed 10,000")
	}

	if config.ProjectID == "" {
		return fmt.Errorf("project is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Logfire client: %w", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list Logfire projects: %w", err)
	}

	var matchedProject *Project
	for i := range projects {
		if strings.TrimSpace(projects[i].ID) == config.ProjectID {
			matchedProject = &projects[i]
			break
		}
	}
	if matchedProject == nil {
		return fmt.Errorf("invalid Logfire project selection %q", config.ProjectID)
	}

	// Check if we already have a valid read token for this project.
	existingToken := findSecretValue(ctx.Integration, readTokenSecretNameForProject(config.ProjectID))
	if existingToken == "" {
		// Also check the legacy secret name for backward compatibility.
		existingToken = findSecretValue(ctx.Integration, readTokenSecretName)
	}

	if existingToken == "" || client.validateReadToken(existingToken) != nil {
		token, err := client.ProvisionReadToken(config.ProjectID)
		if err != nil {
			return fmt.Errorf("failed to provision read token: %w", err)
		}

		if err := ctx.Integration.SetSecret(readTokenSecretNameForProject(config.ProjectID), []byte(token)); err != nil {
			return fmt.Errorf("failed to store read token: %w", err)
		}
	} else if findSecretValue(ctx.Integration, readTokenSecretNameForProject(config.ProjectID)) == "" {
		// Migrate legacy token to per-project secret.
		if err := ctx.Integration.SetSecret(readTokenSecretNameForProject(config.ProjectID), []byte(existingToken)); err != nil {
			return fmt.Errorf("failed to store read token: %w", err)
		}
	}

	return ctx.Metadata.Set(QueryLogfireNodeMetadata{
		Project: &ProjectMetadata{ID: matchedProject.ID, Name: matchedProject.ProjectName},
	})
}

func (c *QueryLogfire) Execute(ctx core.ExecutionContext) error {
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

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Logfire client: %w", err)
	}

	readToken := findSecretValue(ctx.Integration, readTokenSecretNameForProject(config.ProjectID))
	if readToken == "" {
		readToken = findSecretValue(ctx.Integration, readTokenSecretName)
	}
	if readToken == "" {
		return fmt.Errorf("no read token available for project %q - please re-save the component to provision one", config.ProjectID)
	}

	minTs, maxTs := resolveTimeWindow(config)

	response, err := client.ExecuteQueryWithToken(readToken, QueryRequest{
		SQL:          config.SQL,
		MinTimestamp: minTs,
		MaxTimestamp: maxTs,
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
	config.TimeWindow = strings.TrimSpace(config.TimeWindow)
	config.MinTimestamp = strings.TrimSpace(config.MinTimestamp)
	config.MaxTimestamp = strings.TrimSpace(config.MaxTimestamp)
	return config
}

var timeWindowDurations = map[string]time.Duration{
	timeWindow5m:  5 * time.Minute,
	timeWindow15m: 15 * time.Minute,
	timeWindow1h:  time.Hour,
	timeWindow6h:  6 * time.Hour,
	timeWindow24h: 24 * time.Hour,
	timeWindow7d:  7 * 24 * time.Hour,
}

func resolveTimeWindow(config QueryLogfireConfiguration) (string, string) {
	if d, ok := timeWindowDurations[config.TimeWindow]; ok {
		now := time.Now().UTC()
		return now.Add(-d).Format(time.RFC3339), ""
	}

	if config.TimeWindow == timeWindowCustom {
		return config.MinTimestamp, config.MaxTimestamp
	}

	return "", ""
}

func validateReadOnlySQL(sql string) error {
	if forbiddenWriteSQLPattern.MatchString(sql) {
		return fmt.Errorf("only read-only queries are allowed: INSERT, UPDATE, DELETE, DROP, ALTER, TRUNCATE, CREATE, and GRANT are not permitted")
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
	var config QueryLogfireConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return nil
	}

	config = sanitizeQueryLogfireConfiguration(config)
	if config.ProjectID == "" {
		return nil
	}

	secretName := readTokenSecretNameForProject(config.ProjectID)
	if findSecretValue(ctx.Integration, secretName) != "" {
		_ = ctx.Integration.SetSecret(secretName, []byte{})
	}

	return nil
}
