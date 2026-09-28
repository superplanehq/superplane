package github

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	hostedInitialDiscoveryPageSize = 8
	hostedInitialDiscoveryTimeout  = 8 * time.Second
)

func (g *GitHub) refreshHostedInitialInstallations(
	ctx core.SyncContext,
	app common.HostedApp,
	metadata *common.Metadata,
) error {
	if !supportsHostedInitialDiscovery(metadata.SetupReturnPath) {
		completeHostedInitialDiscovery(metadata)
		return nil
	}
	if metadata.InstallationID != "" ||
		strings.TrimSpace(metadata.StartedByUserID) == "" ||
		strings.TrimSpace(metadata.StartedByGitHubLogin) == "" {
		return nil
	}
	ensureHostedInitialDiscovery(metadata)
	if !hostedInitialDiscoveryActive(*metadata) {
		return nil
	}

	requestContext := ctx.Context
	if requestContext == nil {
		requestContext = context.Background()
	}
	requestContext, cancel := context.WithTimeout(requestContext, hostedInitialDiscoveryTimeout)
	defer cancel()

	identity, err := hostedGitHubDiscoveryIdentity(requestContext, ctx.OrganizationID, metadata.StartedByUserID)
	if err != nil {
		return nil
	}
	metadata.StartedByGitHubLogin = identity.Login
	startedAt := time.Now().UTC()

	if !metadata.InstallationDiscovery.PersonalAccountChecked {
		return g.refreshHostedPersonalInstallation(requestContext, ctx, app, *identity, metadata, startedAt)
	}
	if len(metadata.InstallationDiscovery.RetryCandidates) > 0 {
		return g.retryHostedInitialInstallations(requestContext, ctx, app, *identity, metadata, startedAt)
	}
	if metadata.InstallationDiscovery.NextPage > 0 {
		return g.refreshHostedInitialInstallationPage(requestContext, ctx, app, *identity, metadata, startedAt)
	}

	return finishHostedInitialDiscovery(metadata)
}

func (g *GitHub) refreshHostedPersonalInstallation(
	requestContext context.Context,
	ctx core.SyncContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
	metadata *common.Metadata,
	startedAt time.Time,
) error {
	client, err := newAppJWTClient(ctx.Integration, app.ID)
	if err != nil {
		return fmt.Errorf("create GitHub App client: %w", err)
	}

	installation, err := findAppUserInstallation(requestContext, client, identity.Login)
	if err != nil {
		if githubErrorIsNotFound(err) {
			metadata.InstallationDiscovery.PersonalAccountChecked = true
			logHostedDiscovery(ctx, "initial_personal", 0, 0, nil, startedAt)
			return nil
		}
		logHostedDiscovery(ctx, "initial_personal", 1, 0, err, startedAt)
		if hostedDiscoveryErrorIsRetryable(err) {
			metadata.InstallationDiscovery.PersonalAccountChecked = true
			metadata.InstallationDiscovery.TransientFailures = true
			return nil
		}
		return fmt.Errorf("find personal GitHub App installation: %w", err)
	}

	candidate, err := pendingInstallationFromGitHub(installation)
	metadata.InstallationDiscovery.PersonalAccountChecked = true
	if err != nil {
		logHostedDiscovery(ctx, "initial_personal", 1, 0, err, startedAt)
		return nil
	}
	verified, retries, verificationErr := verifyAccessibleInstallationsWithFailures(
		requestContext,
		ctx.Integration,
		app,
		identity,
		[]common.PendingInstallation{candidate},
		nil,
	)
	metadata.SetPendingInstallations(mergeVerifiedInstallations(verified, metadata.PendingInstallations))
	metadata.InstallationDiscovery.RetryCandidates = uniqueDiscoveryCandidates(retries)
	logHostedDiscovery(ctx, "initial_personal", 1, len(verified), verificationErr, startedAt)
	return nil
}

func (g *GitHub) refreshHostedInitialInstallationPage(
	requestContext context.Context,
	ctx core.SyncContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
	metadata *common.Metadata,
	startedAt time.Time,
) error {
	client, err := newAppJWTClient(ctx.Integration, app.ID)
	if err != nil {
		return fmt.Errorf("create GitHub App client: %w", err)
	}
	page := metadata.InstallationDiscovery.NextPage
	candidates, nextPage, err := listRecentAppInstallations(
		requestContext,
		client,
		time.Time{},
		page,
		hostedInitialDiscoveryPageSize,
	)
	if err != nil {
		logHostedDiscovery(ctx, "initial_page", hostedInitialDiscoveryPageSize, 0, err, startedAt)
		if hostedDiscoveryErrorIsRetryable(err) {
			return nil
		}
		return fmt.Errorf("list GitHub App installations page %d: %w", page, err)
	}

	candidates = undiscoveredInstallations(candidates, metadata.PendingInstallations)
	verified, retries, verificationErr := verifyAccessibleInstallationsWithFailures(
		requestContext,
		ctx.Integration,
		app,
		identity,
		candidates,
		nil,
	)
	metadata.SetPendingInstallations(mergeVerifiedInstallations(verified, metadata.PendingInstallations))
	metadata.InstallationDiscovery.NextPage = nextPage
	metadata.InstallationDiscovery.RetryCandidates = uniqueDiscoveryCandidates(retries)
	if nextPage == 0 && len(metadata.InstallationDiscovery.RetryCandidates) == 0 {
		completionErr := finishHostedInitialDiscovery(metadata)
		logHostedDiscovery(ctx, "initial_page", len(candidates), len(verified), errors.Join(verificationErr, completionErr), startedAt)
		return completionErr
	}
	logHostedDiscovery(ctx, "initial_page", len(candidates), len(verified), verificationErr, startedAt)
	return nil
}

func (g *GitHub) retryHostedInitialInstallations(
	requestContext context.Context,
	ctx core.SyncContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
	metadata *common.Metadata,
	startedAt time.Time,
) error {
	candidates := slices.Clone(metadata.InstallationDiscovery.RetryCandidates)
	verified, retries, err := verifyAccessibleInstallationsWithFailures(
		requestContext,
		ctx.Integration,
		app,
		identity,
		candidates,
		nil,
	)
	metadata.SetPendingInstallations(mergeVerifiedInstallations(verified, metadata.PendingInstallations))
	// Discard candidates after this one retry so a persistent GitHub error does
	// not block the next page. The cycle failure prevents empty completion.
	metadata.InstallationDiscovery.RetryCandidates = nil
	if len(retries) > 0 {
		metadata.InstallationDiscovery.TransientFailures = true
	}
	if metadata.InstallationDiscovery.NextPage > 0 {
		logHostedDiscovery(ctx, "initial_retry", len(candidates), len(verified), err, startedAt)
		return nil
	}

	completionErr := finishHostedInitialDiscovery(metadata)
	logHostedDiscovery(ctx, "initial_retry", len(candidates), len(verified), errors.Join(err, completionErr), startedAt)
	return completionErr
}

func supportsHostedInitialDiscovery(returnPath string) bool {
	pathname, _, _ := strings.Cut(returnPath, "?")
	return pathname == "/onboarding" ||
		(strings.Contains(pathname, "/workspaces/") && strings.HasSuffix(pathname, "/setup"))
}

func ensureHostedInitialDiscovery(metadata *common.Metadata) {
	if metadata.InstallationDiscovery != nil {
		return
	}
	metadata.InstallationDiscovery = &common.InstallationDiscovery{
		Active:   true,
		NextPage: 1,
	}
}

func completeHostedInitialDiscovery(metadata *common.Metadata) {
	if metadata.InstallationDiscovery == nil {
		return
	}
	metadata.InstallationDiscovery.Active = false
	metadata.InstallationDiscovery.Complete = true
	metadata.InstallationDiscovery.NextPage = 0
	metadata.InstallationDiscovery.RetryCandidates = nil
	metadata.InstallationDiscovery.TransientFailures = false
	metadata.InstallationDiscovery.InstallAvailable = false
}

func finishHostedInitialDiscovery(metadata *common.Metadata) error {
	if metadata.InstallationDiscovery == nil || !metadata.InstallationDiscovery.TransientFailures {
		completeHostedInitialDiscovery(metadata)
		return nil
	}

	metadata.InstallationDiscovery.PersonalAccountChecked = false
	metadata.InstallationDiscovery.Active = true
	metadata.InstallationDiscovery.Complete = false
	metadata.InstallationDiscovery.NextPage = 1
	metadata.InstallationDiscovery.RetryCandidates = nil
	metadata.InstallationDiscovery.TransientFailures = false
	metadata.InstallationDiscovery.InstallAvailable = true
	return errors.New("GitHub account discovery is temporarily unavailable")
}

func hostedInitialDiscoveryActive(metadata common.Metadata) bool {
	return metadata.InstallationDiscovery != nil &&
		metadata.InstallationDiscovery.Active &&
		!metadata.InstallationDiscovery.Complete
}

func undiscoveredInstallations(
	candidates []common.PendingInstallation,
	discovered []common.PendingInstallation,
) []common.PendingInstallation {
	return slices.DeleteFunc(slices.Clone(candidates), func(candidate common.PendingInstallation) bool {
		return slices.ContainsFunc(discovered, func(existing common.PendingInstallation) bool {
			return existing.ID == candidate.ID
		})
	})
}

func uniqueDiscoveryCandidates(candidates []common.PendingInstallation) []common.PendingInstallation {
	unique := make([]common.PendingInstallation, 0, min(len(candidates), hostedInitialDiscoveryPageSize))
	for _, candidate := range candidates {
		if len(unique) == hostedInitialDiscoveryPageSize {
			break
		}
		if candidate.ID == "" || slices.ContainsFunc(unique, func(existing common.PendingInstallation) bool {
			return existing.ID == candidate.ID
		}) {
			continue
		}
		candidate.Repositories = nil
		unique = append(unique, candidate)
	}
	return unique
}
