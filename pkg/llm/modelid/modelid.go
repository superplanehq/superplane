package modelid

import "strings"

const openRouterGatewayPrefix = "openrouter/"
const customProviderPrefix = "custom/"

// CatalogModelID trims OpenRouter gateway prefixes and keeps the catalog id.
// It does not strip `openrouter/free`, because that string is itself a catalog id.
// A custom provider prefix is one segment, so `custom/kimi-k3` becomes `kimi-k3`.
func CatalogModelID(model string) string {
	normalized := strings.TrimSpace(model)
	if strings.HasPrefix(strings.ToLower(normalized), customProviderPrefix) {
		rest := strings.TrimSpace(normalized[len(customProviderPrefix):])
		if rest != "" {
			normalized = rest
		}
	}
	for {
		lower := strings.ToLower(normalized)
		if !strings.HasPrefix(lower, openRouterGatewayPrefix) {
			return normalized
		}
		rest := normalized[len(openRouterGatewayPrefix):]
		if rest == "" || !strings.Contains(rest, "/") {
			return normalized
		}
		normalized = rest
	}
}
