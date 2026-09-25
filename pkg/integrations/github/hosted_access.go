package github

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	gh "github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type hostedGitHubIdentity struct {
	ID    int64
	Login string
}

type repositoryPermissionLookup func(context.Context, string, string, string) (*gh.RepositoryPermissionLevel, error)

func findStartedByGitHubIdentity(organizationID, userID string) (*hostedGitHubIdentity, error) {
	if organizationID == "" || userID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	user, err := models.FindActiveUserByIDInTransaction(database.Conn(), organizationID, userID)
	if err != nil {
		return nil, err
	}
	if user.AccountID == nil {
		return nil, gorm.ErrRecordNotFound
	}

	identity, err := models.FindAccountGitHubIdentity(database.Conn(), *user.AccountID)
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(identity.ProviderID, 10, 64)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid GitHub identity ID")
	}

	return &hostedGitHubIdentity{ID: id, Login: strings.TrimSpace(identity.Username)}, nil
}

func hostedIdentityConnectURL(baseURL, returnPath string) string {
	if baseURL == "" {
		return ""
	}
	if returnPath == "" {
		returnPath = "/"
	}

	values := url.Values{}
	values.Set("intent", "connect")
	values.Set("redirect", returnPath)
	return strings.TrimRight(baseURL, "/") + "/auth/github?" + values.Encode()
}

func discoverAccessibleInstallations(
	ctx context.Context,
	integration core.IntegrationContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
) ([]common.PendingInstallation, error) {
	appClient, err := newAppJWTClient(integration, app.ID)
	if err != nil {
		return nil, fmt.Errorf("create GitHub App client: %w", err)
	}

	installations, err := listAppInstallations(ctx, appClient)
	if err != nil {
		return nil, fmt.Errorf("list GitHub App installations: %w", err)
	}

	accessible := make([]common.PendingInstallation, 0, len(installations))
	failedChecks := make([]error, 0)
	for _, installation := range installations {
		client, err := newInstallationClient(integration, app.ID, installation.ID)
		if err != nil {
			failedChecks = append(failedChecks, fmt.Errorf("create client for installation %s: %w", installation.ID, err))
			continue
		}

		user, _, err := client.Users.GetByID(ctx, identity.ID)
		if err != nil {
			var responseError *gh.ErrorResponse
			if errors.As(err, &responseError) && responseError.Response != nil && responseError.Response.StatusCode == 404 {
				continue
			}
			failedChecks = append(failedChecks, fmt.Errorf("resolve GitHub identity for installation %s: %w", installation.ID, err))
			continue
		}
		if user.GetID() != identity.ID || user.GetLogin() == "" {
			continue
		}

		repositories, err := listInstallationRepos(ctx, client)
		if err != nil {
			failedChecks = append(failedChecks, fmt.Errorf("list repositories for installation %s: %w", installation.ID, err))
			continue
		}

		writable, err := filterWritableRepositories(
			ctx,
			installation.AccountLogin,
			user.GetLogin(),
			repositories,
			func(ctx context.Context, owner, repository, username string) (*gh.RepositoryPermissionLevel, error) {
				permission, _, err := client.Repositories.GetPermissionLevel(ctx, owner, repository, username)
				return permission, err
			},
		)
		if err != nil {
			failedChecks = append(failedChecks, fmt.Errorf("check repository access for installation %s: %w", installation.ID, err))
			continue
		}
		if len(writable) == 0 {
			continue
		}

		installation.Repositories = writable
		accessible = append(accessible, installation)
	}
	if len(failedChecks) > 0 {
		return accessible, errors.Join(failedChecks...)
	}

	return accessible, nil
}

func filterWritableRepositories(
	ctx context.Context,
	owner string,
	username string,
	repositories []common.Repository,
	lookup repositoryPermissionLookup,
) ([]common.Repository, error) {
	writable := make([]common.Repository, 0, len(repositories))
	for _, repository := range repositories {
		permission, err := lookup(ctx, owner, repository.Name, username)
		if err != nil {
			return nil, err
		}
		if permission == nil || !hasRepositoryWritePermission(permission.GetPermission()) {
			continue
		}

		repository.Name = owner + "/" + repository.Name
		writable = append(writable, repository)
	}
	return writable, nil
}

func hasRepositoryWritePermission(permission string) bool {
	return permission == "write" || permission == "admin"
}

func retainInstalledRepositories(granted, installed []common.Repository) []common.Repository {
	installedIDs := make(map[int64]struct{}, len(installed))
	for _, repository := range installed {
		installedIDs[repository.ID] = struct{}{}
	}

	remaining := make([]common.Repository, 0, len(granted))
	for _, repository := range granted {
		if _, ok := installedIDs[repository.ID]; ok {
			remaining = append(remaining, repository)
		}
	}
	return remaining
}
