package sentry

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const integrationSetupReturnCookie = "sp_integration_setup_return"

type persistentIntegration interface {
	Persist() error
}

func persistIntegrationBeforeRedirect(ctx core.HTTPRequestContext) {
	persister, ok := ctx.Integration.(persistentIntegration)
	if !ok {
		return
	}

	if err := persister.Persist(); err != nil {
		ctx.Logger.Errorf("failed to persist Sentry integration before redirect: %v", err)
	}
}

func redirectToIntegrationSettings(ctx core.HTTPRequestContext) {
	persistIntegrationBeforeRedirect(ctx)
	location := integrationCallbackLocation(ctx)
	if integrationCallbackReturnPath(ctx) != "" {
		clearIntegrationSetupReturnCookie(ctx.Response)
	}
	http.Redirect(ctx.Response, ctx.Request, location, http.StatusSeeOther)
}

func integrationCallbackLocation(ctx core.HTTPRequestContext) string {
	settings := fmt.Sprintf(
		"%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID().String(),
	)
	returnPath := integrationCallbackReturnPath(ctx)
	if returnPath == "" {
		return settings
	}

	return strings.TrimRight(ctx.BaseURL, "/") + returnPath
}

func integrationCallbackReturnPath(ctx core.HTTPRequestContext) string {
	if path := setupReturnPathFromMetadata(ctx); path != "" {
		return path
	}
	if path := integrationSetupReturnPath(ctx.Request); path != "" {
		return path
	}

	return ""
}

func setupReturnPathFromMetadata(ctx core.HTTPRequestContext) string {
	if ctx.Integration == nil {
		return ""
	}

	metadata := Metadata{}
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	return firstSafeSetupReturnPath(metadata.SetupReturnPath)
}

func firstSafeSetupReturnPath(paths ...string) string {
	for _, path := range paths {
		if isSafeIntegrationSetupReturnPath(path) {
			return path
		}
	}

	return ""
}

func integrationSetupReturnPath(request *http.Request) string {
	if request == nil {
		return ""
	}

	cookie, err := request.Cookie(integrationSetupReturnCookie)
	if err != nil || cookie == nil {
		return ""
	}

	path, err := url.QueryUnescape(strings.TrimSpace(cookie.Value))
	if err != nil || !isSafeIntegrationSetupReturnPath(path) {
		return ""
	}

	return path
}

func isSafeIntegrationSetupReturnPath(path string) bool {
	if path == "" || strings.Contains(path, "://") || strings.ContainsAny(path, "\\\t\r\n ") {
		return false
	}

	pathname, _, _ := strings.Cut(path, "?")
	if pathname == "/onboarding" {
		return true
	}
	if !strings.HasPrefix(pathname, "/") || strings.HasPrefix(pathname, "//") {
		return false
	}

	rest := strings.TrimPrefix(pathname, "/")
	organization, after, ok := strings.Cut(rest, "/")
	return ok && organization != "" && after != ""
}

func clearIntegrationSetupReturnCookie(response http.ResponseWriter) {
	if response == nil {
		return
	}

	http.SetCookie(response, &http.Cookie{
		Name:   integrationSetupReturnCookie,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

func clearSentrySetupCookie(response http.ResponseWriter) {
	if response == nil {
		return
	}

	http.SetCookie(response, &http.Cookie{
		Name:   SetupCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}
