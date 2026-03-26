package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type InvokeFunctionComponent struct{}

type InvokeFunctionConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	FunctionApp   string `json:"functionApp" mapstructure:"functionApp"`
	FunctionName  string `json:"functionName" mapstructure:"functionName"`
	HTTPMethod    string `json:"httpMethod" mapstructure:"httpMethod"`
	Payload       string `json:"payload" mapstructure:"payload"`
}

func (c *InvokeFunctionComponent) Name() string {
	return "azure.invokeFunction"
}

func (c *InvokeFunctionComponent) Label() string {
	return "Invoke Function"
}

func (c *InvokeFunctionComponent) Description() string {
	return "Invokes an HTTP-triggered Azure Function"
}

func (c *InvokeFunctionComponent) Documentation() string {
	return `
The Invoke Function component calls an HTTP-triggered Azure Function using the
integration's Azure AD access token for authentication.

## Use Cases

- **Trigger serverless workflows**: Kick off Azure Functions from SuperPlane workflows
- **Call custom logic**: Execute business logic hosted in Azure Functions
- **Integrate with services**: Bridge SuperPlane with any system reachable via Azure Functions

## How It Works

1. Resolves the function app name from the resource group
2. Constructs the function URL: https://{functionApp}.azurewebsites.net/api/{functionName}
3. Sends the HTTP request with the Azure AD Bearer token
4. Returns the HTTP status code and response body

## Configuration

- **Resource Group**: The Azure resource group containing the function app
- **Function App**: The function app to invoke
- **Function Name**: The name of the HTTP-triggered function
- **HTTP Method**: The HTTP method to use (GET, POST, PUT, DELETE, PATCH)
- **Payload**: Optional request body (for POST/PUT requests)

## Output

Returns the function response including:
- **statusCode**: The HTTP status code returned by the function
- **body**: The response body (parsed as JSON if possible, otherwise as a string)
- **functionApp**: The function app name
- **functionName**: The function name
`
}

func (c *InvokeFunctionComponent) Icon() string {
	return "azure"
}

func (c *InvokeFunctionComponent) Color() string {
	return "blue"
}

func (c *InvokeFunctionComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"statusCode":   200,
		"body":         map[string]any{"result": "ok"},
		"functionApp":  "my-function-app",
		"functionName": "my-function",
	}
}

func (c *InvokeFunctionComponent) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *InvokeFunctionComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceGroup",
			Label:       "Resource Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Azure resource group containing the function app",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeResourceGroupDropdown,
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "functionApp",
			Label:       "Function App",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Azure Function App to invoke",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeFunctionAppDropdown,
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name: "resourceGroup",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "resourceGroup",
							},
						},
					},
				},
			},
		},
		{
			Name:        "functionName",
			Label:       "Function Name",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The HTTP-triggered function to invoke",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeFunctionDropdown,
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name: "resourceGroup",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "resourceGroup",
							},
						},
						{
							Name: "functionApp",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "functionApp",
							},
						},
					},
				},
			},
		},
		{
			Name:        "httpMethod",
			Label:       "HTTP Method",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "POST",
			Description: "The HTTP method to use when calling the function",
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
			Name:        "payload",
			Label:       "Payload",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Optional request body to send with the function call (for POST/PUT requests)",
			Placeholder: `{{ $["node"].data }}`,
		},
	}
}

func (c *InvokeFunctionComponent) Setup(ctx core.SetupContext) error {
	config := InvokeFunctionConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
	}
	if config.FunctionApp == "" {
		return fmt.Errorf("function app is required")
	}
	if config.FunctionName == "" {
		return fmt.Errorf("function name is required")
	}

	return nil
}

func (c *InvokeFunctionComponent) Execute(ctx core.ExecutionContext) error {
	config := InvokeFunctionConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	functionApp := azureResourceName(config.FunctionApp)
	resourceGroup := azureResourceName(config.ResourceGroup)

	hostname, err := getFunctionAppHostname(context.Background(), provider, resourceGroup, functionApp)
	if err != nil {
		return fmt.Errorf("failed to resolve function app hostname: %w", err)
	}

	functionURL := fmt.Sprintf("https://%s/api/%s", hostname, config.FunctionName)
	method := config.HTTPMethod
	if method == "" {
		method = http.MethodPost
	}

	ctx.Logger.Infof("Invoking Azure Function: %s %s", method, functionURL)
	output, err := invokeFunction(context.Background(), provider.getClient(), functionURL, method, config.Payload, hostname)
	if err != nil {
		return fmt.Errorf("failed to invoke function: %w", err)
	}

	output["functionApp"] = functionApp
	output["functionName"] = config.FunctionName

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"azure.function.invoked",
		[]any{output},
	)
}

func invokeFunction(ctx context.Context, client *armClient, functionURL, method, payload, hostname string) (map[string]any, error) {
	var bodyReader io.Reader
	if payload != "" {
		bodyReader = strings.NewReader(payload)
	}

	token, err := client.bearerTokenForScope(ctx, fmt.Sprintf("https://%s/.default", hostname))
	if err != nil {
		return nil, fmt.Errorf("failed to get function app token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, functionURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var body any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		body = string(bodyBytes)
	}

	return map[string]any{
		"statusCode": resp.StatusCode,
		"body":       body,
	}, nil
}

func (c *InvokeFunctionComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *InvokeFunctionComponent) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *InvokeFunctionComponent) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *InvokeFunctionComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *InvokeFunctionComponent) HandleAction(_ core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *InvokeFunctionComponent) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
