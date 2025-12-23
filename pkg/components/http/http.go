package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

type Spec struct {
	Method      string   `json:"method"`
	URL         string   `json:"url"`
	SendHeaders bool     `json:"sendHeaders"`
	Headers     []Header `json:"headers"`
	SendBody    bool     `json:"sendBody"`
	ContentType string   `json:"contentType"`
	Payload     any      `json:"payload"`
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
			Name:        "sendHeaders",
			Label:       "Send Headers",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Enable to send custom headers with this request",
		},
		{
			Name:     "headers",
			Label:    "Headers",
			Type:     configuration.FieldTypeList,
			Required: false,
			Default: []map[string]any{
				{"name": "", "value": ""},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sendHeaders", Values: []string{"true"}},
			},
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
		},
		{
			Name:        "sendBody",
			Label:       "Send Body",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Enable to send a request body with this request",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
			},
		},
		{
			Name:        "contentType",
			Label:       "Content Type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "application/json",
			Description: "The content type of the request body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
				{Field: "sendBody", Values: []string{"true"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sendBody", Values: []string{"true"}},
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
			Name:        "payload",
			Type:        configuration.FieldTypeObject,
			Label:       "JSON Payload",
			Required:    false,
			Description: "The JSON object to send as the request body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
				{Field: "sendBody", Values: []string{"true"}},
				{Field: "contentType", Values: []string{"application/json"}},
			},
		},
		{
			Name:     "payloadFormData",
			Label:    "Form Data",
			Type:     configuration.FieldTypeList,
			Required: false,
			Default: []map[string]any{
				{"key": "", "value": ""},
			},
			Description: "Key-value pairs to send as form data",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
				{Field: "sendBody", Values: []string{"true"}},
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
			Name:        "payloadText",
			Type:        configuration.FieldTypeString,
			Label:       "Text Payload",
			Required:    false,
			Description: "Plain text to send as the request body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
				{Field: "sendBody", Values: []string{"true"}},
				{Field: "contentType", Values: []string{"text/plain"}},
			},
			Placeholder: "Enter plain text content",
		},
		{
			Name:        "payloadXML",
			Type:        configuration.FieldTypeString,
			Label:       "XML Payload",
			Required:    false,
			Description: "XML content to send as the request body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
				{Field: "sendBody", Values: []string{"true"}},
				{Field: "contentType", Values: []string{"application/xml"}},
			},
			Placeholder: "<?xml version=\"1.0\"?>\n<root>\n  <element>value</element>\n</root>",
		},
	}
}

func (e *HTTP) Actions() []core.Action {
	return []core.Action{}
}

func (e *HTTP) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("http does not support actions")
}

func (e *HTTP) ProcessQueueItem(ctx core.ProcessQueueContext) (*models.WorkflowNodeExecution, error) {
	return ctx.DefaultProcessing()
}

func (e *HTTP) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	// Backward compatibility: handle old "payload" field for POST/PUT/PATCH
	if !spec.SendBody && spec.Payload != nil &&
		(spec.Method == "POST" || spec.Method == "PUT" || spec.Method == "PATCH") {
		spec.SendBody = true
		if spec.ContentType == "" {
			spec.ContentType = "application/json"
		}
	}

	// Determine which payload field to use based on content type
	var payload any
	if spec.SendBody {
		switch spec.ContentType {
		case "application/json", "":
			payload = ctx.Configuration.(map[string]any)["payload"]
		case "application/x-www-form-urlencoded":
			payload = ctx.Configuration.(map[string]any)["payloadFormData"]
		case "text/plain":
			payload = ctx.Configuration.(map[string]any)["payloadText"]
		case "application/xml":
			payload = ctx.Configuration.(map[string]any)["payloadXML"]
		}
	}

	// Serialize payload based on content type
	var body io.Reader
	var contentType string
	if spec.SendBody && payload != nil &&
		(spec.Method == "POST" || spec.Method == "PUT" || spec.Method == "PATCH") {
		body, contentType, err = e.serializePayload(spec.ContentType, payload)
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

	// Apply custom headers if enabled (can override Content-Type)
	if spec.SendHeaders {
		for _, header := range spec.Headers {
			req.Header.Set(header.Name, header.Value)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
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

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		core.DefaultOutputChannel.Name: {response},
	})
}

func (e *HTTP) serializePayload(contentType string, payload any) (io.Reader, string, error) {
	if contentType == "" {
		contentType = "application/json"
	}

	switch contentType {
	case "application/json":
		if payload == nil {
			return nil, contentType, nil
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal JSON payload: %w", err)
		}
		return bytes.NewReader(data), contentType, nil

	case "application/x-www-form-urlencoded":
		formParams, ok := payload.([]any)
		if !ok || len(formParams) == 0 {
			return nil, contentType, nil
		}
		values := url.Values{}
		for _, item := range formParams {
			param, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key, keyOk := param["key"].(string)
			value, valueOk := param["value"].(string)
			if keyOk && valueOk {
				values.Add(key, value)
			}
		}
		return strings.NewReader(values.Encode()), contentType, nil

	case "text/plain", "application/xml":
		text, ok := payload.(string)
		if !ok || text == "" {
			return nil, contentType, nil
		}
		return strings.NewReader(text), contentType, nil

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
