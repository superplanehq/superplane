package prometheus

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
)

type QueryOptionsConfiguration struct {
	Timeout                             string `json:"timeout" mapstructure:"timeout"`
	MaxSamplesProcessedWarningThreshold int    `json:"maxSamplesProcessedWarningThreshold" mapstructure:"maxSamplesProcessedWarningThreshold"`
	MaxSamplesProcessedErrorThreshold   int    `json:"maxSamplesProcessedErrorThreshold" mapstructure:"maxSamplesProcessedErrorThreshold"`
}

func queryField() configuration.Field {
	return configuration.Field{
		Name:        "query",
		Label:       "Query",
		Type:        configuration.FieldTypeString,
		Required:    true,
		Placeholder: "up",
		Description: "PromQL expression to evaluate",
	}
}

func queryOptionFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "timeout",
			Label:       "Timeout",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Optional query timeout duration",
		},
		{
			Name:        "maxSamplesProcessedWarningThreshold",
			Label:       "Max Samples Warning Threshold",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Optional warning threshold for query samples processed",
		},
		{
			Name:        "maxSamplesProcessedErrorThreshold",
			Label:       "Max Samples Error Threshold",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Optional error threshold for query samples processed",
		},
	}
}

func validateQueryOptions(config QueryOptionsConfiguration) error {
	if config.MaxSamplesProcessedWarningThreshold < 0 {
		return fmt.Errorf("max samples warning threshold cannot be negative")
	}
	if config.MaxSamplesProcessedErrorThreshold < 0 {
		return fmt.Errorf("max samples error threshold cannot be negative")
	}

	return nil
}

func queryOutput(response map[string]any) map[string]any {
	data, ok := response["data"].(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return map[string]any{
		"resultType": data["resultType"],
		"result":     data["result"],
	}
}
