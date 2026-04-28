package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	DefaultTimeout = time.Second * 30
	MaxTimeout     = time.Second * 30

	SuccessOutputChannel = "success"
	FailureOutputChannel = "failure"

	AuthorizationTypeNone   = "none"
	AuthorizationTypeBearer = "bearer"
)

func init() {
	registry.RegisterAction("graphql", &GraphQL{})
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
	URL            string      `json:"url"`
	Query          string      `json:"query"`
	Variables      *[]KeyValue `json:"variables,omitempty"`
	Headers        *[]Header   `json:"headers,omitempty"`
	Authorization  AuthSpec    `json:"authorization,omitempty" mapstructure:"authorization"`
	TimeoutSeconds *int        `json:"timeoutSeconds,omitempty"`
	SuccessCodes   *string     `json:"successCodes,omitempty"`
}

type AuthSpec struct {
	Type  string                     `json:"type" mapstructure:"type"`
	Token configuration.SecretKeyRef `json:"token" mapstructure:"token"`
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

type GraphQL struct{}

func (e *GraphQL) Name() string {
	return "graphql"
}

func (e *GraphQL) Label() string {
	return "GraphQL Request"
}

func (e *GraphQL) Description() string {
	return "Send a GraphQL query to an HTTP endpoint (GraphQL over JSON POST)"
}

func (e *GraphQL) Documentation() string {
	return `The GraphQL component runs a GraphQL document against a URL using the standard **GraphQL over HTTP** JSON body shape.

## Request

- **URL** — GraphQL HTTP endpoint (supports expressions)
- **Query** — Multi-line GraphQL document (no JSON escaping in the canvas)
- **Variables** — Key/value pairs merged into the request variables object
- **Headers** — Request headers
- **Authorization** — Optional bearer token stored in an organization Secret

## Response

- **status** - Response status code
- **headers** - Response headers
- **body** - Response body converted to JSON
`
}

func (e *GraphQL) Icon() string {
	return "network"
}

func (e *GraphQL) Color() string {
	return "violet"
}

func (e *GraphQL) Setup(ctx core.SetupContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	if spec.URL == "" {
		return fmt.Errorf("url is required")
	}

	if strings.TrimSpace(spec.Query) == "" {
		return fmt.Errorf("query is required")
	}

	return e.validateAuthorization(spec.Authorization)
}

func (e *GraphQL) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: SuccessOutputChannel, Label: "Success"},
		{Name: FailureOutputChannel, Label: "Failure"},
	}
}

func (e *GraphQL) Configuration() []configuration.Field {
	minTimeout := 1
	maxTimeout := int(MaxTimeout.Seconds())
	bearerOnly := []configuration.VisibilityCondition{{Field: "type", Values: []string{AuthorizationTypeBearer}}}

	return []configuration.Field{
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "https://api.example.com/graphql",
		},
		{
			Name:        "authorization",
			Label:       "Authorization",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optional Authorization header for this request",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "type",
							Label:       "Type",
							Type:        configuration.FieldTypeSelect,
							Required:    true,
							Default:     AuthorizationTypeNone,
							Description: "Authorization method",
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "None", Value: AuthorizationTypeNone},
										{Label: "Bearer token", Value: AuthorizationTypeBearer},
									},
								},
							},
						},
						{
							Name:                 "token",
							Label:                "Token",
							Type:                 configuration.FieldTypeSecretKey,
							Required:             false,
							Description:          "Stored credential that holds the bearer token",
							RequiredConditions:   []configuration.RequiredCondition{{Field: "type", Values: []string{AuthorizationTypeBearer}}},
							VisibilityConditions: bearerOnly,
						},
					},
				},
			},
		},
		{
			Name:        "query",
			Label:       "Query",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "query {\n  node {\n    id\n  }\n}",
		},
		{
			Name:        "variables",
			Label:       "Variables",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Key/value pairs merged into the request variables object",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Variable",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Type:        configuration.FieldTypeString,
								Label:       "Key",
								Required:    true,
								Placeholder: "owner",
							},
							{
								Name:        "value",
								Type:        configuration.FieldTypeString,
								Label:       "Value",
								Required:    true,
								Placeholder: "acme",
							},
						},
					},
				},
			},
			Default: []map[string]any{
				{
					"key":   "name",
					"value": "value",
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
								Placeholder: "X-Request-ID",
							},
							{
								Name:        "value",
								Type:        configuration.FieldTypeString,
								Label:       "Header Value",
								Required:    true,
								Placeholder: "request-123",
							},
						},
					},
				},
			},
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
			Description: "Timeout in seconds for the request",
			Default:     10,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: &minTimeout,
					Max: &maxTimeout,
				},
			},
		},
	}
}

func (e *GraphQL) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (e *GraphQL) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	response, err := e.postGraphQLRequest(ctx.Logger, ctx.HTTP, ctx.Secrets, spec)
	if err != nil {
		return ctx.ExecutionState.Emit(
			FailureOutputChannel,
			"graphql.request.failed",
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
			"graphql.request.failed",
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

	if e.isSuccessfulResponse(response.StatusCode, spec.GetSuccessCodes()) && !e.hasGraphQLErrors(bodyData) {
		return ctx.ExecutionState.Emit(
			SuccessOutputChannel,
			"graphql.request.finished",
			[]any{map[string]any{
				"status":  response.StatusCode,
				"headers": response.Header,
				"body":    bodyData,
			}},
		)
	}

	return ctx.ExecutionState.Emit(
		FailureOutputChannel,
		"graphql.request.failed",
		[]any{map[string]any{
			"status":  response.StatusCode,
			"headers": response.Header,
			"body":    bodyData,
		}},
	)
}

func (e *GraphQL) hasGraphQLErrors(bodyData any) bool {
	body, ok := bodyData.(map[string]any)
	if !ok {
		return false
	}

	errors, ok := body["errors"].([]any)
	return ok && len(errors) > 0
}

func (e *GraphQL) validateAuthorization(auth AuthSpec) error {
	if auth.Type == "" || auth.Type == AuthorizationTypeNone {
		return nil
	}

	if auth.Type != AuthorizationTypeBearer {
		return fmt.Errorf("invalid authorization type: %s", auth.Type)
	}

	if !auth.Token.IsSet() {
		return fmt.Errorf("bearer token credential is required")
	}

	return nil
}

func (e *GraphQL) buildRequestBody(spec Spec) ([]byte, error) {
	payload := map[string]any{
		"query": spec.Query,
	}

	if spec.Variables != nil && len(*spec.Variables) > 0 {
		vars := make(map[string]any, len(*spec.Variables))
		for _, kv := range *spec.Variables {
			if kv.Key == "" {
				continue
			}
			vars[kv.Key] = kv.Value
		}
		if len(vars) > 0 {
			payload["variables"] = vars
		}
	}

	return json.Marshal(payload)
}

func (e *GraphQL) postGraphQLRequest(logger *log.Entry, httpCtx core.HTTPContext, secrets core.SecretsContext, spec Spec) (*http.Response, error) {
	body, err := e.buildRequestBody(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), spec.Timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, spec.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if spec.Headers != nil {
		for _, header := range *spec.Headers {
			req.Header.Set(header.Name, header.Value)
		}
	}

	if spec.Authorization.Type == AuthorizationTypeBearer {
		if secrets == nil {
			return nil, fmt.Errorf("secrets context is required for bearer authorization")
		}

		token, err := secrets.GetKey(spec.Authorization.Token.Secret, spec.Authorization.Token.Key)
		if err != nil {
			return nil, fmt.Errorf("cannot get bearer token: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", strings.TrimSpace(string(token))))
	}

	logger.Infof("[POST] %s", spec.URL)
	resp, err := httpCtx.Do(req)
	if err != nil {
		if reqCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("request timed out after %s", spec.Timeout())
		}

		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return resp, nil
}

func (e *GraphQL) isSuccessfulResponse(statusCode int, successCodes string) bool {
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

func (e *GraphQL) Hooks() []core.Hook {
	return []core.Hook{}
}

func (e *GraphQL) HandleHook(_ core.ActionHookContext) error {
	return nil
}

func (e *GraphQL) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (e *GraphQL) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (e *GraphQL) Cleanup(_ core.SetupContext) error {
	return nil
}
