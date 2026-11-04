package common

import (
	"errors"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// ListensOnWorkspace reports whether a node in a workspace app listens on
// every workspace repository. Triggers use {{ workspace().repositories }}.
// Actions take their repository from the event, so any expression there
// can name any workspace repository.
func ListensOnWorkspace(factory core.FactoryContext, repository string) bool {
	return factory != nil && IsExpression(repository)
}

// WorkspaceWebhookConfiguration points the webhook at every workspace
// repository when the node listens on the whole workspace. A workspace with
// one repository keeps a single-repository webhook, so its existing webhook
// stays in place.
func WorkspaceWebhookConfiguration(factory core.FactoryContext, config WebhookConfiguration) (WebhookConfiguration, error) {
	if !ListensOnWorkspace(factory, config.Repository) {
		return config, nil
	}

	listed, err := factory.ListRepositories()
	if err != nil {
		return config, err
	}

	repositories := make([]string, 0, len(listed))
	for _, candidate := range listed {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || slices.ContainsFunc(repositories, func(existing string) bool {
			return strings.EqualFold(existing, candidate)
		}) {
			continue
		}
		repositories = append(repositories, candidate)
	}
	if len(repositories) == 0 {
		return config, errors.New("the workspace has no repository")
	}
	config.Repository = repositories[0]
	config.Repositories = nil
	if len(repositories) > 1 {
		config.Repositories = repositories
	}
	return config, nil
}
