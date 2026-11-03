package vcs

import "context"

// PullRequestDescription retains the host's raw Markdown and current revision.
type PullRequestDescription struct {
	Description string
	State       string
	HeadSHA     string
}

// DescriptionClient updates only the description field on the host.
// Readers must not substitute rendered HTML or normalized Markdown.
type DescriptionClient interface {
	ReadDescription(context.Context, PullRequestRef) (PullRequestDescription, error)
	UpdateDescription(context.Context, PullRequestRef, string) error
}
