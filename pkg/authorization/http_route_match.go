package authorization

import (
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func (a *GatewayAuthorizer) RouteFromRequest(r *http.Request) (HTTPRoute, bool) {
	if pattern, ok := runtime.HTTPPathPattern(r.Context()); ok {
		route := HTTPRoute{Method: r.Method, Pattern: pattern}
		_, ok := a.rules[route]
		return route, ok
	}

	return MatchHTTPRoute(r.Method, r.URL.Path, a.rules)
}

func MatchHTTPRoute(method, path string, rules map[HTTPRoute]AuthorizationRule) (HTTPRoute, bool) {
	var (
		best    HTTPRoute
		bestLit = -1
		found   bool
	)

	for route := range rules {
		if route.Method != method {
			continue
		}

		literals, ok := matchPathPattern(route.Pattern, path)
		if !ok || literals <= bestLit {
			continue
		}

		best = route
		bestLit = literals
		found = true
	}

	return best, found
}

func matchPathPattern(pattern, path string) (literalSegments int, ok bool) {
	patternParts := splitPath(pattern)
	pathParts := splitPath(path)
	if len(patternParts) != len(pathParts) {
		return 0, false
	}

	for i, part := range patternParts {
		if isPathParam(part) {
			continue
		}
		if part != pathParts[i] {
			return 0, false
		}
		literalSegments++
	}

	return literalSegments, true
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func isPathParam(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}
