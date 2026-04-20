package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	maxErrorBodyBytes       = 2048
	badRequestDetailType    = "type.googleapis.com/google.rpc.BadRequest"
	badRequestDetailTypeAlt = "google.rpc.BadRequest"
)

func FormatCommandError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *openapi_client.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		return err
	}

	rpcStatus := extractGoogleRPCStatus(apiErr)
	if formatted := formatGoogleRPCStatusError(rpcStatus); formatted != nil {
		return appendFieldViolations(formatted, rpcStatus)
	}

	return fallbackAPIError(apiErr)
}

func extractGoogleRPCStatus(apiErr *openapi_client.GenericOpenAPIError) *openapi_client.GooglerpcStatus {
	switch model := apiErr.Model().(type) {
	case *openapi_client.GooglerpcStatus:
		if model != nil && !isEmptyRPCStatus(model) {
			return model
		}
	case openapi_client.GooglerpcStatus:
		if !isEmptyRPCStatus(&model) {
			return &model
		}
	}

	body := apiErr.Body()
	if len(body) == 0 {
		return nil
	}

	var decoded openapi_client.GooglerpcStatus
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil
	}
	if isEmptyRPCStatus(&decoded) {
		return nil
	}
	return &decoded
}

func isEmptyRPCStatus(status *openapi_client.GooglerpcStatus) bool {
	if status == nil {
		return true
	}
	return status.Message == nil && status.Code == nil && len(status.Details) == 0
}

func formatGoogleRPCStatusError(status *openapi_client.GooglerpcStatus) error {
	if status == nil {
		return nil
	}

	message := strings.TrimSpace(status.GetMessage())
	if message == "" {
		return nil
	}

	switch strings.ToLower(message) {
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
	}

	if prefix := grpcCodePrefix(status.GetCode()); prefix != "" {
		return fmt.Errorf("%s: %s", prefix, message)
	}

	return fmt.Errorf("%s", message)
}

func appendFieldViolations(base error, status *openapi_client.GooglerpcStatus) error {
	violations := extractFieldViolations(status)
	if len(violations) == 0 {
		return base
	}

	var sb strings.Builder
	sb.WriteString(base.Error())
	for _, v := range violations {
		sb.WriteString("\n  - ")
		sb.WriteString(v)
	}
	return errors.New(sb.String())
}

func extractFieldViolations(status *openapi_client.GooglerpcStatus) []string {
	if status == nil {
		return nil
	}

	var out []string
	for _, detail := range status.GetDetails() {
		if !isBadRequestDetail(detail.GetType()) {
			continue
		}
		raw, ok := detail.AdditionalProperties["fieldViolations"]
		if !ok {
			raw = detail.AdditionalProperties["field_violations"]
		}
		list, ok := raw.([]any)
		if !ok {
			continue
		}
		for _, item := range list {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			field, _ := entry["field"].(string)
			desc, _ := entry["description"].(string)
			switch {
			case field != "" && desc != "":
				out = append(out, fmt.Sprintf("%s: %s", field, desc))
			case field != "":
				out = append(out, field)
			case desc != "":
				out = append(out, desc)
			}
		}
	}
	return out
}

func isBadRequestDetail(typeURL string) bool {
	trimmed := strings.TrimSpace(typeURL)
	return trimmed == badRequestDetailType || trimmed == badRequestDetailTypeAlt
}

func fallbackAPIError(apiErr *openapi_client.GenericOpenAPIError) error {
	status := strings.TrimSpace(apiErr.Error())
	body := strings.TrimSpace(string(apiErr.Body()))

	if body == "" {
		if status == "" {
			return apiErr
		}
		return errors.New(status)
	}

	if len(body) > maxErrorBodyBytes {
		body = body[:maxErrorBodyBytes] + "... [truncated]"
	}

	if status == "" {
		return errors.New(body)
	}
	return fmt.Errorf("%s\n%s", status, body)
}

func grpcCodePrefix(code int32) string {
	switch code {
	case 3:
		return "invalid request"
	case 5:
		return "not found"
	case 6:
		return "already exists"
	case 7:
		return "permission denied"
	case 12:
		return "not supported"
	case 13:
		return "internal error"
	case 14:
		return "service unavailable"
	case 16:
		return "authentication required"
	default:
		return ""
	}
}

func usageLimitError(message string) error {
	return fmt.Errorf("%s\nSee current limits with: superplane usage get", message)
}
