package me

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"golang.org/x/sync/errgroup"
)

const (
	vcsProviderInstallationCheckConcurrency = 4
	vcsProviderInstallationCheckTimeout     = 15 * time.Second
)

type VCSProviderInstallationVerifier interface {
	VerifyInstallation(ctx context.Context, installationID int64) error
}

// VCSProviderInstallationChecks keeps one provider check per installation in
// each interval, so many open onboarding screens do not multiply the
// provider API calls.
type VCSProviderInstallationChecks struct {
	interval  time.Duration
	mu        sync.Mutex
	running   map[int64]struct{}
	checkedAt map[int64]time.Time
	prunedAt  time.Time
}

func NewVCSProviderInstallationChecks(interval time.Duration) *VCSProviderInstallationChecks {
	return &VCSProviderInstallationChecks{
		interval:  interval,
		running:   map[int64]struct{}{},
		checkedAt: map[int64]time.Time{},
	}
}

// start returns the installations that have no running check and no check
// that finished in the interval, and records their checks as running.
func (c *VCSProviderInstallationChecks) start(installationIDs []int64, now time.Time) []int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneExpired(now)
	started := make([]int64, 0, len(installationIDs))
	for _, installationID := range installationIDs {
		if _, running := c.running[installationID]; running {
			continue
		}
		if checkedAt, checked := c.checkedAt[installationID]; checked && now.Sub(checkedAt) < c.interval {
			continue
		}
		c.running[installationID] = struct{}{}
		started = append(started, installationID)
	}
	return started
}

func (c *VCSProviderInstallationChecks) finish(installationID int64, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.running, installationID)
	c.checkedAt[installationID] = now
}

func (c *VCSProviderInstallationChecks) pruneExpired(now time.Time) {
	if now.Sub(c.prunedAt) < c.interval {
		return
	}
	c.prunedAt = now
	for installationID, checkedAt := range c.checkedAt {
		if now.Sub(checkedAt) >= c.interval {
			delete(c.checkedAt, installationID)
		}
	}
}

// VerifyVCSProviderInstallations checks that the installations the user can
// see still exist. GitHub does not always send a webhook when an App is
// uninstalled. Without this check, an uninstalled organization stays on the
// onboarding screens until the next full catalog reconciliation.
func VerifyVCSProviderInstallations(
	ctx context.Context,
	provider string,
	checks *VCSProviderInstallationChecks,
	verifier VCSProviderInstallationVerifier,
) (*pb.VerifyVCSProviderInstallationsResponse, error) {
	provider, err := supportedVCSProvider(provider)
	if err != nil {
		return nil, err
	}
	if provider == models.ProviderBitbucket {
		return &pb.VerifyVCSProviderInstallationsResponse{}, nil
	}
	if !vcsProviderConfigured(provider) {
		return nil, grpcerrors.FailedPrecondition(nil, "public GitHub App is not configured")
	}
	identity, err := currentVCSProviderIdentity(ctx, provider)
	if err != nil {
		return nil, vcsProviderIdentityError(err)
	}
	repositories, err := models.ListAccessibleVCSProviderRepositories(database.DB(ctx), provider, identity.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list accessible repositories")
	}

	installationIDs := make([]int64, 0)
	seen := map[int64]struct{}{}
	for _, repository := range repositories {
		if _, ok := seen[repository.InstallationID]; ok {
			continue
		}
		seen[repository.InstallationID] = struct{}{}
		installationIDs = append(installationIDs, repository.InstallationID)
	}

	verifyInstallations(ctx, verifier, checks, checks.start(installationIDs, time.Now()))
	return &pb.VerifyVCSProviderInstallationsResponse{}, nil
}

// The installations are already recorded as running, so the checks must
// finish even when the page closes the request early.
func verifyInstallations(
	ctx context.Context,
	verifier VCSProviderInstallationVerifier,
	checks *VCSProviderInstallationChecks,
	installationIDs []int64,
) {
	checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), vcsProviderInstallationCheckTimeout)
	defer cancel()
	var group errgroup.Group
	group.SetLimit(vcsProviderInstallationCheckConcurrency)
	for _, installationID := range installationIDs {
		group.Go(func() error {
			defer func() { checks.finish(installationID, time.Now()) }()
			if err := verifier.VerifyInstallation(checkCtx, installationID); err != nil {
				log.WithError(err).
					WithField("installation_id", installationID).
					Warn("failed to verify VCS provider installation")
			}
			return nil
		})
	}
	_ = group.Wait()
}
