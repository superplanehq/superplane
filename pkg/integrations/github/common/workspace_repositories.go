package common

import (
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const WorkspaceRepositoriesField = "workspaceRepositories"

// WorkspaceRepositoriesConfigurationField lets a node in a factory-owned
// app listen on every repository of the workspace.
func WorkspaceRepositoriesConfigurationField() configuration.Field {
	return configuration.Field{
		Name:        WorkspaceRepositoriesField,
		Label:       "All workspace repositories",
		Type:        configuration.FieldTypeBool,
		Required:    false,
		Default:     false,
		Description: "Also listen on the other repositories of the workspace. Use this only in apps that a workspace owns.",
	}
}

// WorkspaceWebhookRepositories returns the repositories for a webhook of a
// node that listens on the whole workspace: the configured repository
// first, then the other workspace repositories.
func WorkspaceWebhookRepositories(factory core.FactoryContext, enabled bool, repository string) ([]string, error) {
	if !enabled || factory == nil {
		return nil, nil
	}

	listed, err := factory.ListRepositories()
	if err != nil {
		return nil, err
	}

	repositories := make([]string, 0, len(listed)+1)
	repository = strings.TrimSpace(repository)
	if repository != "" && !IsExpression(repository) {
		repositories = append(repositories, repository)
	}
	for _, candidate := range listed {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || slices.ContainsFunc(repositories, func(existing string) bool {
			return strings.EqualFold(existing, candidate)
		}) {
			continue
		}
		repositories = append(repositories, candidate)
	}
	return repositories, nil
}

// WorkspaceWebhookConfiguration adds the workspace repositories to a
// webhook request when the node listens on the whole workspace. A
// workspace with one repository keeps a single-repository webhook, so its
// existing webhook stays in place.
func WorkspaceWebhookConfiguration(factory core.FactoryContext, enabled bool, config WebhookConfiguration) (WebhookConfiguration, error) {
	repositories, err := WorkspaceWebhookRepositories(factory, enabled, config.Repository)
	if err != nil {
		return config, err
	}
	if len(repositories) == 0 {
		return config, nil
	}
	config.Repository = repositories[0]
	if len(repositories) > 1 {
		config.Repositories = repositories
	}
	return config, nil
}
