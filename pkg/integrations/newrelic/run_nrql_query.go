package newrelic

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const RunNRQLQueryPayloadType = "newrelic.nrqlQuery"

type RunNRQLQuery struct{}

type RunNRQLQuerySpec struct {
	AccountID string `json:"accountId"`
	Query     string `json:"query"`
	Timeout   int    `json:"timeout"`
}

type RunNRQLQueryPayload struct {
	Results     []map[string]interface{} `json:"results"`
	TotalResult map[string]interface{}   `json:"totalResult,omitempty"`
	Metadata    *NRQLMetadata            `json:"metadata,omitempty"`
	Query       string                   `json:"query"`
	AccountID   int64                    `json:"accountId"`
}

func (c *RunNRQLQuery) Name() string {
	return "newrelic.runNRQLQuery"
}

func (c *RunNRQLQuery) Label() string {
	return "Run NRQL Query"
}

func (c *RunNRQLQuery) Description() string {
	return "Execute NRQL queries to retrieve data from New Relic"
}

func (c *RunNRQLQuery) Documentation() string {
	return `The Run NRQL Query component allows you to execute NRQL queries via New Relic's NerdGraph API.

## Use Cases

- **Data retrieval**: Query telemetry data, metrics, events, and logs
- **Custom analytics**: Build custom analytics and reporting workflows
- **Monitoring**: Retrieve monitoring data for downstream processing
- **Alerting**: Query data to make decisions in workflow logic

## Configuration

- **Account ID**: The New Relic account ID to query against (required)
- **Query**: The NRQL query string to execute (required)
- **Timeout**: Query timeout in seconds (optional, default: 10, max: 120)

## Output

Returns query results including:
- **results**: Array of query result objects
- **totalResult**: Aggregated result for queries with aggregation functions
- **metadata**: Query metadata (event types, facets, messages, time window)
- **query**: The original NRQL query executed
- **accountId**: The account ID queried

## Example Queries

- Count transactions: ` + "`SELECT count(*) FROM Transaction SINCE 1 hour ago`" + `
- Average response time: ` + "`SELECT average(duration) FROM Transaction SINCE 1 day ago`" + `
- Faceted query: ` + "`SELECT count(*) FROM Transaction FACET appName SINCE 1 hour ago`" + `

## Notes

- Requires a valid New Relic API key with query permissions
- Queries are subject to New Relic's NRQL query limits
- Invalid NRQL syntax will return an error from the API`
}

func (c *RunNRQLQuery) Icon() string {
	return "newrelic"
}

func (c *RunNRQLQuery) Color() string {
	return "green"
}

func (c *RunNRQLQuery) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RunNRQLQuery) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "accountId",
			Label:       "Account ID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The New Relic account ID to query",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "account",
				},
			},
		},
		{
			Name:        "query",
			Label:       "NRQL Query",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The NRQL query to execute",
			Placeholder: "SELECT count(*) FROM Transaction SINCE 1 hour ago",
		},
		{
			Name:        "timeout",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Query timeout in seconds (default: 10, max: 120)",
			Default:     10,
			Placeholder: "10",
		},
	}
}

func (c *RunNRQLQuery) Setup(ctx core.SetupContext) error {
	spec := RunNRQLQuerySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.AccountID == "" {
		return fmt.Errorf("accountId is required")
	}

	if spec.Query == "" {
		return fmt.Errorf("query is required")
	}

	// Validate timeout if provided
	if spec.Timeout < 0 || spec.Timeout > 120 {
		return fmt.Errorf("timeout must be between 0 and 120 seconds")
	}

	return nil
}

func (c *RunNRQLQuery) Execute(ctx core.ExecutionContext) error {
	spec := RunNRQLQuerySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	// Parse account ID
	var accountID int64
	_, err = fmt.Sscanf(spec.AccountID, "%d", &accountID)
	if err != nil {
		return fmt.Errorf("invalid account ID: %v", err)
	}

	// Set default timeout if not provided
	timeout := spec.Timeout
	if timeout == 0 {
		timeout = 10
	}

	// Execute NRQL query
	response, err := client.RunNRQLQuery(context.Background(), accountID, spec.Query, timeout)
	if err != nil {
		return fmt.Errorf("failed to execute NRQL query: %v", err)
	}

	payload := RunNRQLQueryPayload{
		Results:     response.Results,
		TotalResult: response.TotalResult,
		Metadata:    response.Metadata,
		Query:       spec.Query,
		AccountID:   accountID,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		RunNRQLQueryPayloadType,
		[]any{payload},
	)
}

func (c *RunNRQLQuery) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RunNRQLQuery) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunNRQLQuery) Actions() []core.Action {
	return []core.Action{}
}

func (c *RunNRQLQuery) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *RunNRQLQuery) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *RunNRQLQuery) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RunNRQLQuery) ExampleOutput() map[string]any {
	return map[string]any{
		"results": []map[string]any{
			{
				"count": 1523,
			},
		},
		"metadata": map[string]any{
			"eventTypes": []string{"Transaction"},
			"messages":   []string{},
			"timeWindow": map[string]any{
				"begin": 1707559740000,
				"end":   1707563340000,
			},
		},
		"query":     "SELECT count(*) FROM Transaction SINCE 1 hour ago",
		"accountId": 1234567,
	}
}

