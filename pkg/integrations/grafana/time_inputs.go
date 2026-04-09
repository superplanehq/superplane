package grafana

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

var grafanaRelativeTimePattern = regexp.MustCompile(`^now(([-+]\d+[a-zA-Z]+)|(/[a-zA-Z]+))*$`)

func resolveGrafanaTimeInput(value string, timezone *string, expressions core.ExpressionContext) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}

	if parsed, ok, err := parseGrafanaQueryTime(trimmed, timezone); err != nil {
		return "", err
	} else if ok {
		return fmt.Sprintf("%d", parsed.UTC().UnixMilli()), nil
	}

	if looksLikeGrafanaRelativeTime(trimmed) {
		return trimmed, nil
	}

	if expressions != nil && looksLikeBareExpression(trimmed) {
		resolved, err := expressions.Run(trimmed)
		if err != nil {
			return "", err
		}

		return normalizeEvaluatedGrafanaTime(resolved, timezone)
	}

	return trimmed, nil
}

func looksLikeGrafanaRelativeTime(value string) bool {
	return grafanaRelativeTimePattern.MatchString(strings.TrimSpace(value))
}

func looksLikeBareExpression(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}

	return strings.ContainsAny(trimmed, "()[]\"'$ ") || strings.Contains(trimmed, ".")
}

func normalizeEvaluatedGrafanaTime(value any, timezone *string) (string, error) {
	switch typed := value.(type) {
	case time.Time:
		return fmt.Sprintf("%d", typed.UTC().UnixMilli()), nil
	case *time.Time:
		if typed == nil {
			return "", nil
		}
		return fmt.Sprintf("%d", typed.UTC().UnixMilli()), nil
	case json.Number:
		return typed.String(), nil
	case int:
		return strconv.Itoa(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case float64:
		return strconv.FormatInt(int64(typed), 10), nil
	case string:
		return resolveGrafanaTimeInput(typed, timezone, nil)
	default:
		return resolveQueryTimeValue(fmt.Sprintf("%v", value), timezone)
	}
}
