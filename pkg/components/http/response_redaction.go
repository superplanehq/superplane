package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/httpheaders"
)

// redactResponse runs before any response reaches outputs or retry metadata.
// Unstructured bodies and header values are omitted because they cannot be
// safely filtered by header name.
func redactResponse(headers http.Header, body []byte) (http.Header, []byte) {
	headers = headers.Clone()
	for name := range headers {
		if httpheaders.IsSensitive(name) {
			delete(headers, name)
		}
	}

	var payload any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if decoder.Decode(&payload) != nil {
		return headers, nil
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return headers, nil
	}
	switch payload.(type) {
	case map[string]any, []any:
		redactJSONHeaders(payload)
	default:
		return headers, nil
	}
	filtered, err := json.Marshal(payload)
	if err != nil {
		return headers, nil
	}
	return headers, filtered
}

func redactJSONHeaders(value any) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			switch strings.ToLower(strings.TrimSpace(key)) {
			case "headers":
				value[key] = redactHeaderCollection(child)
			case "cookie", "cookies":
				delete(value, key)
			default:
				redactJSONHeaders(child)
			}
		}
	case []any:
		for _, child := range value {
			redactJSONHeaders(child)
		}
	}
}

func redactHeaderCollection(value any) any {
	switch headers := value.(type) {
	case map[string]any:
		if _, record := headers["value"]; record {
			if _, named := headers["name"]; named {
				return redactHeaderEntry(headers)
			}
			if _, keyed := headers["key"]; keyed {
				return redactHeaderEntry(headers)
			}
		}
		filtered := map[string]any{}
		for name, value := range headers {
			if text, ok := value.(string); ok && !httpheaders.IsSensitive(name) {
				filtered[name] = text
			}
		}
		return filtered
	case []any:
		filtered := []any{}
		for _, entry := range headers {
			if safe := redactHeaderEntry(entry); safe != nil {
				filtered = append(filtered, safe)
			}
		}
		return filtered
	default:
		return nil
	}
}

func redactHeaderEntry(entry any) any {
	switch entry := entry.(type) {
	case []any:
		if len(entry) != 2 {
			return nil
		}
		name, nameOK := entry[0].(string)
		value, valueOK := entry[1].(string)
		if nameOK && valueOK && name != "" && !httpheaders.IsSensitive(name) {
			return []any{name, value}
		}
	case map[string]any:
		nameKey := "name"
		name, _ := entry["name"].(string)
		if name == "" {
			nameKey = "key"
			name, _ = entry["key"].(string)
		}
		if value, ok := entry["value"].(string); ok && name != "" && !httpheaders.IsSensitive(name) {
			return map[string]any{nameKey: name, "value": value}
		}
	}
	return nil
}
