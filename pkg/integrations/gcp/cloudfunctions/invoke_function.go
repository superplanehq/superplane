package cloudfunctions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	invokeFunctionPayloadType   = "gcp.cloudfunctions.invoke"
	invokeFunctionOutputChannel = "default"
)

type InvokeFunction struct{}

type InvokeFunctionConfiguration struct {
	Location string `json:"location" mapstructure:"location"`
	Function string `json:"function" mapstructure:"function"`
	Payload  any    `json:"payload" mapstructure:"payload"`
}

type InvokeFunctionMetadata struct {
	FunctionName string `json:"functionName" mapstructure:"functionName"`
	FunctionURI  string `json:"functionUri,omitempty" mapstructure:"functionUri,omitempty"`
	Environment  string `json:"environment,omitempty" mapstructure:"environment,omitempty"`
}

func (c *InvokeFunction) Name() string {
	return "gcp.cloudfunctions.invokeFunction"
}

func (c *InvokeFunction) Label() string {
	return "Cloud Functions • Invoke Function"
}

func (c *InvokeFunction) Description() string {
	return "Invoke a Google Cloud Function and return the response"
}

func (c *InvokeFunction) Documentation() string {
	return `Invokes a Google Cloud Function and waits for the response.

## Configuration

- **Location** (required): The GCP region where the function is deployed (e.g. ` + "`us-central1`" + `).
- **Function** (required): The Cloud Function to invoke. Select from the list of deployed functions.
- **Payload**: Optional JSON object sent as the function's input data.
- **Project ID Override**: Override the GCP project ID from the integration. Leave empty to use the integration's project.

## Required IAM roles

The service account used by the integration must have ` + "`roles/cloudfunctions.developer`" + ` (or ` + "`roles/cloudfunctions.viewer`" + ` + ` + "`roles/cloudfunctions.invoker`" + `) on the project.

- ` + "`roles/cloudfunctions.viewer`" + ` — list locations and functions (required for dropdowns)
- ` + "`roles/cloudfunctions.invoker`" + ` — invoke the function
- ` + "`roles/cloudfunctions.developer`" + ` — covers both of the above

## Output

The invocation result, including:
- ` + "`functionName`" + `: Full resource name of the invoked function.
- ` + "`executionId`" + `: Unique ID assigned to this invocation.
- ` + "`result`" + `: The function's response, parsed as JSON when possible.
- ` + "`resultRaw`" + `: The raw string response (only present when the response is not valid JSON).`
}

func (c *InvokeFunction) Icon() string  { return "gcp" }
func (c *InvokeFunction) Color() string { return "gray" }

func (c *InvokeFunction) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *InvokeFunction) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the GCP region where your Cloud Function is deployed.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeLocation,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "function",
			Label:       "Function",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Cloud Function to invoke.",
			Placeholder: "Select a function",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "location", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeFunction,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
					},
				},
			},
		},
		{
			Name:        "payload",
			Label:       "Payload",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "JSON object sent as input data to the function.",
		},
	}
}

func decodeInvokeFunctionConfiguration(raw any) (InvokeFunctionConfiguration, error) {
	var config InvokeFunctionConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return InvokeFunctionConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.Location = strings.TrimSpace(config.Location)
	config.Function = strings.TrimSpace(config.Function)
	return config, nil
}

func (c *InvokeFunction) Setup(ctx core.SetupContext) error {
	config, err := decodeInvokeFunctionConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.Location == "" {
		return fmt.Errorf("location is required")
	}
	if config.Function == "" {
		return fmt.Errorf("function is required")
	}

	metadata := InvokeFunctionMetadata{FunctionName: config.Function}
	if ctx.Integration != nil {
		client, err := getClient(ctx.HTTP, ctx.Integration)
		if err == nil {
			if details, err := GetFunctionDetails(context.Background(), client, config.Function); err == nil {
				metadata.FunctionURI = details.URI
				metadata.Environment = details.Environment
			}
		}
	}

	return ctx.Metadata.Set(metadata)
}

func (c *InvokeFunction) Execute(ctx core.ExecutionContext) error {
	config, err := decodeInvokeFunctionConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	var metadata InvokeFunctionMetadata
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode metadata: %v", err))
	}

	output := map[string]any{"functionName": config.Function}

	if metadata.Environment == "GEN_2" || metadata.FunctionURI != "" {
		// Gen 2: invoke via HTTP trigger URL, payload sent directly as JSON body.
		responseBody, err := client.PostURL(context.Background(), metadata.FunctionURI, config.Payload)
		if err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to invoke function: %v", err))
		}

		var parsed any
		if json.Unmarshal(responseBody, &parsed) == nil {
			output["result"] = parsed
		} else {
			output["resultRaw"] = string(responseBody)
		}
	} else {
		// Gen 1: use the v1 :call API, payload wrapped as a JSON string in "data".
		payloadBytes, err := json.Marshal(config.Payload)
		if err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to marshal payload: %v", err))
		}

		responseBody, err := client.PostURL(context.Background(), functionCallURL(config.Function), map[string]any{
			"data": string(payloadBytes),
		})
		if err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to invoke function: %v", err))
		}

		var callResp callFunctionResponse
		if err := json.Unmarshal(responseBody, &callResp); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse invocation response: %v", err))
		}

		if callResp.Error != "" {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("function returned error: %s", callResp.Error))
		}

		output["executionId"] = callResp.ExecutionId
		var parsed any
		if json.Unmarshal([]byte(callResp.Result), &parsed) == nil {
			output["result"] = parsed
		} else {
			output["resultRaw"] = callResp.Result
		}
	}

	return ctx.ExecutionState.Emit(invokeFunctionOutputChannel, invokeFunctionPayloadType, []any{output})
}

type callFunctionResponse struct {
	ExecutionId string `json:"executionId"`
	Result      string `json:"result"`
	Error       string `json:"error"`
}

func (c *InvokeFunction) Actions() []core.Action                  { return nil }
func (c *InvokeFunction) HandleAction(_ core.ActionContext) error { return nil }
func (c *InvokeFunction) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *InvokeFunction) Cancel(_ core.ExecutionContext) error { return nil }
func (c *InvokeFunction) Cleanup(_ core.SetupContext) error    { return nil }
func (c *InvokeFunction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
