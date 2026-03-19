package exprruntime

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

// DateFunctionOption overrides expr's builtin date() with a variant that:
// - defaults to UTC when no timezone is provided
// - accepts timezone arguments as either string, time.Location, or *time.Location
// - tolerates the internal timezone representation used by expr.Timezone(...)
func DateFunctionOption() expr.Option {
	return expr.Function("date", func(params ...any) (any, error) {
		if len(params) != 1 && len(params) != 2 {
			return nil, fmt.Errorf("date() expects 1 or 2 arguments")
		}

		loc := time.UTC
		if len(params) == 2 {
			parsedLoc, err := parseLocation(params[1])
			if err != nil {
				return nil, err
			}
			loc = parsedLoc
		}

		t, ok, err := parseTime(params[0], loc)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}

		return t.In(loc), nil
	})
}

func parseLocation(arg any) (*time.Location, error) {
	switch v := arg.(type) {
	case nil:
		return time.UTC, nil
	case string:
		name := strings.TrimSpace(v)
		if name == "" {
			return time.UTC, nil
		}
		loc, err := time.LoadLocation(name)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %q: %w", name, err)
		}
		return loc, nil
	case *time.Location:
		if v == nil {
			return time.UTC, nil
		}
		return v, nil
	case time.Location:
		loc := v
		return &loc, nil
	default:
		return nil, fmt.Errorf("date() timezone must be a string or time.Location, got %T", arg)
	}
}

func parseTime(value any, loc *time.Location) (time.Time, bool, error) {
	switch v := value.(type) {
	case nil:
		return time.Time{}, false, nil
	case time.Time:
		return v, true, nil
	case *time.Time:
		if v == nil {
			return time.Time{}, false, nil
		}
		return *v, true, nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return time.Time{}, false, fmt.Errorf("date() expects a non-empty string")
		}

		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t, true, nil
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t, true, nil
		}
		if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
			return t, true, nil
		}
		return time.Time{}, false, fmt.Errorf("unsupported date format %q", s)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return fromUnixHeuristic(i, loc), true, nil
		}
		if f, err := v.Float64(); err == nil {
			return fromUnixHeuristic(int64(f), loc), true, nil
		}
		return time.Time{}, false, fmt.Errorf("invalid json.Number %q", v.String())
	case int:
		return fromUnixHeuristic(int64(v), loc), true, nil
	case int32:
		return fromUnixHeuristic(int64(v), loc), true, nil
	case int64:
		return fromUnixHeuristic(v, loc), true, nil
	case float32:
		return fromUnixHeuristic(int64(v), loc), true, nil
	case float64:
		return fromUnixHeuristic(int64(v), loc), true, nil
	default:
		return time.Time{}, false, fmt.Errorf("date() expects a string or time.Time, got %T", value)
	}
}

// fromUnixHeuristic interprets a numeric timestamp as seconds, milliseconds, microseconds, or nanoseconds.
func fromUnixHeuristic(n int64, loc *time.Location) time.Time {
	abs := n
	if abs < 0 {
		abs = -abs
	}

	// Rough thresholds:
	// - seconds: 10 digits (<= 1e10)
	// - millis:  13 digits (<= 1e13)
	// - micros:  16 digits (<= 1e16)
	switch {
	case abs >= 1_000_000_000_000_000: // micros or nanos
		// If it looks like nanos (19 digits), treat as nanoseconds.
		if abs >= 1_000_000_000_000_000_000 {
			return time.Unix(0, n).In(loc)
		}
		return time.Unix(0, n*int64(time.Microsecond)).In(loc)
	case abs >= 1_000_000_000_000: // millis
		return time.Unix(0, n*int64(time.Millisecond)).In(loc)
	default: // seconds
		return time.Unix(n, 0).In(loc)
	}
}
