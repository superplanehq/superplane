package datadog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeEnvironment = "environment"
	spansAggregatePath      = "/api/v2/spans/analytics/aggregate"
	logsAggregatePath       = "/api/v2/logs/analytics/aggregate"
	environmentFacet        = "env"
	environmentAggregateMax = 100
)

// ListEnvironments returns distinct env tag values seen on spans, then logs,
// for the last 30 days. service narrows the query when it is set.
func (c *Client) ListEnvironments(service string) ([]string, error) {
	query := environmentQuery(service)
	names, err := c.aggregateFacet(spansAggregatePath, environmentFacet, query, environmentAggregateMax)
	if err == nil {
		return names, nil
	}
	if !isForbidden(err) {
		return nil, err
	}

	names, logsErr := c.aggregateFacet(logsAggregatePath, environmentFacet, query, environmentAggregateMax)
	if logsErr != nil {
		return nil, err
	}
	return names, nil
}

func environmentQuery(service string) string {
	service = strings.TrimSpace(service)
	if service == "" {
		return "*"
	}
	return "service:" + quoteQueryValue(service)
}

func quoteQueryValue(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func (c *Client) aggregateFacet(path, facet, query string, limit int) ([]string, error) {
	payload := map[string]any{
		"data": map[string]any{
			"type": "aggregate_request",
			"attributes": map[string]any{
				"compute": []any{
					map[string]any{"aggregation": "count", "type": "total"},
				},
				"filter": map[string]any{
					"from":  "now-30d",
					"to":    "now",
					"query": query,
				},
				"group_by": []any{
					map[string]any{
						"facet": facet,
						"limit": limit,
						"sort":  map[string]any{"type": "alphabetical", "order": "asc"},
					},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling aggregate query: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	names, err := facetValues(responseBody, facet)
	if err != nil {
		return nil, fmt.Errorf("error parsing aggregate list: %w", err)
	}
	return names, nil
}

// Span aggregates return data as a list of buckets. Log aggregates return
// data.buckets. A key with apm_read hits the span shape first.
type facetBucket struct {
	By map[string]any `json:"by"`
}

func facetValues(body []byte, facet string) ([]string, error) {
	buckets, err := aggregateBuckets(body)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	names := make([]string, 0, len(buckets))
	for _, bucket := range buckets {
		name := strings.TrimSpace(facetString(bucket.By[facet]))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}

func aggregateBuckets(body []byte) ([]facetBucket, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	data := bytes.TrimSpace(envelope.Data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil, nil
	}

	if data[0] == '[' {
		var spans []struct {
			Attributes facetBucket `json:"attributes"`
		}
		if err := json.Unmarshal(data, &spans); err != nil {
			return nil, err
		}
		buckets := make([]facetBucket, 0, len(spans))
		for _, span := range spans {
			buckets = append(buckets, span.Attributes)
		}
		return buckets, nil
	}

	var logs struct {
		Buckets []facetBucket `json:"buckets"`
	}
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, err
	}
	return logs.Buckets, nil
}

func facetString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func isForbidden(err error) bool {
	return apiStatus(err) == http.StatusForbidden
}

func apiStatus(err error) int {
	apiError, ok := err.(*APIError)
	if !ok {
		return 0
	}
	return apiError.StatusCode
}

func environmentResources(names []string) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(names))
	for _, name := range names {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeEnvironment,
			ID:   name,
			Name: name,
		})
	}
	return resources
}
