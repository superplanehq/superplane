package githubapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	gh "github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	repositoryDiscoveryDelay = 10 * time.Second
)

// Catalog synchronizes the public SuperPlane GitHub App into the global
// installation catalog. It never uses a GitHub user OAuth token.
type Catalog struct {
	db           *gorm.DB
	config       config.GitHubHostedAppConfig
	appClient    *gh.Client
	installation func(int64) (*gh.Client, error)
	now          func() time.Time
}

func NewCatalog(db *gorm.DB, cfg config.GitHubHostedAppConfig) (*Catalog, error) {
	if !cfg.Enabled() {
		return nil, errors.New("public GitHub App is not configured")
	}

	appTransport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, cfg.ID, []byte(cfg.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("create GitHub App transport: %w", err)
	}

	return &Catalog{
		db:        db,
		config:    cfg,
		appClient: gh.NewClient(&http.Client{Transport: appTransport}),
		installation: func(installationID int64) (*gh.Client, error) {
			transport, err := ghinstallation.New(
				http.DefaultTransport,
				cfg.ID,
				installationID,
				[]byte(cfg.PrivateKey),
			)
			if err != nil {
				return nil, fmt.Errorf("create GitHub installation transport: %w", err)
			}
			return gh.NewClient(&http.Client{Transport: transport}), nil
		},
		now: time.Now,
	}, nil
}

// Reconcile imports installations, repositories, and pending approval
// requests. App-JWT requests follow the endpoints and pagination documented in
// onboarding-guide.md.
func (c *Catalog) Reconcile(ctx context.Context) error {
	installations, err := c.listInstallations(ctx)
	if err != nil {
		return err
	}

	seen := make(map[int64]struct{}, len(installations))
	for _, installation := range installations {
		if installation == nil || installation.GetID() <= 0 || installation.GetAccount() == nil {
			continue
		}
		seen[installation.GetID()] = struct{}{}
		if err := c.reconcileInstallation(ctx, installation); err != nil {
			return err
		}
	}

	if err := c.removeMissingInstallations(seen); err != nil {
		return err
	}

	requests, err := c.listInstallationRequests(ctx)
	if err != nil {
		return err
	}
	return models.ReplaceGitHubAppInstallRequests(c.db, installRequestModels(requests, c.now()))
}

func (c *Catalog) ReconcileInstallation(ctx context.Context, installationID int64) error {
	installation, _, err := c.appClient.Apps.GetInstallation(ctx, installationID)
	if err != nil {
		return fmt.Errorf("get GitHub App installation %d: %w", installationID, err)
	}
	return c.reconcileInstallation(ctx, installation)
}

func (c *Catalog) reconcileInstallation(ctx context.Context, installation *gh.Installation) error {
	model := installationModel(installation)
	if err := models.UpsertGitHubAppInstallation(c.db, &model); err != nil {
		return fmt.Errorf("save GitHub App installation %d: %w", model.InstallationID, err)
	}
	if err := models.DeleteGitHubAppInstallRequestsForAccount(c.db, model.AccountID, model.AccountLogin); err != nil {
		return fmt.Errorf("remove approved GitHub App request: %w", err)
	}
	if model.SuspendedAt != nil {
		return nil
	}

	repositories, err := c.listRepositories(ctx, model.InstallationID)
	if err != nil {
		return err
	}
	repositoryModels := repositoryModels(model.InstallationID, repositories)
	if err := models.ReplaceGitHubAppRepositories(c.db, model.InstallationID, repositoryModels); err != nil {
		return fmt.Errorf("save GitHub App repositories: %w", err)
	}
	return enqueueRepositories(c.db, repositoryModels, c.now().Add(repositoryDiscoveryDelay))
}

func (c *Catalog) SyncRepositoryCollaborators(ctx context.Context, repositoryID int64) error {
	repository, err := models.FindGitHubAppRepository(c.db, repositoryID)
	if err != nil {
		return err
	}
	client, err := c.installation(repository.InstallationID)
	if err != nil {
		return err
	}

	owner, name, ok := strings.Cut(repository.FullName, "/")
	if !ok || owner == "" || name == "" {
		return fmt.Errorf("invalid GitHub repository name %q", repository.FullName)
	}

	collaborators, err := listCollaborators(ctx, client, owner, name)
	if err != nil {
		return err
	}
	return models.ReplaceGitHubAppRepositoryCollaborators(
		c.db,
		repositoryID,
		collaboratorModels(repositoryID, collaborators),
	)
}

// EnqueueRefresh schedules database-cached repositories only. It does not
// block the request on GitHub.
func (c *Catalog) EnqueueRefresh(repositoryIDs []int64) error {
	runAt := c.now()
	for _, repositoryID := range repositoryIDs {
		if err := models.EnqueueGitHubAppRepositorySync(c.db, repositoryID, runAt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) listInstallations(ctx context.Context) ([]*gh.Installation, error) {
	installations := []*gh.Installation{}
	options := &gh.ListOptions{PerPage: 100}
	for {
		page, response, err := c.appClient.Apps.ListInstallations(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("list GitHub App installations: %w", err)
		}
		installations = append(installations, page...)
		if response.NextPage == 0 {
			return installations, nil
		}
		options.Page = response.NextPage
	}
}

func (c *Catalog) listInstallationRequests(ctx context.Context) ([]*gh.InstallationRequest, error) {
	requests := []*gh.InstallationRequest{}
	options := &gh.ListOptions{PerPage: 100}
	for {
		page, response, err := c.appClient.Apps.ListInstallationRequests(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("list GitHub App installation requests: %w", err)
		}
		requests = append(requests, page...)
		if response.NextPage == 0 {
			return requests, nil
		}
		options.Page = response.NextPage
	}
}

func (c *Catalog) listRepositories(ctx context.Context, installationID int64) ([]*gh.Repository, error) {
	client, err := c.installation(installationID)
	if err != nil {
		return nil, err
	}

	repositories := []*gh.Repository{}
	options := &gh.ListOptions{PerPage: 100}
	for {
		page, response, err := client.Apps.ListRepos(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("list repositories for GitHub installation %d: %w", installationID, err)
		}
		repositories = append(repositories, page.Repositories...)
		if response.NextPage == 0 {
			return repositories, nil
		}
		options.Page = response.NextPage
	}
}

func listCollaborators(ctx context.Context, client *gh.Client, owner, name string) ([]*gh.User, error) {
	collaborators := []*gh.User{}
	options := &gh.ListCollaboratorsOptions{ListOptions: gh.ListOptions{PerPage: 100}}
	for {
		page, response, err := client.Repositories.ListCollaborators(ctx, owner, name, options)
		if err != nil {
			return nil, fmt.Errorf("list collaborators for %s/%s: %w", owner, name, err)
		}
		collaborators = append(collaborators, page...)
		if response.NextPage == 0 {
			return collaborators, nil
		}
		options.Page = response.NextPage
	}
}

func installationModel(installation *gh.Installation) models.GitHubAppInstallation {
	var accountID *int64
	if installation.GetAccount().GetID() > 0 {
		value := installation.GetAccount().GetID()
		accountID = &value
	}
	var suspendedAt *time.Time
	if installation.SuspendedAt != nil {
		value := installation.SuspendedAt.Time
		suspendedAt = &value
	}
	return models.GitHubAppInstallation{
		InstallationID:      installation.GetID(),
		AccountID:           accountID,
		AccountLogin:        installation.GetAccount().GetLogin(),
		AccountType:         installation.GetTargetType(),
		HTMLURL:             installation.GetHTMLURL(),
		RepositorySelection: installation.GetRepositorySelection(),
		SuspendedAt:         suspendedAt,
	}
}

func repositoryModels(installationID int64, repositories []*gh.Repository) []models.GitHubAppRepository {
	result := make([]models.GitHubAppRepository, 0, len(repositories))
	for _, repository := range repositories {
		if repository == nil || repository.GetID() <= 0 || repository.GetFullName() == "" {
			continue
		}
		result = append(result, models.GitHubAppRepository{
			RepositoryID:   repository.GetID(),
			InstallationID: installationID,
			FullName:       repository.GetFullName(),
			Private:        repository.GetPrivate(),
			DefaultBranch:  repository.GetDefaultBranch(),
		})
	}
	return result
}

func collaboratorModels(repositoryID int64, collaborators []*gh.User) []models.GitHubAppRepositoryCollaborator {
	result := make([]models.GitHubAppRepositoryCollaborator, 0, len(collaborators))
	for _, collaborator := range collaborators {
		if collaborator == nil || collaborator.GetID() <= 0 || !collaborator.GetPermissions().GetPush() {
			continue
		}
		result = append(result, models.GitHubAppRepositoryCollaborator{
			RepositoryID: repositoryID,
			GitHubUserID: collaborator.GetID(),
			GitHubLogin:  collaborator.GetLogin(),
		})
	}
	return result
}

func installRequestModels(requests []*gh.InstallationRequest, now time.Time) []models.GitHubAppInstallRequest {
	result := make([]models.GitHubAppInstallRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil || request.GetID() <= 0 || request.GetRequester() == nil {
			continue
		}
		var accountID *int64
		if request.GetAccount().GetID() > 0 {
			value := request.GetAccount().GetID()
			accountID = &value
		}
		requestedAt := now
		if request.CreatedAt != nil {
			requestedAt = request.CreatedAt.Time
		}
		result = append(result, models.GitHubAppInstallRequest{
			RequestID:      request.GetID(),
			AccountID:      accountID,
			AccountLogin:   request.GetAccount().GetLogin(),
			AccountType:    request.GetAccount().GetType(),
			RequesterID:    request.GetRequester().GetID(),
			RequesterLogin: request.GetRequester().GetLogin(),
			RequestedAt:    requestedAt,
		})
	}
	return result
}

func enqueueRepositories(tx *gorm.DB, repositories []models.GitHubAppRepository, runAt time.Time) error {
	for _, repository := range repositories {
		if err := models.EnqueueGitHubAppRepositorySync(tx, repository.RepositoryID, runAt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) removeMissingInstallations(seen map[int64]struct{}) error {
	var current []models.GitHubAppInstallation
	if err := c.db.Find(&current).Error; err != nil {
		return err
	}
	for _, installation := range current {
		if _, ok := seen[installation.InstallationID]; ok {
			continue
		}
		if err := models.DeleteGitHubAppInstallation(c.db, installation.InstallationID); err != nil {
			return err
		}
	}
	return nil
}
