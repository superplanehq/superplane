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
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
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
	Method          string      `json:"method"`
	URL             string      `json:"url"`
	QueryParams     *[]KeyValue `json:"queryParams,omitempty"`
	Headers         *[]Header   `json:"headers,omitempty"`
	ContentType     *string     `json:"contentType,omitempty"`
	JSON            *any        `json:"json,omitempty"`
	XML             *string     `json:"xml,omitempty"`
	Text            *string     `json:"text,omitempty"`
	FormData        *[]KeyValue `json:"formData,omitempty"`
	SuccessCodes    *string     `json:"successCodes,omitempty"`
	TimeoutStrategy *string     `json:"timeoutStrategy,omitempty"`
	TimeoutSeconds  *int        `json:"timeoutSeconds,omitempty"`
	Retries         *int        `json:"retries,omitempty"`
}

type RetryMetadata struct {
	Attempt         int    `json:"attempt"`
	MaxRetries      int    `json:"maxRetries"`
	TimeoutStrategy string `json:"timeoutStrategy"`
	TimeoutSeconds  int    `json:"timeoutSeconds"`
	LastError       string `json:"lastError,omitempty"`
	TotalRetries    int    `json:"totalRetries"`
	FinalStatus     int    `json:"finalStatus,omitempty"`
	Result          string `json:"result"`
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

## Error Handling & Retries

Configure timeout and retry behavior:
- **Fixed timeout**: Same timeout for all retry attempts
- **Exponential backoff**: Timeout increases with each retry (capped at 120s)
- **Success codes**: Define which status codes are considered successful (default: 2xx)

## Output Events

- **http.request.finished**: Emitted on successful request
- **http.request.failed**: Emitted when request fails after all retries
- **http.request.error**: Emitted on network/parsing errors`
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

	return nil
}

func (e *HTTP) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
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
			Default: "[{\"key\": \"foo\", \"value\": \"bar\"}]",
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
								Name:               "name",
								Type:               configuration.FieldTypeString,
								Label:              "Header Name",
								Required:           true,
								Placeholder:        "Content-Type",
								DisallowExpression: true,
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
			Default: "[{\"name\": \"X-Foo\", \"value\": \"Bar\"}]",
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
			Name:        "timeoutStrategy",
			Type:        configuration.FieldTypeSelect,
			Label:       "Set Timeout and Retries",
			Required:    false,
			Togglable:   true,
			Description: "Configure timeout and retry behavior for failed requests",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Fixed", Value: "fixed"},
						{Label: "Exponential", Value: "exponential"},
					},
				},
			},
		},
		{
			Name:        "timeoutSeconds",
			Type:        configuration.FieldTypeNumber,
			Label:       "Timeout (seconds)",
			Description: "Timeout in seconds for each request attempt",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "timeoutStrategy", Values: []string{"fixed", "exponential"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "timeoutStrategy", Values: []string{"fixed", "exponential"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
					Max: func() *int { max := 300; return &max }(),
				},
			},
			Default: "10",
		},
		{
			Name:        "retries",
			Type:        configuration.FieldTypeNumber,
			Label:       "Retries",
			Description: "Number of retry attempts. Wait longer after each failed attempt (timeout capped to 120s)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "timeoutStrategy", Values: []string{"fixed", "exponential"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "timeoutStrategy", Values: []string{"fixed", "exponential"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 0; return &min }(),
					Max: func() *int { max := 10; return &max }(),
				},
			},
			Default: "3",
		},
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

func (e *HTTP) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (e *HTTP) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	retryMetadata := RetryMetadata{
		Attempt:         0,
		MaxRetries:      0,
		TimeoutStrategy: "fixed",
		TimeoutSeconds:  30,
		TotalRetries:    0,
		Result:          "pending",
	}

	if spec.TimeoutStrategy != nil && *spec.TimeoutStrategy != "" {
		retryMetadata.TimeoutStrategy = *spec.TimeoutStrategy
	}

	if spec.TimeoutSeconds != nil {
		retryMetadata.TimeoutSeconds = *spec.TimeoutSeconds
	}

	if spec.Retries != nil {
		retryMetadata.MaxRetries = *spec.Retries
	}

	err = ctx.Metadata.Set(retryMetadata)
	if err != nil {
		return err
	}

	return e.executeHTTPRequest(ctx, spec, retryMetadata)
}

func (e *HTTP) executeHTTPRequest(ctx core.ExecutionContext, spec Spec, retryMetadata RetryMetadata) error {
	currentTimeout := e.calculateTimeoutForAttempt(retryMetadata.TimeoutStrategy, retryMetadata.TimeoutSeconds, retryMetadata.Attempt)

	resp, err := e.executeRequest(ctx.HTTP, spec, currentTimeout)
	if err != nil {
		if retryMetadata.Attempt < retryMetadata.MaxRetries {
			return e.scheduleRetry(ctx, err.Error(), retryMetadata)
		}

		return e.handleRequestError(ctx, err, retryMetadata.Attempt+1)
	}

	var isSuccess bool
	if spec.SuccessCodes != nil && *spec.SuccessCodes != "" {
		isSuccess = e.matchesSuccessCode(resp.StatusCode, *spec.SuccessCodes)
	} else {
		isSuccess = e.matchesSuccessCode(resp.StatusCode, "2xx")
	}

	if !isSuccess && retryMetadata.Attempt < retryMetadata.MaxRetries {

		return e.scheduleRetry(ctx, fmt.Sprintf("HTTP status %d", resp.StatusCode), retryMetadata)
	}

	return e.processResponse(ctx, resp, spec)
}

func (e *HTTP) scheduleRetry(ctx core.ExecutionContext, lastError string, retryMetadata RetryMetadata) error {
	retryMetadata.Attempt++
	retryMetadata.TotalRetries++
	retryMetadata.LastError = lastError

	err := ctx.Metadata.Set(retryMetadata)
	if err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("retryRequest", map[string]any{}, e.calculateTimeoutForAttempt(retryMetadata.TimeoutStrategy, 1, retryMetadata.Attempt-1))
}

func (e *HTTP) handleRetryRequest(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := ctx.Metadata.Get()

	var retryMetadata RetryMetadata
	err := mapstructure.Decode(metadata, &retryMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode retry metadata: %w", err)
	}

	spec := Spec{}
	err = mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	execCtx := core.ExecutionContext{
		Configuration:  ctx.Configuration,
		ExecutionState: ctx.ExecutionState,
		Metadata:       ctx.Metadata,
		Requests:       ctx.Requests,
		Auth:           ctx.Auth,
		HTTP:           ctx.HTTP,
	}

	return e.executeHTTPRequest(execCtx, spec, retryMetadata)
}

func (e *HTTP) calculateTimeoutForAttempt(strategy string, timeoutSeconds int, attempt int) time.Duration {
	baseTimeout := time.Duration(timeoutSeconds) * time.Second

	if strategy == "exponential" {

		timeout := time.Duration(float64(baseTimeout) * math.Pow(2, float64(attempt)))
		maxTimeout := 120 * time.Second
		if timeout > maxTimeout {
			return maxTimeout
		}
		return timeout
	}

	return baseTimeout
}

func (e *HTTP) executeRequest(httpCtx core.HTTPContext, spec Spec, timeout time.Duration) (*http.Response, error) {
	var body io.Reader
	var contentType string
	var err error
	if spec.ContentType != nil && (spec.Method == "POST" || spec.Method == "PUT" || spec.Method == "PATCH") {
		body, contentType, err = e.serializePayload(spec)
		if err != nil {
			return nil, err
		}
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

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

	resp, err := httpCtx.Do(req)
	if err != nil {
		if reqCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("request timed out after %s", timeout)
		}

		return nil, err
	}

	return resp, nil
}

func (e *HTTP) handleRequestError(ctx core.ExecutionContext, err error, totalAttempts int) error {
	// Get current metadata and update with final result
	metadata := ctx.Metadata.Get()
	var retryMetadata RetryMetadata
	mapstructure.Decode(metadata, &retryMetadata)

	retryMetadata.Result = "error"
	retryMetadata.LastError = err.Error()

	if ctx.Metadata != nil {
		ctx.Metadata.Set(retryMetadata)
	}

	errorResponse := map[string]any{
		"error":    err.Error(),
		"attempts": totalAttempts,
	}
	emitErr := ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"http.request.error",
		[]any{errorResponse},
	)
	if emitErr != nil {
		return fmt.Errorf("request failed after %d attempts: %w (and failed to emit event: %v)", totalAttempts, err, emitErr)
	}

	err = ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, fmt.Sprintf("Request failed after %d attempts: %v", totalAttempts, err))
	if err != nil {
		return fmt.Errorf("request failed after %d attempts: %w (and failed to mark execution as failed: %v)", totalAttempts, err, err)
	}

	return nil
}

func (e *HTTP) processResponse(ctx core.ExecutionContext, resp *http.Response, spec Spec) error {
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		// Get current metadata and update with final result
		metadata := ctx.Metadata.Get()
		var retryMetadata RetryMetadata
		mapstructure.Decode(metadata, &retryMetadata)

		retryMetadata.Result = "error"
		retryMetadata.FinalStatus = resp.StatusCode
		retryMetadata.LastError = fmt.Sprintf("failed to read response body: %v", err)

		if ctx.Metadata != nil {
			ctx.Metadata.Set(retryMetadata)
		}

		errorResponse := map[string]any{
			"status": resp.StatusCode,
			"error":  fmt.Sprintf("failed to read response body: %v", err),
		}
		emitErr := ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"http.request.error",
			[]any{errorResponse},
		)
		if emitErr != nil {
			return fmt.Errorf("failed to read response: %w (and failed to emit event: %v)", err, emitErr)
		}
		return fmt.Errorf("failed to read response: %w", err)
	}

	var bodyData any
	if len(respBody) > 0 {
		err := json.Unmarshal(respBody, &bodyData)
		if err != nil {

			bodyData = string(respBody)
		}
	}

	response := map[string]any{
		"status":  resp.StatusCode,
		"headers": resp.Header,
		"body":    bodyData,
	}

	var isSuccess bool
	if spec.SuccessCodes != nil && *spec.SuccessCodes != "" {
		isSuccess = e.matchesSuccessCode(resp.StatusCode, *spec.SuccessCodes)
	} else {

		isSuccess = e.matchesSuccessCode(resp.StatusCode, "2xx")
	}

	// Get current metadata and update with final result
	metadata := ctx.Metadata.Get()
	var retryMetadata RetryMetadata
	mapstructure.Decode(metadata, &retryMetadata)

	retryMetadata.FinalStatus = resp.StatusCode
	if isSuccess {
		retryMetadata.Result = "success"
	} else {
		retryMetadata.Result = "failed"
	}

	if ctx.Metadata != nil {
		ctx.Metadata.Set(retryMetadata)
	}

	eventType := "http.request.finished"
	if !isSuccess {
		eventType = "http.request.failed"
	}

	if !isSuccess {
		ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, fmt.Sprintf("HTTP request failed with status %d", resp.StatusCode))
		return nil
	}

	err = ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		eventType,
		[]any{response},
	)

	if err != nil {
		return err
	}

	return nil
}

func (e *HTTP) matchesSuccessCode(statusCode int, successCodes string) bool {
	if successCodes == "" {
		successCodes = "2xx"
	}

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

func (e *HTTP) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (e *HTTP) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (e *HTTP) Cleanup(ctx core.SetupContext) error {
	return nil
}
