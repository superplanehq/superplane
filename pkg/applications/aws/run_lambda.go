package aws

import (
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
	lambdaModeUseExisting = "use-existing"
	lambdaModeCreate      = "create"
)

type RunLambda struct{}

type RunLambdaConfiguration struct {
	Mode             string `json:"mode" mapstructure:"mode"`
	LambdaArn        string `json:"lambdaArn" mapstructure:"lambdaArn"`
	FunctionName     string `json:"functionName" mapstructure:"functionName"`
	ExecutionRoleArn string `json:"executionRoleArn" mapstructure:"executionRoleArn"`
	Code             string `json:"code" mapstructure:"code"`
	Runtime          string `json:"runtime" mapstructure:"runtime"`
	Handler          string `json:"handler" mapstructure:"handler"`
	TimeoutSeconds   int    `json:"timeoutSeconds" mapstructure:"timeoutSeconds"`
	MemoryMB         int    `json:"memoryMB" mapstructure:"memoryMB"`
	Payload          string `json:"payload" mapstructure:"payload"`
	InvocationType   string `json:"invocationType" mapstructure:"invocationType"`
	Description      string `json:"description" mapstructure:"description"`
}

type RunLambdaOutput struct {
	StatusCode    int    `json:"statusCode"`
	FunctionError string `json:"functionError,omitempty"`
	LogResult     string `json:"logResult,omitempty"`
	Payload       any    `json:"payload,omitempty"`
	PayloadRaw    string `json:"payloadRaw,omitempty"`
	FunctionArn   string `json:"functionArn,omitempty"`
}

func (c *RunLambda) Name() string {
	return "aws.lambda.runFunction"
}

func (c *RunLambda) Label() string {
	return "Lambda - Run Function"
}

func (c *RunLambda) Description() string {
	return "Invoke a Lambda function, optionally creating it from inline JavaScript"
}

func (c *RunLambda) Icon() string {
	return "aws"
}

func (c *RunLambda) Color() string {
	return "orange"
}

func (c *RunLambda) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RunLambda) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "mode",
			Label:       "Mode",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     lambdaModeUseExisting,
			Description: "Choose whether to use an existing Lambda or create one from code",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Use existing", Value: lambdaModeUseExisting},
						{Label: "Create", Value: lambdaModeCreate},
					},
				},
			},
		},
		{
			Name:        "lambdaArn",
			Label:       "Lambda ARN",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "ARN of the Lambda function to invoke",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{lambdaModeUseExisting}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{lambdaModeUseExisting}},
			},
		},
		{
			Name:        "functionName",
			Label:       "Function Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Name for the new Lambda function",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
		},
		{
			Name:        "executionRoleArn",
			Label:       "Execution Role ARN",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "IAM role ARN for the Lambda execution role",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
		},
		{
			Name:        "code",
			Label:       "JavaScript Code",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "JavaScript handler code for the Lambda function",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
		},
		{
			Name:        "runtime",
			Label:       "Runtime",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "nodejs20.x",
			Description: "Lambda runtime for new functions",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Node.js 20.x", Value: "nodejs20.x"},
						{Label: "Node.js 18.x", Value: "nodejs18.x"},
					},
				},
			},
		},
		{
			Name:        "handler",
			Label:       "Handler",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "index.handler",
			Description: "Handler entrypoint for the Lambda function",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
		},
		{
			Name:        "timeoutSeconds",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "10",
			Description: "Lambda timeout in seconds",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
					Max: func() *int { max := 900; return &max }(),
				},
			},
		},
		{
			Name:        "memoryMB",
			Label:       "Memory (MB)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "128",
			Description: "Lambda memory in MB",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 128; return &min }(),
					Max: func() *int { max := 10240; return &max }(),
				},
			},
		},
		{
			Name:        "payload",
			Label:       "Payload",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Payload to send to the Lambda function",
		},
		{
			Name:        "invocationType",
			Label:       "Invocation Type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "RequestResponse",
			Description: "Invocation type to use for the Lambda call",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Request/Response", Value: "RequestResponse"},
						{Label: "Event", Value: "Event"},
					},
				},
			},
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Description for the new Lambda function",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{lambdaModeCreate}},
			},
		},
	}
}

func (c *RunLambda) ExampleOutput() map[string]any {
	return map[string]any{
		"statusCode": 200,
		"payload": map[string]any{
			"message": "hello from lambda",
		},
	}
}

func (c *RunLambda) Setup(ctx core.SetupContext) error {
	config := RunLambdaConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Mode != lambdaModeUseExisting && config.Mode != lambdaModeCreate {
		return fmt.Errorf("invalid mode %q", config.Mode)
	}

	if config.Mode == lambdaModeUseExisting && strings.TrimSpace(config.LambdaArn) == "" {
		return fmt.Errorf("lambdaArn is required")
	}

	if config.Mode == lambdaModeCreate {
		if strings.TrimSpace(config.FunctionName) == "" {
			return fmt.Errorf("functionName is required")
		}
		if strings.TrimSpace(config.ExecutionRoleArn) == "" {
			return fmt.Errorf("executionRoleArn is required")
		}
		if strings.TrimSpace(config.Code) == "" {
			return fmt.Errorf("code is required")
		}
	}

	return nil
}

func (c *RunLambda) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunLambda) Execute(ctx core.ExecutionContext) error {
	config := RunLambdaConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	creds, err := getSessionCredentials(ctx.AppInstallation)
	if err != nil {
		return err
	}

	appRegion := getRegionFromInstallation(ctx.AppInstallation)
	region, err := resolveLambdaRegion(appRegion, config.LambdaArn, config.Mode)
	if err != nil {
		return err
	}

	client := newLambdaClient(ctx.HTTP, creds, region)

	functionArn := strings.TrimSpace(config.LambdaArn)
	if config.Mode == lambdaModeCreate {
		functionArn, err = client.CreateFunction(createFunctionRequest{
			FunctionName:   strings.TrimSpace(config.FunctionName),
			Runtime:        fallbackString(strings.TrimSpace(config.Runtime), "nodejs20.x"),
			Handler:        fallbackString(strings.TrimSpace(config.Handler), "index.handler"),
			RoleArn:        strings.TrimSpace(config.ExecutionRoleArn),
			Code:           config.Code,
			TimeoutSeconds: fallbackInt(config.TimeoutSeconds, 10),
			MemoryMB:       fallbackInt(config.MemoryMB, 128),
			Description:    strings.TrimSpace(config.Description),
		})
		if err != nil {
			return err
		}
	}

	payload := []byte(strings.TrimSpace(config.Payload))
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	result, err := client.Invoke(functionArn, payload, fallbackString(strings.TrimSpace(config.InvocationType), "RequestResponse"))
	if err != nil {
		return err
	}

	output := RunLambdaOutput{
		StatusCode:    result.StatusCode,
		FunctionError: result.FunctionError,
		LogResult:     result.LogResult,
		FunctionArn:   functionArn,
	}

	var parsed any
	if len(result.Payload) > 0 && json.Unmarshal(result.Payload, &parsed) == nil {
		output.Payload = parsed
	} else {
		output.PayloadRaw = string(result.Payload)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.lambda.run", []any{output})
}

func (c *RunLambda) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RunLambda) Actions() []core.Action {
	return []core.Action{}
}

func (c *RunLambda) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *RunLambda) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func resolveLambdaRegion(appRegion string, functionArn string, mode string) (string, error) {
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

	if mode == lambdaModeUseExisting {
		return "", fmt.Errorf("region is required (set app region or use a regional Lambda ARN)")
	}

	return "", fmt.Errorf("region is required (set app region)")
}

func regionFromArn(arn string) (string, bool) {
	parts := strings.Split(arn, ":")
	if len(parts) < 6 || parts[0] != "arn" {
		return "", false
	}
	return parts[3], strings.TrimSpace(parts[3]) != ""
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func fallbackInt(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
