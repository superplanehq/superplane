package authorization

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	/*
	 * The gRPC gateway serves exactly these proto services — see
	 * Server.RegisterGRPCGateway in pkg/public/server.go. They are imported for
	 * the side effect of registering their file descriptors in
	 * protoregistry.GlobalFiles, which gatewayRoutes walks to discover routes.
	 *
	 * When a new service is registered on the gateway, add it here too.
	 * TestEveryAuthorizationRuleMatchesAGatewayRoute fails if a service that owns
	 * authorization rules is missing from this list, so the two stay in sync.
	 */
	_ "github.com/superplanehq/superplane/pkg/protos/actions"
	_ "github.com/superplanehq/superplane/pkg/protos/agents"
	_ "github.com/superplanehq/superplane/pkg/protos/api_keys"
	_ "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	_ "github.com/superplanehq/superplane/pkg/protos/canvases"
	_ "github.com/superplanehq/superplane/pkg/protos/factories"
	_ "github.com/superplanehq/superplane/pkg/protos/groups"
	_ "github.com/superplanehq/superplane/pkg/protos/integrations"
	_ "github.com/superplanehq/superplane/pkg/protos/me"
	_ "github.com/superplanehq/superplane/pkg/protos/organizations"
	_ "github.com/superplanehq/superplane/pkg/protos/roles"
	_ "github.com/superplanehq/superplane/pkg/protos/secrets"
	_ "github.com/superplanehq/superplane/pkg/protos/triggers"
	_ "github.com/superplanehq/superplane/pkg/protos/users"
	_ "github.com/superplanehq/superplane/pkg/protos/widgets"
)

// protoSourceDir is the repository's proto directory, relative to this package.
const protoSourceDir = "../../protos"

/*
 * publicRoutes are the gateway routes that intentionally have no authorization
 * rule. GatewayAuthorizer.AuthorizeHTTP treats an unknown route as public, so a
 * route that is missing from DefaultAuthorizationRules is served without any
 * permission check at all.
 *
 * Adding an entry here is a deliberate security decision: it declares that the
 * route is safe to reach without an organization permission check. Each entry
 * records why. Prefer adding an authorization rule instead.
 */
var publicRoutes = map[HTTPRoute]string{
	{Method: http.MethodGet, Pattern: "/api/v1/me"}:                           "returns only the authenticated caller's own profile",
	{Method: http.MethodPost, Pattern: "/api/v1/me/token"}:                    "regenerates only the authenticated caller's own API token",
	{Method: http.MethodGet, Pattern: "/api/v1/widgets"}:                      "static widget catalog from the in-process registry; identical for every organization",
	{Method: http.MethodGet, Pattern: "/api/v1/widgets/{name}"}:               "static widget catalog from the in-process registry; identical for every organization",
	{Method: http.MethodPost, Pattern: "/api/v1/invite-links/{token}/accept"}: "the invite token is the credential, and the caller is not a member of the organization yet",
}

/*
 * Every route the gateway serves must either be covered by an authorization rule
 * or be listed in publicRoutes.
 *
 * This is the guard around a fail-open default: AuthorizeHTTP allows any route it
 * has no rule for, so an endpoint added to protos/ without a matching entry in
 * DefaultAuthorizationRules silently ships with no RBAC check.
 */
func TestEveryGatewayRouteIsAuthorizedOrExplicitlyPublic(t *testing.T) {
	rules := DefaultAuthorizationRules()
	routes := gatewayRoutes(t)

	var uncovered []string
	for route, rpc := range routes {
		if _, hasRule := rules[route]; hasRule {
			continue
		}
		if _, isPublic := publicRoutes[route]; isPublic {
			continue
		}

		uncovered = append(uncovered, fmt.Sprintf("%s %s (%s)", route.Method, route.Pattern, rpc))
	}

	sort.Strings(uncovered)
	assert.Empty(t, uncovered,
		"these gateway routes have no authorization rule and are not listed as public, so they are served "+
			"without any permission check. Add a rule to DefaultAuthorizationRules, or add the route to "+
			"publicRoutes with a reason:\n  %s", strings.Join(uncovered, "\n  "))
}

/*
 * Every authorization rule must correspond to a route the gateway actually
 * serves. This catches rules left behind after a route is renamed or removed —
 * a stale rule protects nothing — and it doubles as the guard that keeps the
 * blank imports above complete: if a gateway service is not imported, its routes
 * are invisible to gatewayRoutes and all of its rules surface here.
 */
func TestEveryAuthorizationRuleMatchesAGatewayRoute(t *testing.T) {
	routes := gatewayRoutes(t)

	var orphaned []string
	for route := range DefaultAuthorizationRules() {
		if _, served := routes[route]; !served {
			orphaned = append(orphaned, fmt.Sprintf("%s %s", route.Method, route.Pattern))
		}
	}

	sort.Strings(orphaned)
	assert.Empty(t, orphaned,
		"these authorization rules do not match any route served by the gateway. Either the route was "+
			"renamed or removed and the rule is now dead, or the route's proto package is missing from the "+
			"blank imports in this file:\n  %s", strings.Join(orphaned, "\n  "))
}

/*
 * A route listed as public must not also carry an authorization rule. The rule
 * would win at runtime, leaving a misleading "this route is public" claim behind.
 */
func TestPublicRoutesHaveNoAuthorizationRule(t *testing.T) {
	rules := DefaultAuthorizationRules()

	for route, reason := range publicRoutes {
		_, hasRule := rules[route]
		assert.False(t, hasRule,
			"%s %s is listed in publicRoutes (%q) but also has an authorization rule. "+
				"Remove it from publicRoutes — the rule is what applies at runtime.",
			route.Method, route.Pattern, reason)
	}
}

/*
 * Every route listed as public must still be a route the gateway serves, so the
 * allowlist cannot accumulate entries that quietly stop applying.
 */
func TestPublicRoutesMatchAGatewayRoute(t *testing.T) {
	routes := gatewayRoutes(t)

	for route := range publicRoutes {
		_, served := routes[route]
		assert.True(t, served,
			"%s %s is listed in publicRoutes but is not served by the gateway. Remove the stale entry.",
			route.Method, route.Pattern)
	}
}

/*
 * Every proto file that declares HTTP routes must be registered by the blank
 * imports above, so that gatewayRoutes can see its routes.
 *
 * Without this, a newly added service that is missing from the import list would
 * contribute no routes and no rules, and the coverage tests above would pass
 * while every one of its endpoints went unchecked. Reading protos/ from disk
 * makes the proto sources the source of truth rather than the import list.
 */
func TestEveryProtoDeclaringHTTPRoutesIsRegistered(t *testing.T) {
	registered := map[string]bool{}
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		registered[path.Base(file.Path())] = true
		return true
	})

	entries, err := os.ReadDir(protoSourceDir)
	require.NoError(t, err, "unable to read the proto source directory")

	var missing []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".proto") {
			continue
		}

		contents, err := os.ReadFile(filepath.Join(protoSourceDir, entry.Name()))
		require.NoError(t, err)

		if !strings.Contains(string(contents), "google.api.http") {
			continue
		}

		if !registered[entry.Name()] {
			missing = append(missing, entry.Name())
		}
	}

	sort.Strings(missing)
	assert.Empty(t, missing,
		"these proto files declare HTTP routes but are not registered in protoregistry.GlobalFiles, so "+
			"their routes are invisible to the authorization coverage tests. Add the matching package to "+
			"the blank imports in this file:\n  %s", strings.Join(missing, "\n  "))
}

/*
 * gatewayRoutes returns every HTTP route declared by a google.api.http annotation
 * on a registered proto service method, keyed by route and valued by the RPC that
 * serves it.
 *
 * The path templates come straight from the proto annotations, which is what
 * grpc-gateway reports through runtime.HTTPPathPattern at request time and what
 * DefaultAuthorizationRules is keyed by. Reading them here — rather than from the
 * generated OpenAPI spec — keeps the test aligned with the runtime matcher, whose
 * parameter names are the proto field names.
 */
func gatewayRoutes(t *testing.T) map[HTTPRoute]string {
	t.Helper()

	routes := map[HTTPRoute]string{}
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for i := range services.Len() {
			service := services.Get(i)
			methods := service.Methods()
			for j := range methods.Len() {
				method := methods.Get(j)
				for _, route := range httpRoutesForMethod(method) {
					routes[route] = fmt.Sprintf("%s.%s", service.FullName(), method.Name())
				}
			}
		}

		return true
	})

	require.NotEmpty(t, routes,
		"no gateway routes were discovered; the proto packages are expected to register "+
			"their descriptors in protoregistry.GlobalFiles")

	return routes
}

func httpRoutesForMethod(method protoreflect.MethodDescriptor) []HTTPRoute {
	options, ok := method.Options().(*descriptorpb.MethodOptions)
	if !ok || !proto.HasExtension(options, annotations.E_Http) {
		return nil
	}

	rule, ok := proto.GetExtension(options, annotations.E_Http).(*annotations.HttpRule)
	if !ok || rule == nil {
		return nil
	}

	var routes []HTTPRoute
	for _, binding := range append([]*annotations.HttpRule{rule}, rule.GetAdditionalBindings()...) {
		route, ok := httpRouteFromBinding(binding)
		if !ok || !strings.HasPrefix(route.Pattern, "/api/") {
			continue
		}

		routes = append(routes, route)
	}

	return routes
}

func httpRouteFromBinding(binding *annotations.HttpRule) (HTTPRoute, bool) {
	switch pattern := binding.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		return HTTPRoute{Method: http.MethodGet, Pattern: pattern.Get}, true
	case *annotations.HttpRule_Post:
		return HTTPRoute{Method: http.MethodPost, Pattern: pattern.Post}, true
	case *annotations.HttpRule_Put:
		return HTTPRoute{Method: http.MethodPut, Pattern: pattern.Put}, true
	case *annotations.HttpRule_Delete:
		return HTTPRoute{Method: http.MethodDelete, Pattern: pattern.Delete}, true
	case *annotations.HttpRule_Patch:
		return HTTPRoute{Method: http.MethodPatch, Pattern: pattern.Patch}, true
	}

	return HTTPRoute{}, false
}
