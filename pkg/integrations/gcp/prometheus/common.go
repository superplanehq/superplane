package prometheus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

// roleHintRead is the IAM role required to read metrics from Managed Service
// for Prometheus.
const roleHintRead = "roles/monitoring.viewer"

// instantQueryURL builds the GMP Prometheus-compatible instant-query URL,
// evaluating the expression at query time ("now"). Managed Service for
// Prometheus uses the singular `location/global` segment (not `locations`),
// unlike most Cloud APIs.
func instantQueryURL(project, query string) string {
	v := url.Values{}
	v.Set("query", query)
	return fmt.Sprintf(
		"%s/projects/%s/location/global/prometheus/api/v1/query?%s",
		queryBaseURL, project, v.Encode(),
	)
}

// rangeQueryURL builds the GMP Prometheus-compatible range-query URL.
func rangeQueryURL(project, query, start, end, step string) string {
	v := url.Values{}
	v.Set("query", query)
	v.Set("start", start)
	v.Set("end", end)
	v.Set("step", step)
	return fmt.Sprintf(
		"%s/projects/%s/location/global/prometheus/api/v1/query_range?%s",
		queryBaseURL, project, v.Encode(),
	)
}

// promResponse models the standard Prometheus HTTP API envelope returned by the
// GMP query frontend.
type promResponse struct {
	Status    string   `json:"status"`
	Data      promData `json:"data"`
	ErrorType string   `json:"errorType"`
	Error     string   `json:"error"`
}

type promData struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`
}

// runQuery issues the query and normalizes the Prometheus response into the
// component output payload (resultType, result, seriesCount).
func runQuery(client Client, fullURL string) (map[string]any, error) {
	body, err := client.GetURL(context.Background(), fullURL)
	if err != nil {
		return nil, err
	}

	var resp promResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse query response: %v", err)
	}
	if resp.Status != "success" {
		return nil, formatPromError(resp.ErrorType, resp.Error)
	}

	var result any
	if len(resp.Data.Result) > 0 {
		if err := json.Unmarshal(resp.Data.Result, &result); err != nil {
			return nil, fmt.Errorf("parse query result: %v", err)
		}
	}

	// Only vector/matrix results are arrays of series. scalar/string results are
	// a single [timestamp, value] pair — counting their array length would
	// misreport seriesCount as 2.
	seriesCount := 0
	switch resp.Data.ResultType {
	case "vector", "matrix":
		if series, ok := result.([]any); ok {
			seriesCount = len(series)
		}
	case "scalar", "string":
		if result != nil {
			seriesCount = 1
		}
	}

	return map[string]any{
		"resultType":  resp.Data.ResultType,
		"result":      result,
		"seriesCount": seriesCount,
	}, nil
}

// formatPromError renders the Prometheus error envelope (status="error") into a
// readable error.
func formatPromError(errorType, errMsg string) error {
	errMsg = strings.TrimSpace(errMsg)
	errorType = strings.TrimSpace(errorType)
	switch {
	case errMsg != "" && errorType != "":
		return fmt.Errorf("prometheus query failed (%s): %s", errorType, errMsg)
	case errMsg != "":
		return fmt.Errorf("prometheus query failed: %s", errMsg)
	default:
		return errors.New("prometheus query failed")
	}
}

// apiErrorMessage formats an API error for the execution state, appending an IAM
// hint on 403 since a missing monitoring.viewer role is the most common cause of
// query failures.
func apiErrorMessage(action string, err error) string {
	var apiErr *gcpcommon.GCPAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		return fmt.Sprintf("%s: %v — ensure the integration's service account has the %s IAM role", action, err, roleHintRead)
	}
	return fmt.Sprintf("%s: %v", action, err)
}
