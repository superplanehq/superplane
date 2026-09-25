package github

import (
	"net/http"
	"slices"
	"strconv"

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

	if err := ctx.Request.ParseForm(); err != nil {
		http.Error(ctx.Response, "invalid request", http.StatusBadRequest)
		return
	}
	state := ctx.Request.FormValue("state")
	installationID := ctx.Request.FormValue("installation_id")
	repositoryValues := ctx.Request.Form["repository_id"]

	// A bound connection accepts a rebind with a valid state, so the
	// onboarding account picker can move it to another account.
	if metadata.InstallationID != "" && !allowsRebind(metadata, state) {
		http.Error(ctx.Response, "invalid state", http.StatusBadRequest)
		return
	}

	if state == "" || state != metadata.State || installationID == "" || len(repositoryValues) == 0 {
		http.Error(ctx.Response, "invalid installation, repository, or state", http.StatusBadRequest)
		return
	}

	repositoryIDs := make([]int64, 0, len(repositoryValues))
	for _, value := range repositoryValues {
		repositoryID, err := strconv.ParseInt(value, 10, 64)
		if err != nil || repositoryID <= 0 {
			http.Error(ctx.Response, "invalid repository", http.StatusBadRequest)
			return
		}
		repositoryIDs = append(repositoryIDs, repositoryID)
	}

	identity, err := findStartedByGitHubIdentity(ctx.OrganizationID, metadata.StartedByUserID)
	if err != nil {
		http.Error(ctx.Response, "GitHub identity is required", http.StatusForbidden)
		return
	}
	app, ok := common.HostedAppFromEnv()
	if !ok {
		http.Error(ctx.Response, "hosted GitHub App is not configured", http.StatusServiceUnavailable)
		return
	}
	installations, err := discoverAccessibleInstallations(ctx.Request.Context(), ctx.Integration, app, *identity)
	if err != nil {
		ctx.Logger.Errorf("failed to verify GitHub repository access: %v", err)
		http.Error(ctx.Response, "failed to verify GitHub repository access", http.StatusBadGateway)
		return
	}
	metadata.SetPendingInstallations(installations)
	repositories, allowed := metadata.SelectPendingRepositories(installationID, repositoryIDs)
	if !allowed {
		http.Error(ctx.Response, "repository is not allowed", http.StatusForbidden)
		return
	}

	installation, found := pendingInstallationByID(installations, installationID)
	if !found {
		http.Error(ctx.Response, "installation is not allowed", http.StatusForbidden)
		return
	}
	if err := g.bindHostedInstallationRepositories(ctx, metadata, installation, repositories); err != nil {
		ctx.Logger.Errorf("%v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	ctx.Response.WriteHeader(http.StatusNoContent)
}

func pendingInstallationByID(installations []common.PendingInstallation, installationID string) (common.PendingInstallation, bool) {
	index := slices.IndexFunc(installations, func(installation common.PendingInstallation) bool {
		return installation.ID == installationID
	})
	if index == -1 {
		return common.PendingInstallation{}, false
	}
	return installations[index], true
}

func decodeHostedMetadata(ctx core.HTTPRequestContext) (common.Metadata, bool) {
	metadata := common.Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("failed to decode metadata: %v", err)
		return metadata, false
	}
	return metadata, true
}
