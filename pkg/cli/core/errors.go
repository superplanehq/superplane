package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/openapi_client"
	"google.golang.org/grpc/codes"
)

const (
	maxErrorBodyBytes      = 2048
	badRequestDetailSuffix = "BadRequest"
)

var grpcCodePrefixes = map[codes.Code]string{
	codes.InvalidArgument:  "invalid request",
	codes.NotFound:         "not found",
	codes.AlreadyExists:    "already exists",
	codes.PermissionDenied: "permission denied",
	codes.Unimplemented:    "not supported",
	codes.Internal:         "internal error",
	codes.Unavailable:      "service unavailable",
	codes.Unauthenticated:  "authentication required",
}

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
		for _, entry := range getViolations(detail.AdditionalProperties) {
			field, _ := entry["field"].(string)
			desc, _ := entry["description"].(string)
			if s := formatFieldDesc(field, desc); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func getViolations(props map[string]any) []map[string]any {
	raw, ok := props["fieldViolations"]
	if !ok {
		raw = props["field_violations"]
	}

	items, ok := raw.([]any)
	if !ok {
		return nil
	}

	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func formatFieldDesc(field, desc string) string {
	switch {
	case field != "" && desc != "":
		return fmt.Sprintf("%s: %s", field, desc)
	case field != "":
		return field
	case desc != "":
		return desc
	default:
		return ""
	}
}

func isBadRequestDetail(typeURL string) bool {
	return strings.HasSuffix(strings.TrimSpace(typeURL), badRequestDetailSuffix)
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
	return grpcCodePrefixes[codes.Code(code)]
}

func usageLimitError(message string) error {
	return fmt.Errorf("%s\nSee current limits with: superplane usage get", message)
}
