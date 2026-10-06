// Package vcs is the factory workspace Git host seam.
// GitHub is the implementation used when the provider is empty.
// Bitbucket returns ErrNotSupported until its own client exists.
// Clone tokens stay on the integration ResolveSecrets path.
package vcs

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
)

// ErrNotSupported is returned when the workspace Git host cannot run
// the operation. Callers must not turn this into a GitHub client error.
var ErrNotSupported = errors.New("bitbucket is not supported")

// Provider is the forge operations a factory workspace uses.
type Provider interface {
	// ListMergedPullRequests returns pull requests that merged in [from, to).
	ListMergedPullRequests(ctx context.Context, repository string, from, to time.Time) ([]MergedPullRequest, error)
	// ReadMergeability reads merge status for one open pull request.
	ReadMergeability(ctx context.Context, pullRequest *models.FactoryPullRequest) (Mergeability, error)
	// MergePullRequest merges one open pull request.
	// method is the GitHub merge method name: squash, merge, or rebase.
	MergePullRequest(ctx context.Context, repository string, number int, method, expectedSHA string) error
	// ClosePullRequest declines one open pull request.
	ClosePullRequest(ctx context.Context, ref PullRequestRef) error
}

// MergedPullRequest is one merged pull request for the Velocity report.
type MergedPullRequest struct {
	Repository      string
	Number          int64
	Source          string
	AuthorLogin     string
	AuthorName      string
	AuthorAvatarURL string
	MergedAt        time.Time
}

// Mergeability is the merge status of one open pull request.
// BlockedReason and AllowedMethods use the stored column names.
type Mergeability struct {
	CanMerge       bool
	BlockedReason  string
	Message        string
	AllowedMethods []string
	HeadSHA        string
	CanRetry       bool
}

// PullRequestRef identifies one pull request to close.
type PullRequestRef struct {
	Provider   string
	Repository string
	Number     int64
}

// Select returns the Git host implementation for a factory.
// An empty provider uses GitHub. Only Bitbucket is unsupported.
// Any other provider keeps the GitHub implementation.
func Select(provider string, github Provider) Provider {
	if strings.EqualFold(strings.TrimSpace(provider), models.ProviderBitbucket) {
		return Unsupported{}
	}
	return github
}

// Unsupported rejects every forge operation.
type Unsupported struct{}

func (Unsupported) ListMergedPullRequests(context.Context, string, time.Time, time.Time) ([]MergedPullRequest, error) {
	return nil, ErrNotSupported
}

func (Unsupported) ReadMergeability(context.Context, *models.FactoryPullRequest) (Mergeability, error) {
	return Mergeability{}, ErrNotSupported
}

func (Unsupported) MergePullRequest(context.Context, string, int, string, string) error {
	return ErrNotSupported
}

func (Unsupported) ClosePullRequest(context.Context, PullRequestRef) error {
	return ErrNotSupported
}
