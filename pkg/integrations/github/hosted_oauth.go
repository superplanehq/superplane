package github

import (
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

// allowsRebind reports whether a bound hosted connection can move to another
// installation: the CSRF state survived the first bind and the request
// carries it.
func allowsRebind(metadata common.Metadata, state string) bool {
	return metadata.HostedApp && metadata.State != "" && state == metadata.State
}

func (g *GitHub) afterHostedAppBind(ctx core.HTTPRequestContext) {
	metadata, ok := decodeHostedMetadata(ctx)
	if !ok {
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	state := ctx.Request.URL.Query().Get("state")
	installationID := ctx.Request.URL.Query().Get("installation_id")

	// A bound connection accepts a rebind with a valid state, so the
	// onboarding account picker can move it to another account. A request
	// without that state is a stale callback and goes back to settings.
	if metadata.InstallationID != "" && !allowsRebind(metadata, state) {
		redirectToIntegrationSettings(ctx)
		return
	}

	if state == "" || state != metadata.State || installationID == "" {
		http.Error(ctx.Response, "invalid installation ID or state", http.StatusBadRequest)
		return
	}

	if !metadata.AllowsPendingInstallation(installationID) {
		http.Error(ctx.Response, "installation is not allowed", http.StatusBadRequest)
		return
	}

	if err := g.ensureHostedBindAllowed(ctx, metadata, installationID); err != nil {
		ctx.Logger.Errorf("%v", err)
		http.Error(ctx.Response, "installation is not allowed", http.StatusBadRequest)
		return
	}

	if err := g.bindHostedInstallation(ctx, metadata, installationID); err != nil {
		ctx.Logger.Errorf("%v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	redirectToIntegrationSettings(ctx)
}

func decodeHostedMetadata(ctx core.HTTPRequestContext) (common.Metadata, bool) {
	metadata := common.Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("failed to decode metadata: %v", err)
		return metadata, false
	}
	return metadata, true
}
