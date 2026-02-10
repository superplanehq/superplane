package newrelic

import (
	"encoding/json"
	"fmt"
)

const (
	// US Region
	restAPIBaseUS      = "https://api.newrelic.com/v2"
	nerdGraphAPIBaseUS = "https://api.newrelic.com/graphql"
	metricsAPIBaseUS   = "https://metric-api.newrelic.com/metric/v1"

	// EU Region
	restAPIBaseEU      = "https://api.eu.newrelic.com/v2"
	nerdGraphAPIBaseEU = "https://api.eu.newrelic.com/graphql"
	metricsAPIBaseEU   = "https://metric-api.eu.newrelic.com/metric/v1"
)

type Account struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type APIError struct {
	ErrorDetails struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *APIError) Error() string {
	if e.ErrorDetails.Message != "" {
		return fmt.Sprintf("%s: %s", e.ErrorDetails.Title, e.ErrorDetails.Message)
	}
	return e.ErrorDetails.Title
}

func parseErrorResponse(url string, body []byte, statusCode int) error {
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.ErrorDetails.Title != "" {
		return fmt.Errorf("request to %s failed: %w", url, &apiErr)
	}
	// Include full URL and response body for debugging 404s and other errors
	return fmt.Errorf("request to %s failed with status %d: %s", url, statusCode, string(body))
}
