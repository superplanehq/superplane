package perplexity

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const SearchPayloadType = "perplexity.search.results"

type webSearch struct{}

type webSearchSpec struct {
	Query         string `mapstructure:"query"`
	MaxResults    int    `mapstructure:"maxResults"`
	DomainFilter  string `mapstructure:"domainFilter"`
	RecencyFilter string `mapstructure:"recencyFilter"`
}

type searchPayload struct {
	ID      string         `json:"id"`
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

func (c *webSearch) Name() string {
	return "perplexity.webSearch"
}

func (c *webSearch) Label() string {
	return "Web Search"
}

func (c *webSearch) Description() string {
	return "Search the web using Perplexity's Search API"
}

func (c *webSearch) Documentation() string {
	return `The Web Search component performs ranked web searches using the Perplexity Search API.

## Use Cases

- **Research**: Find up-to-date information on any topic
- **Monitoring**: Search for recent news or events matching specific criteria
- **Data collection**: Gather web results with titles, URLs, and snippets

## Configuration

- **Query**: The search query (supports expressions)
- **Max Results**: Number of results to return (1–10, default 5)
- **Domain Filter**: Comma-separated domains to restrict results to (or exclude with - prefix)
- **Recency Filter**: Limit results by age (day, week, month, year)

## Output

Returns a list of search results, each with:
- **title**: Page title
- **url**: Page URL
- **snippet**: Relevant excerpt
- **date**: Publication date`
}

func (c *webSearch) Icon() string {
	return "search"
}

func (c *webSearch) Color() string {
	return "teal"
}

func (c *webSearch) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *webSearch) Configuration() []configuration.Field {
	minResults := 1
	maxResults := 10

	return []configuration.Field{
		{
			Name:        "query",
			Label:       "Query",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Enter search query",
			Description: "The search query to run",
		},
		{
			Name:        "maxResults",
			Label:       "Max Results",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     5,
			Description: "Number of results to return (1–10)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: &minResults,
					Max: &maxResults,
				},
			},
		},
		{
			Name:        "domainFilter",
			Label:       "Domain Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g. example.com, docs.org",
			Description: "Comma-separated list of domains to include (or exclude with - prefix)",
		},
		{
			Name:     "recencyFilter",
			Label:    "Recency Filter",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Past Day", Value: "day"},
						{Label: "Past Week", Value: "week"},
						{Label: "Past Month", Value: "month"},
						{Label: "Past Year", Value: "year"},
					},
				},
			},
		},
	}
}

func (c *webSearch) Setup(ctx core.SetupContext) error {
	spec := webSearchSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Query == "" {
		return fmt.Errorf("query is required")
	}

	return nil
}

func (c *webSearch) Execute(ctx core.ExecutionContext) error {
	spec := webSearchSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Query == "" {
		return fmt.Errorf("query is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	req := SearchRequest{
		Query: spec.Query,
	}

	if spec.MaxResults > 0 {
		req.MaxResults = spec.MaxResults
	}

	if spec.DomainFilter != "" {
		domains := strings.Split(spec.DomainFilter, ",")
		filtered := make([]string, 0, len(domains))
		for _, d := range domains {
			if trimmed := strings.TrimSpace(d); trimmed != "" {
				filtered = append(filtered, trimmed)
			}
		}
		req.SearchDomainFilter = filtered
	}

	if spec.RecencyFilter != "" {
		req.SearchRecencyFilter = spec.RecencyFilter
	}

	response, err := client.Search(req)
	if err != nil {
		return err
	}

	payload := searchPayload{
		ID:      response.ID,
		Query:   spec.Query,
		Results: response.Results,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		SearchPayloadType,
		[]any{payload},
	)
}

func (c *webSearch) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *webSearch) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *webSearch) Actions() []core.Action {
	return []core.Action{}
}

func (c *webSearch) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *webSearch) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *webSearch) Cleanup(ctx core.SetupContext) error {
	return nil
}
