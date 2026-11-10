package bitbucket

import (
	"errors"

	"github.com/superplanehq/superplane/pkg/core"
)

// requestRepositoryWebhook resolves the node repository and requests its
// webhook. A node in a workspace app with an expression repository listens
// on every workspace repository. A workspace with one repository keeps the
// single-repository webhook, so its existing webhook stays in place.
func requestRepositoryWebhook(
	http core.HTTPContext,
	metadata core.MetadataWriter,
	integration core.IntegrationContext,
	factory core.FactoryContext,
	repository string,
	eventTypes []string,
) error {
	var repositories []string
	if listensOnWorkspace(factory, repository) {
		listed, err := factory.ListRepositories()
		if err != nil {
			return err
		}
		if len(listed) == 0 {
			return errors.New("the workspace has no repository")
		}
		repositories = listed
		repository = listed[0]
	}

	repo, err := ensureRepoInMetadata(http, metadata, integration, repository)
	if err != nil {
		return err
	}

	config := WebhookConfiguration{EventTypes: eventTypes, RepositorySlug: repo.Slug}
	if len(repositories) > 1 {
		config.Repositories = repositories
	}
	return integration.RequestWebhook(config)
}

// listensOnWorkspace reports whether a node in a workspace app listens on
// every workspace repository. The stored configuration keeps the
// expression, so such a node takes the repository from the event.
func listensOnWorkspace(factory core.FactoryContext, repository string) bool {
	return factory != nil && isExpression(repository)
}
