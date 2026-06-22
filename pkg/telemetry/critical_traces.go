package telemetry

import (
	"net/http"
	"strings"

	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	pbMe "github.com/superplanehq/superplane/pkg/protos/me"
	pbOrganizations "github.com/superplanehq/superplane/pkg/protos/organizations"
)

var criticalHTTPRoutes = map[string]struct{}{
	"/api/v1/me":                                   {},
	"/api/v1/organizations/{id}":                   {},
	"/api/v1/organizations/{id}/usage":             {},
	"/api/v1/canvases/{canvas_id}":                 {},
	"/api/v1/canvases/{canvas_id}/versions":        {},
	"/api/v1/canvases/{canvas_id}/runs":            {},
	"/api/v1/canvases/{canvas_id}/repository/file": {},
	"/api/v1/canvases/{canvas_id}/memory":          {},
}

var criticalHTTPHandlers = map[string]struct{}{
	"GET /organizations": {},
}

var criticalGRPCMethods = map[string]struct{}{
	pbMe.Me_Me_FullMethodName: {},
	pbOrganizations.Organizations_DescribeOrganization_FullMethodName: {},
	pbOrganizations.Organizations_DescribeUsage_FullMethodName:        {},
	pbCanvases.Canvases_DescribeCanvas_FullMethodName:                 {},
	pbCanvases.Canvases_ListCanvasVersions_FullMethodName:             {},
	pbCanvases.Canvases_ListRuns_FullMethodName:                       {},
	pbCanvases.Canvases_DescribeRun_FullMethodName:                    {},
	pbCanvases.Canvases_ListCanvasMemories_FullMethodName:             {},
}

func IsCriticalHTTPRoute(route string) bool {
	if route == "" {
		return false
	}

	_, ok := criticalHTTPRoutes[route]
	return ok
}

func IsCriticalHTTPHandler(method, path string) bool {
	_, ok := criticalHTTPHandlers[method+" "+path]
	return ok
}

func IsCriticalGRPCMethod(fullMethod string) bool {
	_, ok := criticalGRPCMethods[fullMethod]
	return ok
}

func MayTraceHTTPRequest(r *http.Request) bool {
	if IsCriticalHTTPHandler(r.Method, r.URL.Path) {
		return true
	}

	path := r.URL.Path
	if path == "/api/v1/me" {
		return true
	}

	if strings.HasPrefix(path, "/api/v1/organizations/") {
		rest := strings.TrimPrefix(path, "/api/v1/organizations/")
		parts := strings.Split(rest, "/")
		if len(parts) == 1 && parts[0] != "" {
			return true
		}
		if len(parts) == 2 && parts[1] == "usage" {
			return true
		}

		return false
	}

	if !strings.HasPrefix(path, "/api/v1/canvases/") {
		return false
	}

	rest := strings.TrimPrefix(path, "/api/v1/canvases/")
	parts := strings.Split(rest, "/")
	if len(parts) == 1 && parts[0] != "" {
		return true
	}

	if len(parts) < 2 {
		return false
	}

	switch parts[1] {
	case "runs", "events", "versions", "memory":
		return true
	case "repository":
		return len(parts) >= 3 && parts[2] == "file"
	default:
		return false
	}
}
