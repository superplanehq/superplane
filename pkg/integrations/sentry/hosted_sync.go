package sentry

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

func (s *Sentry) syncHostedApp(ctx core.SyncContext) error {
	if !HostedAppConfigured() {
		return fmt.Errorf("hosted Sentry app is not configured")
	}

	var existing Metadata
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &existing)

	if existing.InstallationID != "" {
		return s.refreshHostedMetadata(ctx)
	}

	returnPath := firstSafeSetupReturnPath(hostedSetupReturnPath(ctx), existing.SetupReturnPath)
	if existing.HostedApp && existing.State != "" {
		existing.SetupReturnPath = returnPath
		s.refreshHostedPendingAction(ctx, existing)
		return nil
	}

	state, err := crypto.Base64String(32)
	if err != nil {
		return fmt.Errorf("failed to generate Sentry setup state: %w", err)
	}

	startedBy := ctx.ActorUserID
	if existing.StartedByUserID != "" {
		startedBy = existing.StartedByUserID
	}

	s.refreshHostedPendingAction(ctx, Metadata{
		State:           state,
		HostedApp:       true,
		StartedByUserID: startedBy,
		SetupReturnPath: returnPath,
	})
	return nil
}

func (s *Sentry) refreshHostedPendingAction(ctx core.SyncContext, metadata Metadata) {
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: hostedInstallDescription,
		URL:         HostedAppStartURL(ctx.BaseURL, ctx.Integration.ID().String()),
		Method:      http.MethodGet,
	})
	ctx.Integration.SetMetadata(metadata)
}

func (s *Sentry) refreshHostedMetadata(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	if err := s.populateMetadataFromOrg(ctx, client); err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

func hostedSetupReturnPath(ctx core.SyncContext) string {
	if ctx.Configuration == nil {
		return ""
	}

	config := Configuration{}
	_ = mapstructure.Decode(ctx.Configuration, &config)
	return strings.TrimSpace(config.SetupReturnPath)
}
