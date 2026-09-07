package sentry

import (
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (s *Sentry) afterHostedSetup(ctx core.HTTPRequestContext) {
	app, ok := HostedAppFromEnv()
	if !ok {
		http.Error(ctx.Response, "hosted Sentry app is not configured", http.StatusNotFound)
		return
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil || !metadata.HostedApp {
		http.Error(ctx.Response, "integration not found", http.StatusNotFound)
		return
	}

	query := ctx.Request.URL.Query()
	code := strings.TrimSpace(query.Get("code"))
	installationID := firstNonEmpty(query.Get("installationId"), query.Get("installation_id"))
	orgSlug := firstNonEmpty(query.Get("orgSlug"), query.Get("org"))
	if code == "" || installationID == "" {
		http.Error(ctx.Response, "missing installation parameters", http.StatusBadRequest)
		return
	}

	authorization, err := ExchangeSentryAppAuthorization(ctx.HTTP, app, installationID, code)
	if err != nil {
		ctx.Logger.Errorf("failed to exchange Sentry grant code: %v", err)
		ctx.Integration.Error("failed to finish Sentry install")
		http.Error(ctx.Response, "failed to finish Sentry install", http.StatusBadGateway)
		return
	}

	if err := storeInstallationAuthorization(ctx.Integration, authorization); err != nil {
		ctx.Logger.Errorf("failed to store Sentry installation token: %v", err)
		ctx.Integration.Error("failed to store Sentry installation token")
		http.Error(ctx.Response, "failed to finish Sentry install", http.StatusInternalServerError)
		return
	}

	client := NewAPIClient(ctx.HTTP, app.BaseURL, authorization.Token)
	if err := client.MarkSentryAppInstallationInstalled(installationID); err != nil {
		ctx.Logger.Errorf("failed to verify Sentry installation: %v", err)
		ctx.Integration.Error("failed to verify Sentry installation")
		http.Error(ctx.Response, "failed to finish Sentry install", http.StatusBadGateway)
		return
	}

	metadata.InstallationID = installationID
	metadata.State = ""
	if orgSlug == "" {
		organizations, err := client.ListOrganizations()
		if err != nil {
			ctx.Logger.Errorf("failed to list Sentry organizations: %v", err)
			ctx.Integration.Error("failed to load Sentry organization")
			http.Error(ctx.Response, "failed to finish Sentry install", http.StatusBadGateway)
			return
		}
		if len(organizations) != 1 {
			ctx.Integration.Error("failed to identify Sentry organization")
			http.Error(ctx.Response, "failed to finish Sentry install", http.StatusBadGateway)
			return
		}
		orgSlug = organizations[0].Slug
	}

	client.orgSlug = orgSlug
	ctx.Integration.SetMetadata(metadata)
	if err := s.populateMetadataFromOrg(core.SyncContext{
		Logger:      ctx.Logger,
		HTTP:        ctx.HTTP,
		Integration: ctx.Integration,
	}, client); err != nil {
		ctx.Logger.Errorf("failed to load Sentry organization: %v", err)
		ctx.Integration.Error("failed to load Sentry organization")
		http.Error(ctx.Response, "failed to finish Sentry install", http.StatusBadGateway)
		return
	}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Integration.Error("failed to finish Sentry install")
		http.Error(ctx.Response, "failed to finish Sentry install", http.StatusInternalServerError)
		return
	}

	metadata.HostedApp = true
	metadata.InstallationID = installationID
	metadata.State = ""
	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	clearSentrySetupCookie(ctx.Response)
	redirectToIntegrationSettings(ctx)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
