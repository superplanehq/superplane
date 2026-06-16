package database

import (
	"reflect"
	"strings"
	"time"
)

const redactedSQLLiteral = "'?'"

func redactSQLParams(params ...interface{}) []interface{} {
	if len(params) == 0 {
		return params
	}

	redacted := make([]interface{}, len(params))
	for i, param := range params {
		redacted[i] = redactSQLParam(param)
	}

	return redacted
}

func redactSQLParam(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch typed := value.(type) {
	case bool:
		return typed
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return typed
	case string:
		return "?"
	case []byte:
		return "?"
	case time.Time:
		return "?"
	case *time.Time:
		if typed == nil {
			return nil
		}
		return "?"
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			return nil
		}
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				return nil
			}
			return redactSQLParam(rv.Elem().Interface())
		}
		if isNumericKind(rv.Kind()) {
			return value
		}
		return "?"
	}
}

func isNumericKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func sanitizeSQLStatement(sql string) string {
	if sql == "" {
		return sql
	}

	var builder strings.Builder
	builder.Grow(len(sql))

	for i := 0; i < len(sql); {
		if sql[i] == '$' {
			if end, ok := readDollarQuotedString(sql, i); ok {
				builder.WriteString(redactedSQLLiteral)
				i = end
				continue
			}
		}

		if (sql[i] == 'E' || sql[i] == 'e') && i+1 < len(sql) && sql[i+1] == '\'' {
			end := readSingleQuotedString(sql, i+1)
			builder.WriteString(redactedSQLLiteral)
			i = end
			continue
		}

		if sql[i] == '\'' {
			end := readSingleQuotedString(sql, i)
			literal := sql[i:end]
			if literal == redactedSQLLiteral {
				builder.WriteString(literal)
			} else {
				builder.WriteString(redactedSQLLiteral)
			}
			i = end
			continue
		}

		builder.WriteByte(sql[i])
		i++
	}

	return builder.String()
}

func readSingleQuotedString(sql string, start int) int {
	if start >= len(sql) || sql[start] != '\'' {
		return start + 1
	}

	i := start + 1
	for i < len(sql) {
		if sql[i] != '\'' {
			i++
			continue
		}
		if i+1 < len(sql) && sql[i+1] == '\'' {
			i += 2
			continue
		}
		return i + 1
	}

	return len(sql)
}

func readDollarQuotedString(sql string, start int) (int, bool) {
	if start >= len(sql) || sql[start] != '$' {
		return start, false
	}

	tagEnd := strings.IndexByte(sql[start+1:], '$')
	if tagEnd < 0 {
		return start, false
	}

	tagEnd += start + 1
	tag := sql[start : tagEnd+1]
	if tag == "$$" {
		bodyEnd := strings.Index(sql[tagEnd+1:], "$$")
		if bodyEnd < 0 {
			return start, false
		}
		return tagEnd + 1 + bodyEnd + 2, true
	}

	bodyEnd := strings.Index(sql[tagEnd+1:], tag)
	if bodyEnd < 0 {
		return start, false
	}

	return tagEnd + 1 + bodyEnd + len(tag), true
}
