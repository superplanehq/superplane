package lambda

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type RunFunction struct{}

type RunFunctionConfiguration struct {
	FunctionArn string `json:"functionArn" mapstructure:"functionArn"`
	Payload     any    `json:"payload" mapstructure:"payload"`
}

type RunFunctionMetadata struct {
	FunctionArn string `json:"functionArn" mapstructure:"functionArn"`
}

func (c *RunFunction) Name() string {
	return "aws.lambda.runFunction"
}

func (c *RunFunction) Label() string {
	return "Lambda • Run Function"
}

func (c *RunFunction) Description() string {
	return "Invoke a Lambda function, optionally creating it from inline JavaScript"
}

func (c *RunFunction) Documentation() string {
	return `
The Run Lambda component invokes a Lambda function.

## Use Cases

- **Automated workflows**: Trigger Lambda functions from SuperPlane workflows
- **Event processing**: Process events from other applications
- **Data transformation**: Transform data in real-time
- **API integrations**: Call Lambda functions from other applications

## How It Works

1. Invokes the specified Lambda function with the provided payload
2. Returns the function's response including status code, payload, and log output
3. Optionally creates a new Lambda function from inline JavaScript code
`
}

func (c *RunFunction) Icon() string {
	return "aws"
}

func (c *RunFunction) Color() string {
	return "orange"
}

func (c *RunFunction) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RunFunction) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "functionArn",
			Label:       "Lambda Function ARN",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "ARN of the Lambda function to invoke",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "lambda.function",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "payload",
			Label:       "Payload",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Payload to send to the Lambda function",
		},
	}
}

func (c *RunFunction) Setup(ctx core.SetupContext) error {
	config := RunFunctionConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	functionArn := strings.TrimSpace(config.FunctionArn)
	if functionArn == "" {
		return fmt.Errorf("Function ARN is required")
	}

	return ctx.Metadata.Set(RunFunctionMetadata{
		FunctionArn: functionArn,
	})
}

func (c *RunFunction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunFunction) Execute(ctx core.ExecutionContext) error {
	config := RunFunctionConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadata := RunFunctionMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return err
	}

	appRegion := common.RegionFromInstallation(ctx.Integration)
	region, err := resolveLambdaRegion(appRegion, metadata.FunctionArn)
	if err != nil {
		return err
	}

	client := NewClient(ctx.HTTP, creds, region)
	payload, err := json.Marshal(config.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	result, err := client.Invoke(metadata.FunctionArn, payload)
	if err != nil {
		return err
	}

	if result.FunctionError != "" {
		return c.handleFunctionError(result)
	}

	output := map[string]any{"requestId": result.RequestID}
	if report, err := parseLambdaLogReport(result.LogResult); err == nil {
		output["report"] = report
	}

	var parsed any
	if len(result.Payload) > 0 && json.Unmarshal(result.Payload, &parsed) == nil {
		output["payload"] = parsed
	} else {
		output["payloadRaw"] = string(result.Payload)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.lambda.run", []any{output})
}

func (c *RunFunction) handleFunctionError(result *InvokeResult) error {
	var errorResponse ErrorResponse
	if err := json.Unmarshal(result.Payload, &errorResponse); err != nil {
		return fmt.Errorf("failed to unmarshal error response: %w", err)
	}

	return fmt.Errorf("Lambda function error: %s: %s", errorResponse.ErrorType, errorResponse.ErrorMessage)
}

func (c *RunFunction) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RunFunction) Actions() []core.Action {
	return []core.Action{}
}

func (c *RunFunction) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *RunFunction) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func resolveLambdaRegion(appRegion string, functionArn string) (string, error) {
	appRegion = strings.TrimSpace(appRegion)
	if appRegion != "" {
		return appRegion, nil
	}

	if strings.TrimSpace(functionArn) != "" {
		region, ok := regionFromArn(functionArn)
		if ok {
			return region, nil
		}
	}

	return "", fmt.Errorf("region is required")
}

func regionFromArn(arn string) (string, bool) {
	parts := strings.Split(arn, ":")
	if len(parts) < 6 || parts[0] != "arn" {
		return "", false
	}
	return parts[3], strings.TrimSpace(parts[3]) != ""
}

type ErrorResponse struct {
	ErrorType    string   `json:"errorType"`
	ErrorMessage string   `json:"errorMessage"`
	Trace        []string `json:"trace"`
}

type LambdaLogReport struct {
	Duration       string `json:"duration"`
	BilledDuration string `json:"billedDuration"`
	MemorySize     string `json:"memorySize"`
	MaxMemoryUsed  string `json:"maxMemoryUsed"`
	InitDuration   string `json:"initDuration"`
}

/*
 * The last line of the log result is a report of the function execution, that looks like this:
 *
 * REPORT RequestId: {REQUEST_ID}	Duration: 89.81 ms	Billed Duration: 251 ms	Memory Size: 128 MB	Max Memory Used: 82 MB	Init Duration: 160.97 ms
 *
 */
func parseLambdaLogReport(logResult string) (*LambdaLogReport, error) {
	if strings.TrimSpace(logResult) == "" {
		return nil, fmt.Errorf("log result is empty")
	}

	decoded, err := base64.StdEncoding.DecodeString(logResult)
	if err != nil {
		return nil, fmt.Errorf("failed to decode log result: %w", err)
	}

	logText := string(decoded)
	lines := strings.Split(logText, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "REPORT ") {
			continue
		}

		report := LambdaLogReport{}
		parts := strings.Split(line, "\t")
		for _, part := range parts {
			token := strings.TrimPrefix(strings.TrimSpace(part), "REPORT ")
			key, value, ok := strings.Cut(token, ": ")
			if !ok {
				continue
			}
			switch key {
			case "Duration":
				if duration, ok := parseLambdaReportValue(value); ok {
					report.Duration = duration
				}
			case "Billed Duration":
				if duration, ok := parseLambdaReportValue(value); ok {
					report.BilledDuration = duration
				}
			case "Memory Size":
				if memory, ok := parseLambdaReportValue(value); ok {
					report.MemorySize = memory
				}
			case "Max Memory Used":
				if memory, ok := parseLambdaReportValue(value); ok {
					report.MaxMemoryUsed = memory
				}
			case "Init Duration":
				if duration, ok := parseLambdaReportValue(value); ok {
					report.InitDuration = duration
				}
			}
		}

		return &report, nil
	}

	return nil, fmt.Errorf("no report found in log result")
}

func parseLambdaReportValue(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func (c *RunFunction) Cleanup(ctx core.SetupContext) error {
	return nil
}
