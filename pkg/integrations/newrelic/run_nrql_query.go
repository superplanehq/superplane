package newrelic

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const RunNRQLQueryPayloadType = "newrelic.nrqlQuery"

// PollInterval is the interval between async query poll attempts.
const PollInterval = 10 * time.Second

type RunNRQLQuery struct{}

// RunNRQLQuerySpec defines the configuration for the Run NRQL Query component.
type RunNRQLQuerySpec struct {
	Account string `json:"account" mapstructure:"account"`
	Query   string `json:"query"   mapstructure:"query"`
}

type RunNRQLQueryPayload struct {
	Results     []map[string]interface{} `json:"results"      mapstructure:"results"`
	TotalResult map[string]interface{}   `json:"totalResult,omitempty" mapstructure:"totalResult"`
	Metadata    *NRQLMetadata            `json:"metadata,omitempty"    mapstructure:"metadata"`
	Query       string                   `json:"query"          mapstructure:"query"`
	AccountID   string                   `json:"accountId"      mapstructure:"accountId"`
}

// RunNRQLQueryNodeMetadata stores verified account details in the node metadata.
type RunNRQLQueryNodeMetadata struct {
	Account *Account `json:"account" mapstructure:"account"`
	Manual  bool     `json:"manual"  mapstructure:"manual"`
}

// RunNRQLQueryExecutionMetadata stores async query state for polling.
type RunNRQLQueryExecutionMetadata struct {
	QueryId   string `json:"queryId"   mapstructure:"queryId"`
	AccountID int64  `json:"accountId" mapstructure:"accountId"`
	Query     string `json:"query"     mapstructure:"query"`
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

- **Account**: The New Relic account to query (select from dropdown)
- **Query**: The NRQL query string to execute (required)

## How It Works

Queries use New Relic's asynchronous query API:

1. The query is submitted via NerdGraph.
2. If results are available within the internal API timeout, they are emitted immediately.
3. If the query takes longer, the component polls for results automatically.
4. Polling intervals respect the "Retry-After" suggestion from New Relic API, falling back to 10 seconds if not provided.

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
			Name:        "account",
			Label:       "Account",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The New Relic account to query",
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
	}
}

func (c *RunNRQLQuery) Setup(ctx core.SetupContext) error {
	// NRQL queries require a User API Key (NRAK-) for NerdGraph access
	userAPIKey, err := ctx.Integration.GetConfig("userApiKey")
	if err != nil || len(userAPIKey) == 0 {
		msg := "User API Key is required for this component. Please configure it in the Integration settings."
		return fmt.Errorf("%s", msg)
	}

	spec := RunNRQLQuerySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	accountIDStr := spec.Account

	if accountIDStr == "" {
		return fmt.Errorf("account is required (select from dropdown)")
	}

	// Guard: reject unresolved template tags early
	if isUnresolvedTemplate(accountIDStr) {
		return fmt.Errorf("account ID contains unresolved template variable: %s — configure the upstream trigger first", accountIDStr)
	}

	if spec.Query == "" {
		return fmt.Errorf("query is required")
	}

	if isUnresolvedTemplate(spec.Query) {
		return fmt.Errorf("query contains unresolved template variable: %s — configure the upstream trigger first", spec.Query)
	}

	//
	// Integration Resource Validation
	//
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	accounts, err := client.ListAccounts(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list accounts: %v", err)
	}

	var verifiedAccount *Account
	for _, acc := range accounts {
		if strconv.FormatInt(acc.ID, 10) == strings.TrimSpace(accountIDStr) {
			verifiedAccount = &acc
			break
		}
	}

	if verifiedAccount == nil {
		return fmt.Errorf("account ID %s not found or not accessible with the provided API key", accountIDStr)
	}

	// Persist verified details to metadata
	metadata := RunNRQLQueryNodeMetadata{
		Account: verifiedAccount,
		Manual:  true,
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
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

	// Extract account ID from configuration
	accountIDStr := spec.Account

	// Fallback: try to resolve from upstream trigger event data (ctx.Data)
	if accountIDStr == "" {
		accountIDStr = extractStringFromData(ctx.Data, "accountId", "account_id", "account")
	}

	query := spec.Query
	if query == "" {
		query = extractStringFromData(ctx.Data, "query", "nrqlQuery")
	}

	// Guard: reject unresolved template tags — don't waste an API call
	if isUnresolvedTemplate(accountIDStr) {
		return fmt.Errorf("account ID contains unresolved template variable: %s — ensure the upstream trigger is configured and variables are mapped", accountIDStr)
	}
	if isUnresolvedTemplate(query) {
		return fmt.Errorf("query contains unresolved template variable: %s — ensure the upstream trigger is configured and variables are mapped", query)
	}

	if accountIDStr == "" {
		return fmt.Errorf("account ID is missing — set it in configuration or connect an upstream trigger that provides it")
	}

	if query == "" {
		return fmt.Errorf("NRQL query is missing — set it in configuration or connect an upstream trigger that provides it")
	}

	// Parse account ID to int64 for the NerdGraph API call
	accountID, err := strconv.ParseInt(strings.TrimSpace(accountIDStr), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid account ID '%s': must be a numeric string (e.g. '1234567')", accountIDStr)
	}

	// Execute async NRQL query via NerdGraph (fixed 10s timeout)
	response, err := client.RunNRQLQuery(context.Background(), accountID, query)
	if err != nil {
		return fmt.Errorf("failed to execute NRQL query: %v", err)
	}

	// If query completed within the timeout, emit results immediately
	if response.QueryProgress == nil || response.QueryProgress.Completed {
		return c.emitResults(ctx, response, query, accountIDStr)
	}

	// Query is still running — store state and schedule polling
	if err := ctx.Metadata.Set(RunNRQLQueryExecutionMetadata{
		QueryId:   response.QueryProgress.QueryId,
		AccountID: accountID,
		Query:     query,
	}); err != nil {
		return fmt.Errorf("failed to store execution metadata: %w", err)
	}

	interval := PollInterval
	if response.QueryProgress.RetryAfter > 0 {
		interval = time.Duration(response.QueryProgress.RetryAfter) * time.Second
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, interval)
}

func (c *RunNRQLQuery) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RunNRQLQuery) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunNRQLQuery) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *RunNRQLQuery) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *RunNRQLQuery) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := RunNRQLQueryExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode execution metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	response, err := client.PollNRQLQuery(context.Background(), metadata.AccountID, metadata.QueryId)
	if err != nil {
		return fmt.Errorf("failed to poll NRQL query: %v", err)
	}

	// If still running, check deadline and schedule another poll
	if response.QueryProgress != nil && !response.QueryProgress.Completed {
		// Stop polling if we've passed the retry deadline
		if response.QueryProgress.RetryDeadline > 0 && time.Now().UnixMilli() > response.QueryProgress.RetryDeadline {
			return fmt.Errorf("async query timed out: retry deadline exceeded")
		}

		interval := PollInterval
		if response.QueryProgress.RetryAfter > 0 {
			interval = time.Duration(response.QueryProgress.RetryAfter) * time.Second
		}

		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, interval)
	}

	// Results are ready — emit them
	accountIDStr := strconv.FormatInt(metadata.AccountID, 10)
	payload := RunNRQLQueryPayload{
		Results:     response.Results,
		TotalResult: response.TotalResult,
		Metadata:    response.Metadata,
		Query:       metadata.Query,
		AccountID:   accountIDStr,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		RunNRQLQueryPayloadType,
		[]any{payload},
	)
}

func (c *RunNRQLQuery) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *RunNRQLQuery) Cleanup(ctx core.SetupContext) error {
	return nil
}

// emitResults emits the NRQL query results to the default output channel.
func (c *RunNRQLQuery) emitResults(ctx core.ExecutionContext, response *NRQLQueryResponse, query, accountIDStr string) error {
	payload := RunNRQLQueryPayload{
		Results:     response.Results,
		TotalResult: response.TotalResult,
		Metadata:    response.Metadata,
		Query:       query,
		AccountID:   accountIDStr,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		RunNRQLQueryPayloadType,
		[]any{payload},
	)
}

// isUnresolvedTemplate detects raw template tags like {{account_id}} that
// haven't been substituted by the platform engine. Calling the API with
// these would always fail, so we intercept them early.
func isUnresolvedTemplate(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}

// extractStringFromData attempts to read a string value from upstream trigger
// event data (ctx.Data) by trying each key in order. Returns "" if nothing found.
func extractStringFromData(data any, keys ...string) string {
	if data == nil {
		return ""
	}

	m, ok := data.(map[string]any)
	if !ok {
		return ""
	}

	for _, key := range keys {
		if val, exists := m[key]; exists && val != nil {
			return extractResourceID(val)
		}
	}

	return ""
}

func extractResourceID(v any) string {
	if v == nil {
		return ""
	}

	// Handle raw string
	if s, ok := v.(string); ok {
		return s
	}

	// Handle raw numbers (int, float, etc.)
	switch n := v.(type) {
	case int:
		return strconv.Itoa(n)
	case int64:
		return strconv.FormatInt(n, 10)
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(n), 'f', -1, 32)
	}

	// Handle maps
	if m, ok := v.(map[string]any); ok {
		// Keys to check in order of preference
		keys := []string{"id", "ID", "value", "Value", "accountId", "account"}

		for _, key := range keys {
			if val, exists := m[key]; exists && val != nil {
				return extractResourceID(val) // Recursively extract from the found value
			}
		}
	}

	// Fallback to string representation
	return fmt.Sprintf("%v", v)
}
