package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
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
	Method       string      `json:"method"`
	URL          string      `json:"url"`
	Headers      *[]Header   `json:"headers,omitempty"`
	ContentType  *string     `json:"contentType,omitempty"`
	JSON         *any        `json:"json,omitempty"`
	XML          *string     `json:"xml,omitempty"`
	Text         *string     `json:"text,omitempty"`
	FormData     *[]KeyValue `json:"formData,omitempty"`
	SuccessCodes *string     `json:"successCodes,omitempty"`
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
			Name:        "headers",
			Label:       "Headers",
			Type:        configuration.FieldTypeTogglableList,
			Required:    false,
			Description: "Custom headers to send with this request",
			TypeOptions: &configuration.TypeOptions{
				TogglableList: &configuration.TogglableListTypeOptions{
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
		},
		{
			Name:        "contentType",
			Label:       "Content Type",
			Type:        configuration.FieldTypeTogglableSelect,
			Required:    false,
			Description: "The content type of the request body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
			},
			TypeOptions: &configuration.TypeOptions{
				TogglableSelect: &configuration.TogglableSelectTypeOptions{
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
			Type:        configuration.FieldTypeTogglableString,
			Label:       "Success Codes",
			Required:    false,
			Description: "Comma-separated list of success status codes (e.g., 200, 201, 2xx). Leave empty for default 2xx behavior",
			TypeOptions: &configuration.TypeOptions{
				TogglableString: &configuration.TogglableStringTypeOptions{
					Placeholder: "2xx, 3xx",
				},
			},
		},
	}
}

func (e *HTTP) Actions() []core.Action {
	return []core.Action{}
}

func (e *HTTP) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("http does not support actions")
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

	// Serialize payload based on content type
	var body io.Reader
	var contentType string
	if spec.ContentType != nil && (spec.Method == "POST" || spec.Method == "PUT" || spec.Method == "PATCH") {
		body, contentType, err = e.serializePayload(spec)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequest(spec.Method, spec.URL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Content-Type if we have one
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Apply custom headers if provided (can override Content-Type)
	if spec.Headers != nil {
		for _, header := range *spec.Headers {
			req.Header.Set(header.Name, header.Value)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Set metadata result to "error" for network/connection failures
		if ctx.MetadataContext != nil {
			ctx.MetadataContext.Set(map[string]any{
				"result": "error",
			})
		}

		// Emit error event for network/connection failures
		errorResponse := map[string]any{
			"error": err.Error(),
		}
		emitErr := ctx.ExecutionStateContext.Emit(
			core.DefaultOutputChannel.Name,
			"http.request.error",
			[]any{errorResponse},
		)
		if emitErr != nil {
			return fmt.Errorf("request failed: %w (and failed to emit event: %v)", err, emitErr)
		}
		return fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		// Set metadata result to "error" for response reading failures
		if ctx.MetadataContext != nil {
			ctx.MetadataContext.Set(map[string]any{
				"result": "error",
			})
		}

		// Emit error event for response reading failures
		errorResponse := map[string]any{
			"status": resp.StatusCode,
			"error":  fmt.Sprintf("failed to read response body: %v", err),
		}
		emitErr := ctx.ExecutionStateContext.Emit(
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
		// Try to parse as JSON, but don't fail if it's not JSON
		err := json.Unmarshal(respBody, &bodyData)
		if err != nil {
			// If not valid JSON, store as string
			bodyData = string(respBody)
		}
	}

	// Build response with status, headers, and body
	response := map[string]any{
		"status":  resp.StatusCode,
		"headers": resp.Header,
		"body":    bodyData,
	}

	// Check if status code matches success codes
	var isSuccess bool
	if spec.SuccessCodes != nil && *spec.SuccessCodes != "" {
		isSuccess = e.matchesSuccessCode(resp.StatusCode, *spec.SuccessCodes)
	} else {
		// Default behavior: 2xx is success
		isSuccess = e.matchesSuccessCode(resp.StatusCode, "2xx")
	}

	// Set metadata result based on success/failure
	var metadataResult string
	if isSuccess {
		metadataResult = "success"
	} else {
		metadataResult = "failed"
	}
	if ctx.MetadataContext != nil {
		ctx.MetadataContext.Set(map[string]any{
			"result": metadataResult,
		})
	}

	// Emit event with response data
	eventType := "http.request.finished"
	if !isSuccess {
		eventType = "http.request.failed"
	}

	err = ctx.ExecutionStateContext.Emit(
		core.DefaultOutputChannel.Name,
		eventType,
		[]any{response},
	)
	if err != nil {
		return err
	}

	// Return error if request failed to mark execution as failed
	if !isSuccess {
		return fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	return nil
}

// matchesSuccessCode checks if the given status code matches any of the success code patterns
func (e *HTTP) matchesSuccessCode(statusCode int, successCodes string) bool {
	// Default to 2xx if not specified
	if successCodes == "" {
		successCodes = "2xx"
	}

	// Split by comma and trim spaces
	codes := strings.Split(successCodes, ",")
	for _, code := range codes {
		code = strings.TrimSpace(code)

		// Handle wildcard patterns like 2xx, 3xx, etc.
		if strings.HasSuffix(code, "xx") {
			prefix := strings.TrimSuffix(code, "xx")
			statusStr := strconv.Itoa(statusCode)
			if strings.HasPrefix(statusStr, prefix) {
				return true
			}
		} else {
			// Handle specific status codes like 200, 201, etc.
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
