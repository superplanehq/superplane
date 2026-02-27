package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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

type SecretKeyRef struct {
	Secret string `json:"secret" mapstructure:"secret"`
	Key    string `json:"key" mapstructure:"key"`
}

func (r SecretKeyRef) IsSet() bool {
	return r.Secret != "" && r.Key != ""
}

const (
	// Kept for backward compatibility with previously saved node configurations.
	authMethodNone   = "none"
	authMethodBasic  = "basic"
	authMethodBearer = "bearer"
	authMethodAPIKey = "api_key"

	authLocationHeader = "header"
	authLocationQuery  = "query"
)

type AuthorizationSpec struct {
	AuthMethod string       `json:"authMethod" mapstructure:"authMethod"`
	Username   string       `json:"username" mapstructure:"username"`
	Password   SecretKeyRef `json:"password" mapstructure:"password"`
	Token      SecretKeyRef `json:"token" mapstructure:"token"`
	Prefix     string       `json:"prefix" mapstructure:"prefix"`
	APIKey     SecretKeyRef `json:"apiKey" mapstructure:"apiKey"`
	Location   string       `json:"location" mapstructure:"location"`
	Name       string       `json:"name" mapstructure:"name"`
}

type Spec struct {
	Method          string              `json:"method"`
	URL             string              `json:"url"`
	QueryParams     *[]KeyValue         `json:"queryParams,omitempty"`
	Headers         *[]Header           `json:"headers,omitempty"`
	Authorization   *AuthorizationSpec `json:"authorization,omitempty" mapstructure:"authorization"`
	ContentType     *string             `json:"contentType,omitempty"`
	JSON            *any                `json:"json,omitempty"`
	XML             *string             `json:"xml,omitempty"`
	Text            *string             `json:"text,omitempty"`
	FormData        *[]KeyValue         `json:"formData,omitempty"`
	SuccessCodes    *string             `json:"successCodes,omitempty"`
	TimeoutStrategy *string             `json:"timeoutStrategy,omitempty"`
	TimeoutSeconds  *int                `json:"timeoutSeconds,omitempty"`
	Retries         *int                `json:"retries,omitempty"`
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
		return e.validateAuthorization(spec.Authorization)
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

	return e.validateAuthorization(spec.Authorization)
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
			Name:        "authorization",
			Label:       "Authorization",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Configure request authorization using organization secrets",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "authMethod",
							Label:       "Method",
							Type:        configuration.FieldTypeSelect,
							Description: "Authorization scheme to apply",
							Required:    true,
							Default:     authMethodBearer,
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Basic Auth", Value: authMethodBasic},
										{Label: "Bearer Token", Value: authMethodBearer},
										{Label: "API Key", Value: authMethodAPIKey},
									},
								},
							},
						},
						{
							Name:                 "username",
							Type:                 configuration.FieldTypeString,
							Label:                "Username",
							Required:             false,
							Description:          "Username used in Basic authorization",
							Placeholder:          "e.g. api-user",
							RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{authMethodBasic}}},
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{authMethodBasic}}},
						},
						{
							Name:                 "password",
							Type:                 configuration.FieldTypeSecretKey,
							Label:                "Password",
							Required:             false,
							Description:          "Secret key for Basic authorization password",
							RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{authMethodBasic}}},
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{authMethodBasic}}},
						},
						{
							Name:                 "token",
							Type:                 configuration.FieldTypeSecretKey,
							Label:                "Token",
							Required:             false,
							Description:          "Secret key for bearer token",
							RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{authMethodBearer}}},
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{authMethodBearer}}},
						},
						{
							Name:                 "prefix",
							Type:                 configuration.FieldTypeString,
							Label:                "Prefix",
							Required:             false,
							Default:              "Bearer",
							Description:          "Token prefix used in Authorization header",
							Placeholder:          "Bearer",
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{authMethodBearer}}},
						},
						{
							Name:                 "apiKey",
							Type:                 configuration.FieldTypeSecretKey,
							Label:                "API Key",
							Required:             false,
							Description:          "Secret key for API key authorization",
							RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{authMethodAPIKey}}},
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{authMethodAPIKey}}},
						},
						{
							Name:                 "location",
							Type:                 configuration.FieldTypeSelect,
							Label:                "Location",
							Required:             false,
							Default:              authLocationHeader,
							Description:          "Where to place the API key",
							RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{authMethodAPIKey}}},
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{authMethodAPIKey}}},
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Header", Value: authLocationHeader},
										{Label: "Query Parameter", Value: authLocationQuery},
									},
								},
							},
						},
						{
							Name:                 "name",
							Type:                 configuration.FieldTypeString,
							Label:                "Name",
							Required:             false,
							Description:          "Header or query parameter name",
							Placeholder:          "e.g. X-API-Key",
							RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{authMethodAPIKey}}},
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{authMethodAPIKey}}},
						},
					},
				},
			},
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
	if err := e.validateAuthorization(spec.Authorization); err != nil {
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

	resp, cancel, err := e.executeRequest(ctx.HTTP, ctx.Secrets, spec, currentTimeout)
	if err != nil {
		if retryMetadata.Attempt < retryMetadata.MaxRetries {
			return e.scheduleRetry(ctx, err.Error(), retryMetadata)
		}

		return e.handleRequestError(ctx, err, retryMetadata.Attempt+1)
	}
	defer cancel()

	var isSuccess bool
	if spec.SuccessCodes != nil && *spec.SuccessCodes != "" {
		isSuccess = e.matchesSuccessCode(resp.StatusCode, *spec.SuccessCodes)
	} else {
		isSuccess = e.matchesSuccessCode(resp.StatusCode, "2xx")
	}

	if !isSuccess && retryMetadata.Attempt < retryMetadata.MaxRetries {

		return e.scheduleRetry(ctx, fmt.Sprintf("HTTP status %d", resp.StatusCode), retryMetadata)
	}

	err = e.processResponse(ctx, resp, spec, currentTimeout)
	if err != nil {
		if retryMetadata.Attempt < retryMetadata.MaxRetries {
			return e.scheduleRetry(ctx, err.Error(), retryMetadata)
		}

		return e.handleRequestError(ctx, err, retryMetadata.Attempt+1)
	}

	return nil
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
		Secrets:        ctx.Secrets,
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

type requestAuthValues struct {
	HeaderName  string
	HeaderValue string
	QueryName   string
	QueryValue  string
}

func (e *HTTP) executeRequest(httpCtx core.HTTPContext, secretsCtx core.SecretsContext, spec Spec, timeout time.Duration) (*http.Response, context.CancelFunc, error) {
	var body io.Reader
	var contentType string
	var err error
	if spec.ContentType != nil && (spec.Method == "POST" || spec.Method == "PUT" || spec.Method == "PATCH") {
		body, contentType, err = e.serializePayload(spec)
		if err != nil {
			return nil, nil, err
		}
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), timeout)

	authValues, err := e.resolveAuthorization(spec.Authorization, secretsCtx)
	if err != nil {
		cancel()
		return nil, nil, err
	}

	parsedURL, parseErr := url.Parse(spec.URL)
	if parseErr != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to parse url: %w", parseErr)
	}

	query := parsedURL.Query()
	if authValues.QueryName != "" {
		query.Set(authValues.QueryName, authValues.QueryValue)
	}
	if spec.QueryParams != nil && len(*spec.QueryParams) > 0 {
		for _, param := range *spec.QueryParams {
			query.Set(param.Key, param.Value)
		}
	}
	parsedURL.RawQuery = query.Encode()
	requestURL := parsedURL.String()

	req, err := http.NewRequestWithContext(reqCtx, spec.Method, requestURL, body)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if authValues.HeaderName != "" {
		req.Header.Set(authValues.HeaderName, authValues.HeaderValue)
	}

	if spec.Headers != nil {
		for _, header := range *spec.Headers {
			req.Header.Set(header.Name, header.Value)
		}
	}

	resp, err := httpCtx.Do(req)
	if err != nil {
		cancel()
		if reqCtx.Err() == context.DeadlineExceeded {
			return nil, nil, fmt.Errorf("request timed out after %s", timeout)
		}
		if reqCtx.Err() == context.Canceled {
			return nil, nil, fmt.Errorf("request was canceled")
		}

		return nil, nil, err
	}

	return resp, cancel, nil
}

func (e *HTTP) validateAuthorization(auth *AuthorizationSpec) error {
	if auth == nil || auth.AuthMethod == "" || auth.AuthMethod == authMethodNone {
		return nil
	}

	switch auth.AuthMethod {
	case authMethodBasic:
		if strings.TrimSpace(auth.Username) == "" {
			return fmt.Errorf("authorization.username is required for basic auth")
		}
		if !auth.Password.IsSet() {
			return fmt.Errorf("authorization.password secret and key are required for basic auth")
		}
	case authMethodBearer:
		if !auth.Token.IsSet() {
			return fmt.Errorf("authorization.token secret and key are required for bearer auth")
		}
	case authMethodAPIKey:
		if !auth.APIKey.IsSet() {
			return fmt.Errorf("authorization.apiKey secret and key are required for api key auth")
		}
		if strings.TrimSpace(auth.Name) == "" {
			return fmt.Errorf("authorization.name is required for api key auth")
		}

		location := auth.Location
		if location == "" {
			location = authLocationHeader
		}
		if location != authLocationHeader && location != authLocationQuery {
			return fmt.Errorf("authorization.location must be header or query for api key auth")
		}
	default:
		return fmt.Errorf("unsupported authorization method: %s", auth.AuthMethod)
	}

	return nil
}

func (e *HTTP) resolveAuthorization(auth *AuthorizationSpec, secretsCtx core.SecretsContext) (requestAuthValues, error) {
	values := requestAuthValues{}

	if auth == nil || auth.AuthMethod == "" || auth.AuthMethod == authMethodNone {
		return values, nil
	}
	if secretsCtx == nil {
		return values, fmt.Errorf("secrets context is required for authorization")
	}

	switch auth.AuthMethod {
	case authMethodBasic:
		password, err := secretsCtx.GetKey(auth.Password.Secret, auth.Password.Key)
		if err != nil {
			return values, fmt.Errorf("failed to resolve basic auth password from secret")
		}

		token := base64.StdEncoding.EncodeToString([]byte(auth.Username + ":" + string(password)))
		values.HeaderName = "Authorization"
		values.HeaderValue = "Basic " + token
	case authMethodBearer:
		token, err := secretsCtx.GetKey(auth.Token.Secret, auth.Token.Key)
		if err != nil {
			return values, fmt.Errorf("failed to resolve bearer token from secret")
		}

		prefix := strings.TrimSpace(auth.Prefix)
		if prefix == "" {
			prefix = "Bearer"
		}
		values.HeaderName = "Authorization"
		values.HeaderValue = prefix + " " + string(token)
	case authMethodAPIKey:
		apiKey, err := secretsCtx.GetKey(auth.APIKey.Secret, auth.APIKey.Key)
		if err != nil {
			return values, fmt.Errorf("failed to resolve api key from secret")
		}

		location := auth.Location
		if location == "" {
			location = authLocationHeader
		}
		if location == authLocationHeader {
			values.HeaderName = auth.Name
			values.HeaderValue = string(apiKey)
		} else {
			values.QueryName = auth.Name
			values.QueryValue = string(apiKey)
		}
	default:
		return values, fmt.Errorf("unsupported authorization method: %s", auth.AuthMethod)
	}

	return values, nil
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

func (e *HTTP) processResponse(ctx core.ExecutionContext, resp *http.Response, spec Spec, timeout time.Duration) error {
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("request timed out after %s", timeout)
		}
		if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
			return fmt.Errorf("request was canceled while reading response")
		}
		return fmt.Errorf("failed to read response body: %w", err)
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
