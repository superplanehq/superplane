package authorization

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAuthorizationRulesCoverAllProtoHTTPRoutes(t *testing.T) {
	protosDir := findProtosDir(t)
	protoRoutes, err := listProtoHTTPRoutes(protosDir)
	require.NoError(t, err)
	require.NotEmpty(t, protoRoutes)

	rules := DefaultAuthorizationRules()
	var missing []string
	for _, route := range protoRoutes {
		if _, ok := rules[route]; !ok {
			missing = append(missing, route.String())
		}
	}

	assert.Empty(t, missing, "gateway auth rules missing for proto HTTP routes:\n%s", strings.Join(missing, "\n"))
}

func findProtosDir(t *testing.T) string {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	dir := cwd
	for {
		candidate := filepath.Join(dir, "protos")
		if isProtoSourceDir(candidate) {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find protos/ directory with .proto sources from %s", cwd)
		}
		dir = parent
	}
}

func isProtoSourceDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".proto") {
			return true
		}
	}

	return false
}

func listProtoHTTPRoutes(protosDir string) ([]HTTPRoute, error) {
	entries, err := os.ReadDir(protosDir)
	if err != nil {
		return nil, err
	}

	optionStart := regexp.MustCompile(`option\s*\(\s*google\.api\.http\s*\)\s*=\s*\{`)
	methodPath := regexp.MustCompile(`(?i)\b(get|post|put|patch|delete)\s*:\s*"([^"]+)"`)

	seen := map[HTTPRoute]struct{}{}
	var routes []HTTPRoute

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".proto") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(protosDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		text := string(content)

		for _, start := range optionStart.FindAllStringIndex(text, -1) {
			block, ok := extractBraceBlock(text, start[1]-1)
			if !ok {
				continue
			}
			for _, match := range methodPath.FindAllStringSubmatch(block, -1) {
				route := HTTPRoute{
					Method:  strings.ToUpper(match[1]),
					Pattern: match[2],
				}
				if _, exists := seen[route]; exists {
					continue
				}
				seen[route] = struct{}{}
				routes = append(routes, route)
			}
		}
	}

	return routes, nil
}

func extractBraceBlock(text string, openIdx int) (string, bool) {
	if openIdx < 0 || openIdx >= len(text) || text[openIdx] != '{' {
		return "", false
	}

	depth := 0
	for i := openIdx; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[openIdx : i+1], true
			}
		}
	}

	return "", false
}
