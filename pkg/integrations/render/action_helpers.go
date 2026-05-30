package render

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
)

func decodeActionConfiguration(input any, output any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           output,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create configuration decoder: %w", err)
	}

	if err := decoder.Decode(input); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return nil
}

func cleanStringList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}

	return result
}
