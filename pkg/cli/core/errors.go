package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func FormatCommandError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr openapi_client.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		return err
	}

	switch model := apiErr.Model().(type) {
	case *openapi_client.GooglerpcStatus:
		if formatted := formatGoogleRPCStatusError(model); formatted != nil {
			return formatted
		}
	case openapi_client.GooglerpcStatus:
		if formatted := formatGoogleRPCStatusError(&model); formatted != nil {
			return formatted
		}
	}

	return err
}

func formatGoogleRPCStatusError(status *openapi_client.GooglerpcStatus) error {
	if status == nil {
		return nil
	}

	message := strings.TrimSpace(strings.ToLower(status.GetMessage()))
	if message == "" {
		return nil
	}

	switch message {
	case "account organization limit exceeded":
		return usageLimitError("usage limit reached: this account has reached its organization limit")
	case "organization canvas limit exceeded":
		return usageLimitError("usage limit reached: this organization has reached its canvas limit")
	case "canvas node limit exceeded":
		return usageLimitError("usage limit reached: this canvas exceeds the plan node limit")
	case "organization user limit exceeded":
		return usageLimitError("usage limit reached: this organization has reached its member limit")
	case "organization integration limit exceeded":
		return usageLimitError("usage limit reached: this organization has reached its integration limit")
	case "organization exceeds configured account usage limits":
		return usageLimitError("usage limit reached: this organization is blocked by account-level usage limits")
	default:
		return nil
	}
}

func usageLimitError(message string) error {
	return fmt.Errorf("%s\nSee current limits with: superplane usage get", message)
}
