package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	RetryStrategyFixed       = "fixed"
	RetryStrategyExponential = "exponential"
	DefaultTimeout           = time.Second * 30
	MaxTimeout               = time.Second * 30
	RetryMinInterval         = time.Second * 5
	RetryMaxInterval         = time.Minute * 5
	RetryMaxAttempts         = 30

	SuccessOutputChannel = "success"
	FailureOutputChannel = "failure"
)

func init() {
	registry.RegisterComponent("http", &HTTP{})
}

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Spec struct {
	Method         string      `json:"method"`
	URL            string      `json:"url"`
	QueryParams    *[]KeyValue `json:"queryParams,omitempty"`
	Headers        *[]Header   `json:"headers,omitempty"`
	ContentType    *string     `json:"contentType,omitempty"`
	JSON           *any        `json:"json,omitempty"`
	XML            *string     `json:"xml,omitempty"`
	Text           *string     `json:"text,omitempty"`
	FormData       *[]KeyValue `json:"formData,omitempty"`
	TimeoutSeconds *int        `json:"timeoutSeconds,omitempty"`
	Retry          *RetrySpec  `json:"retry,omitempty"`
	SuccessCodes   *string     `json:"successCodes,omitempty"`
}

func (s *Spec) Timeout() time.Duration {
	if s.TimeoutSeconds == nil {
		return DefaultTimeout
	}

	return time.Duration(*s.TimeoutSeconds) * time.Second
}

func (s *Spec) GetSuccessCodes() string {
	if s.SuccessCodes == nil {
		return "2xx"
	}

	return *s.SuccessCodes
}

type RetrySpec struct {
	Enabled         bool   `json:"enabled" mapstructure:"enabled"`
	Strategy        string `json:"strategy" mapstructure:"strategy"`
	MaxAttempts     int    `json:"maxAttempts" mapstructure:"maxAttempts"`
	IntervalSeconds int    `json:"intervalSeconds" mapstructure:"intervalSeconds"`
}

type Metadata struct {
	TimeoutSeconds int            `json:"timeoutSeconds" mapstructure:"timeoutSeconds"`
	Retry          *RetryMetadata `json:"retry" mapstructure:"retry"`
}

type RetryMetadata struct {
	Strategy     string  `json:"strategy"`
	Interval     int     `json:"interval"`
	Attempts     int     `json:"attempts"`
	MaxAttempts  int     `json:"maxAttempts"`
	LastResponse string  `json:"lastResponse"`
	LastStatus   *int    `json:"lastStatus,omitempty"`
	LastError    string  `json:"lastError"`
	NextRetryAt  *string `json:"nextRetryAt,omitempty"`
}

type HTTP struct{}

func (e *HTTP) Name() string {
	return "http"
}

func (e *HTTP) Label() string {
	return "HTTP Request"
}

func (e *HTTP) Description() string {
	return "Make HTTP requests"
}

func (e *HTTP) Documentation() string {
	return `The HTTP component allows you to make HTTP requests to external APIs and services as part of your workflow.

## Use Cases

- **API integration**: Call external REST APIs
- **Webhook notifications**: Send notifications to external systems
- **Data fetching**: Retrieve data from external services
- **Service orchestration**: Coordinate with microservices

## Supported Methods

- GET, POST, PUT, DELETE, PATCH

## Request Configuration

- **URL**: The endpoint to call (supports expressions)
- **Method**: HTTP method to use
- **Query Parameters**: Optional URL query parameters
- **Headers**: Custom HTTP headers (header names cannot use expressions)
- **Body**: Request body in various formats:
  - **JSON**: Structured JSON payload
  - **Form Data**: URL-encoded form data
  - **Plain Text**: Raw text content
  - **XML**: XML formatted content

## Response Handling

The component emits the response with:
- **status**: HTTP status code
- **headers**: Response headers
- **body**: Parsed response body (JSON if possible, otherwise string)
`
}

func (e *HTTP) Icon() string {
	return "globe"
}

func (e *HTTP) Color() string {
	return "blue"
}

func (e *HTTP) Setup(ctx core.SetupContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	if spec.URL == "" {
		return fmt.Errorf("url is required")
	}

	if spec.Method == "" {
		return fmt.Errorf("method is required")
	}

	if spec.ContentType == nil {
		return nil
	}

	switch *spec.ContentType {
	case "application/json":
		if spec.JSON == nil {
			return fmt.Errorf("json is required")
		}

	case "application/x-www-form-urlencoded":
		if spec.FormData == nil {
			return fmt.Errorf("form data is required")
		}

	case "text/plain":
		if spec.Text == nil {
			return fmt.Errorf("text is required")
		}

	case "application/xml":
		if spec.XML == nil {
			return fmt.Errorf("xml is required")
		}
	}

	if spec.Retry != nil && spec.Retry.Enabled {
		if spec.Retry.Strategy != RetryStrategyFixed && spec.Retry.Strategy != RetryStrategyExponential {
			return fmt.Errorf("invalid retry strategy: %s", spec.Retry.Strategy)
		}

		if spec.Retry.MaxAttempts > RetryMaxAttempts {
			return fmt.Errorf("max attempts must be less than or equal to %d", RetryMaxAttempts)
		}

		if spec.Retry.IntervalSeconds < int(RetryMinInterval.Seconds()) {
			return fmt.Errorf("interval seconds must be greater than or equal to %d", int(RetryMinInterval.Seconds()))
		}

		if spec.Retry.IntervalSeconds > int(RetryMaxInterval.Seconds()) {
			return fmt.Errorf("interval seconds must be less than or equal to %d", int(RetryMaxInterval.Seconds()))
		}
	}

	return nil
}

func (e *HTTP) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: SuccessOutputChannel, Label: "Success"},
		{Name: FailureOutputChannel, Label: "Failure"},
	}
}

func (e *HTTP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "method",
			Type:     configuration.FieldTypeSelect,
			Label:    "Method",
			Required: true,
			Default:  "POST",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GET", Value: "GET"},
						{Label: "POST", Value: "POST"},
						{Label: "PUT", Value: "PUT"},
						{Label: "DELETE", Value: "DELETE"},
						{Label: "PATCH", Value: "PATCH"},
					},
				},
			},
		},
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "https://api.example.com/endpoint",
		},
		{
			Name:        "queryParams",
			Label:       "Query Params",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Query parameters to append to the URL",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Parameter",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Type:        configuration.FieldTypeString,
								Label:       "Key",
								Required:    true,
								Placeholder: "search",
							},
							{
								Name:        "value",
								Type:        configuration.FieldTypeString,
								Label:       "Value",
								Required:    true,
								Placeholder: "shoes",
							},
						},
					},
				},
			},
			Default: []map[string]any{
				{
					"key":   "foo",
					"value": "bar",
				},
			},
		},
		{
			Name:        "headers",
			Label:       "Headers",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Custom headers to send with this request",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Header",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Type:        configuration.FieldTypeString,
								Label:       "Header Name",
								Required:    true,
								Placeholder: "Content-Type",
							},
							{
								Name:        "value",
								Type:        configuration.FieldTypeString,
								Label:       "Header Value",
								Required:    true,
								Placeholder: "application/json",
							},
						},
					},
				},
			},
			Default: []map[string]any{
				{
					"name":  "X-Foo",
					"value": "Bar",
				},
			},
		},
		{
			Name:        "contentType",
			Label:       "Body",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Body content type for POST, PUT, and PATCH requests",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "JSON", Value: "application/json"},
						{Label: "Form Data", Value: "application/x-www-form-urlencoded"},
						{Label: "Plain Text", Value: "text/plain"},
						{Label: "XML", Value: "application/xml"},
					},
				},
			},
		},
		{
			Name:        "json",
			Type:        configuration.FieldTypeObject,
			Label:       "JSON Payload",
			Required:    false,
			Description: "The JSON object to send as the request body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
				{Field: "contentType", Values: []string{"application/json"}},
			},
			Default: "{\"foo\": \"bar\"}",
		},
		{
			Name:     "formData",
			Label:    "Form Data",
			Type:     configuration.FieldTypeList,
			Required: false,
			Default: []map[string]any{
				{"key": "", "value": ""},
			},
			Description: "Key-value pairs to send as form data",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
				{Field: "contentType", Values: []string{"application/x-www-form-urlencoded"}},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Parameter",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Type:        configuration.FieldTypeString,
								Label:       "Key",
								Required:    true,
								Placeholder: "username",
							},
							{
								Name:        "value",
								Type:        configuration.FieldTypeString,
								Label:       "Value",
								Required:    true,
								Placeholder: "john.doe",
							},
						},
					},
				},
			},
		},
		{
			Name:        "text",
			Type:        configuration.FieldTypeText,
			Label:       "Text Payload",
			Required:    false,
			Description: "Plain text to send as the request body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
				{Field: "contentType", Values: []string{"text/plain"}},
			},
			Placeholder: "Enter plain text content",
		},
		{
			Name:        "xml",
			Type:        configuration.FieldTypeXML,
			Label:       "XML Payload",
			Required:    false,
			Description: "XML content to send as the request body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
				{Field: "contentType", Values: []string{"application/xml"}},
			},
			Placeholder: "<?xml version=\"1.0\"?>\n<root>\n  <element>value</element>\n</root>",
		},
		{
			Name:        "successCodes",
			Type:        configuration.FieldTypeString,
			Label:       "Overwrite success definition",
			Required:    false,
			Togglable:   true,
			Description: "Comma-separated list of success status codes (e.g., 200, 201, 2xx). Leave empty for default 2xx behavior",
			Default:     "2xx",
		},
		{
			Name:        "timeoutSeconds",
			Type:        configuration.FieldTypeNumber,
			Label:       "Timeout (seconds)",
			Description: "Timeout in seconds for each request attempt",
			Default:     10,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
					Max: func() *int { max := int(MaxTimeout.Seconds()); return &max }(),
				},
			},
		},
		{
			Name:        "retry",
			Type:        configuration.FieldTypeObject,
			Label:       "Retry",
			Required:    false,
			Description: "Retry configuration",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "enabled",
							Label:       "Enable retries",
							Type:        configuration.FieldTypeBool,
							Required:    false,
							Default:     false,
							Description: "Retry the request on failure.",
						},
						{
							Name:        "strategy",
							Type:        configuration.FieldTypeSelect,
							Label:       "Strategy",
							Required:    false,
							Default:     RetryStrategyFixed,
							Description: "Retry strategy",
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Fixed", Value: RetryStrategyFixed},
										{Label: "Exponential", Value: RetryStrategyExponential},
									},
								},
							},
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "enabled", Values: []string{"true"}},
							},
						},
						{
							Name:        "maxAttempts",
							Label:       "Max Attempts",
							Type:        configuration.FieldTypeNumber,
							Required:    false,
							Default:     5,
							Description: "Maximum number of retry attempts.",
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "enabled", Values: []string{"true"}},
							},
						},
						{
							Name:        "intervalSeconds",
							Label:       "Retry interval (seconds)",
							Type:        configuration.FieldTypeNumber,
							Required:    false,
							Default:     15,
							Description: "Seconds to wait between retry attempts",
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "enabled", Values: []string{"true"}},
							},
						},
					},
				},
			},
		},
	}
}

func (e *HTTP) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (e *HTTP) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	//
	// Handle request without retries configured.
	//
	if spec.Retry == nil || !spec.Retry.Enabled {
		return e.executeRequestWithoutRetry(ctx, spec)
	}

	//
	// Handle request with retries configured
	//
	response, err := e.executeRequest(ctx.Logger, ctx.HTTP, spec)
	if err == nil && e.isSuccessfulResponse(response.StatusCode, spec.GetSuccessCodes()) {
		return e.processResponse(ctx.Metadata, ctx.ExecutionState, response, spec)
	}

	//
	// Set the retry metadata and schedule the action call to retry the request.
	//
	interval := time.Duration(spec.Retry.IntervalSeconds) * time.Second
	nextRetryAt := time.Now().Add(interval).Format(time.RFC3339)
	metadata := Metadata{
		TimeoutSeconds: int(spec.Timeout().Seconds()),
		Retry: &RetryMetadata{
			Attempts:    1,
			Interval:    spec.Retry.IntervalSeconds,
			NextRetryAt: &nextRetryAt,
			MaxAttempts: spec.Retry.MaxAttempts,
			Strategy:    spec.Retry.Strategy,
		},
	}

	if err != nil {
		metadata.Retry.LastError = err.Error()
	}

	if response != nil {
		metadata.Retry.LastStatus = &response.StatusCode
		body, _ := io.ReadAll(response.Body)
		if body != nil {
			metadata.Retry.LastResponse = string(body)
		}
	}

	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("retryRequest", map[string]any{}, interval)
}

func (e *HTTP) executeRequestWithoutRetry(ctx core.ExecutionContext, spec Spec) error {
	response, err := e.executeRequest(ctx.Logger, ctx.HTTP, spec)
	if err != nil {
		return ctx.ExecutionState.Emit(
			FailureOutputChannel,
			"http.request.failed",
			[]any{map[string]any{
				"error": fmt.Sprintf("error executing request: %v", err),
			}},
		)
	}

	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return ctx.ExecutionState.Emit(
			FailureOutputChannel,
			"http.request.failed",
			[]any{map[string]any{
				"status":  response.StatusCode,
				"headers": response.Header,
				"error":   fmt.Errorf("failed to read response: %v", err),
			}},
		)
	}

	var bodyData any
	if len(respBody) > 0 {
		err := json.Unmarshal(respBody, &bodyData)
		if err != nil {
			bodyData = string(respBody)
		}
	}

	if e.isSuccessfulResponse(response.StatusCode, spec.GetSuccessCodes()) {
		return ctx.ExecutionState.Emit(
			SuccessOutputChannel,
			"http.request.finished",
			[]any{map[string]any{
				"status":  response.StatusCode,
				"headers": response.Header,
				"body":    bodyData,
			}},
		)
	}

	return ctx.ExecutionState.Emit(
		FailureOutputChannel,
		"http.request.failed",
		[]any{map[string]any{
			"status":  response.StatusCode,
			"headers": response.Header,
			"body":    bodyData,
		}},
	)
}

func (e *HTTP) executeRequest(logger *log.Entry, httpCtx core.HTTPContext, spec Spec) (*http.Response, error) {
	var body io.Reader
	var contentType string
	var err error
	if spec.ContentType != nil && (spec.Method == "POST" || spec.Method == "PUT" || spec.Method == "PATCH") {
		body, contentType, err = e.serializePayload(spec)
		if err != nil {
			return nil, err
		}
	}

	requestURL := spec.URL
	if spec.QueryParams != nil && len(*spec.QueryParams) > 0 {
		parsedURL, parseErr := url.Parse(spec.URL)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse url: %w", parseErr)
		}

		query := parsedURL.Query()
		for _, param := range *spec.QueryParams {
			query.Set(param.Key, param.Value)
		}

		parsedURL.RawQuery = query.Encode()
		requestURL = parsedURL.String()
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), spec.Timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, spec.Method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if spec.Headers != nil {
		for _, header := range *spec.Headers {
			req.Header.Set(header.Name, header.Value)
		}
	}

	logger.Infof("[%s] %s", spec.Method, spec.URL)
	resp, err := httpCtx.Do(req)
	if err != nil {
		if reqCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("request timed out after %s", spec.Timeout())
		}

		return nil, err
	}

	//
	// Read the entire response body before returning, because the deferred
	// cancel() will cancel the request context and abort any in-flight reads.
	// We replace resp.Body with the buffered content so callers (processResponse)
	// can read it without hitting "context canceled" errors.
	//
	// See: https://github.com/superplanehq/superplane/issues/3141
	//
	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return resp, nil
}

func (e *HTTP) handleRequestFailure(metadataCtx core.MetadataContext, executionStateCtx core.ExecutionStateContext, resp *http.Response, reqErr error) error {
	var metadata Metadata
	if decodeErr := mapstructure.Decode(metadataCtx.Get(), &metadata); decodeErr != nil {
		return fmt.Errorf("failed to decode metadata: %w", decodeErr)
	}

	metadata.Retry.NextRetryAt = nil
	metadata.Retry.LastStatus = &resp.StatusCode
	if reqErr != nil {
		metadata.Retry.LastError = reqErr.Error()
	}

	body, _ := io.ReadAll(resp.Body)
	if body != nil {
		metadata.Retry.LastResponse = string(body)
	}

	var bodyData any
	if len(body) > 0 {
		err := json.Unmarshal(body, &bodyData)
		if err != nil {
			bodyData = string(body)
		}
	}

	if err := metadataCtx.Set(metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	errorResponse := map[string]any{
		"status":  resp.StatusCode,
		"headers": resp.Header,
		"body":    bodyData,
		"retry":   metadata.Retry,
	}

	if reqErr != nil {
		errorResponse["error"] = reqErr.Error()
	}

	return executionStateCtx.Emit(
		FailureOutputChannel,
		"http.request.failure",
		[]any{errorResponse},
	)
}

func (e *HTTP) processResponse(metadataCtx core.MetadataContext, executionStateCtx core.ExecutionStateContext, resp *http.Response, spec Spec) error {
	defer resp.Body.Close()

	var metadata Metadata
	err := mapstructure.Decode(metadataCtx.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return e.handleRequestFailure(metadataCtx, executionStateCtx, resp, fmt.Errorf("failed to read response: %v", err))
	}

	var bodyData any
	if len(respBody) > 0 {
		err := json.Unmarshal(respBody, &bodyData)
		if err != nil {
			bodyData = string(respBody)
		}
	}

	if e.isSuccessfulResponse(resp.StatusCode, spec.GetSuccessCodes()) {
		return executionStateCtx.Emit(
			SuccessOutputChannel,
			"http.request.finished",
			[]any{map[string]any{
				"status":  resp.StatusCode,
				"headers": resp.Header,
				"body":    bodyData,
			}},
		)
	}

	return executionStateCtx.Emit(
		FailureOutputChannel,
		"http.request.failed",
		[]any{map[string]any{
			"status":  resp.StatusCode,
			"headers": resp.Header,
			"body":    bodyData,
		}},
	)
}

func (e *HTTP) isSuccessfulResponse(statusCode int, successCodes string) bool {
	codes := strings.Split(successCodes, ",")
	for _, code := range codes {
		code = strings.TrimSpace(code)

		if strings.HasSuffix(code, "xx") {
			prefix := strings.TrimSuffix(code, "xx")
			statusStr := strconv.Itoa(statusCode)
			if strings.HasPrefix(statusStr, prefix) {
				return true
			}
		} else {
			expectedCode, err := strconv.Atoi(code)
			if err == nil && statusCode == expectedCode {
				return true
			}
		}
	}

	return false
}

func (e *HTTP) serializePayload(spec Spec) (io.Reader, string, error) {
	if spec.ContentType == nil {
		return nil, "", fmt.Errorf("content type is required")
	}

	contentType := *spec.ContentType
	switch contentType {
	case "application/json":
		data, err := json.Marshal(spec.JSON)
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal JSON payload: %w", err)
		}
		return bytes.NewReader(data), contentType, nil

	case "application/x-www-form-urlencoded":
		if spec.FormData == nil {
			return nil, "", fmt.Errorf("form data is required for application/x-www-form-urlencoded")
		}

		values := url.Values{}
		for _, kv := range *spec.FormData {
			values.Add(kv.Key, kv.Value)
		}
		return strings.NewReader(values.Encode()), contentType, nil

	case "text/plain":
		if spec.Text == nil {
			return nil, "", fmt.Errorf("text is required for text/plain")
		}

		return strings.NewReader(*spec.Text), contentType, nil

	case "application/xml":
		if spec.XML == nil {
			return nil, "", fmt.Errorf("xml is required for application/xml")
		}

		return strings.NewReader(*spec.XML), contentType, nil

	default:
		return nil, "", fmt.Errorf("unsupported content type: %s", contentType)
	}
}

func (e *HTTP) Actions() []core.Action {
	return []core.Action{
		{
			Name: "retryRequest",
		},
	}
}

func (e *HTTP) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "retryRequest":
		return e.handleRetryRequest(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (e *HTTP) handleRetryRequest(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata Metadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	spec := Spec{}
	err = mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	resp, err := e.executeRequest(ctx.Logger, ctx.HTTP, spec)

	//
	// Handle successful scenario
	//
	if err == nil && e.isSuccessfulResponse(resp.StatusCode, spec.GetSuccessCodes()) {
		return e.processResponse(ctx.Metadata, ctx.ExecutionState, resp, spec)
	}

	//
	// If we should not retry anymore, fail here.
	//
	if metadata.Retry.Attempts >= metadata.Retry.MaxAttempts {
		if err != nil {
			return e.processResponse(ctx.Metadata, ctx.ExecutionState, resp, spec)
		}

		return e.handleRequestFailure(ctx.Metadata, ctx.ExecutionState, resp, err)
	}

	//
	// We should still retry.
	// Update metadata and schedule the action call to retry the request.
	//
	nextInterval := e.calculateNextRetryDelay(metadata.Retry.Strategy, metadata.Retry.Attempts, spec.Retry.IntervalSeconds)
	nextRetryAt := time.Now().Add(nextInterval).Format(time.RFC3339)
	newMetadata := Metadata{
		TimeoutSeconds: int(spec.Timeout().Seconds()),
		Retry: &RetryMetadata{
			Interval:    int(nextInterval.Seconds()),
			NextRetryAt: &nextRetryAt,
			MaxAttempts: spec.Retry.MaxAttempts,
			Strategy:    spec.Retry.Strategy,
			Attempts:    metadata.Retry.Attempts + 1,
		},
	}

	if err != nil {
		newMetadata.Retry.LastError = err.Error()
	}

	if resp != nil {
		newMetadata.Retry.LastStatus = &resp.StatusCode
		body, _ := io.ReadAll(resp.Body)
		if body != nil {
			newMetadata.Retry.LastResponse = string(body)
		}
	}

	err = ctx.Metadata.Set(newMetadata)
	if err != nil {
		return err
	}

	ctx.Logger.Infof("Scheduling retry in %s - %d/%d", nextInterval, newMetadata.Retry.Attempts, spec.Retry.MaxAttempts)
	return ctx.Requests.ScheduleActionCall("retryRequest", map[string]any{}, nextInterval)
}

func (e *HTTP) calculateNextRetryDelay(strategy string, attempt int, intervalSeconds int) time.Duration {
	switch strategy {
	case RetryStrategyExponential:
		return time.Duration(intervalSeconds) * time.Second * time.Duration(math.Pow(2, float64(attempt)))
	default:
		return time.Duration(intervalSeconds) * time.Second
	}
}

func (e *HTTP) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (e *HTTP) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (e *HTTP) Cleanup(ctx core.SetupContext) error {
	return nil
}
