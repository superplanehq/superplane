package provider

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
)

// FormatRunnerTags encodes tags for the runner's RUNNER_TAGS environment variable.
func FormatRunnerTags(tags map[string]string) (string, error) {
	if len(tags) == 0 {
		return "", nil
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		if strings.TrimSpace(key) == "" || strings.Contains(key, "=") {
			return "", fmt.Errorf("runner tag key %q is invalid", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]string, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, key+"="+tags[key])
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write(fields); err != nil {
		return "", err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return strings.TrimSuffix(output.String(), "\n"), nil
}
