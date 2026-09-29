package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GitHubAppInstallation struct {
	InstallationID      int64 `gorm:"primaryKey"`
	AccountID           *int64
	AccountLogin        string
	AccountType         string
	HTMLURL             string `gorm:"column:html_url"`
	RepositorySelection string
	SuspendedAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (GitHubAppInstallation) TableName() string {
	return "github_app_installations"
}

type GitHubAppRepository struct {
	RepositoryID   int64 `gorm:"primaryKey"`
	InstallationID int64
	FullName       string
	Private        bool
	DefaultBranch  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (GitHubAppRepository) TableName() string {
	return "github_app_repositories"
}

type GitHubAppRepositoryCollaborator struct {
	RepositoryID int64  `gorm:"primaryKey"`
	GitHubUserID int64  `gorm:"primaryKey;column:github_user_id"`
	GitHubLogin  string `gorm:"column:github_login"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (GitHubAppRepositoryCollaborator) TableName() string {
	return "github_app_repository_collaborators"
}

type GitHubAppInstallRequest struct {
	RequestID      int64 `gorm:"primaryKey"`
	AccountID      *int64
	AccountLogin   string
	AccountType    string
	RequesterID    int64
	RequesterLogin string
	RequestedAt    time.Time
	UpdatedAt      time.Time
}

func (GitHubAppInstallRequest) TableName() string {
	return "github_app_install_requests"
}

type GitHubAppRepositorySyncJob struct {
	RepositoryID int64 `gorm:"primaryKey"`
	RunAt        time.Time
	Attempts     int
	LockedAt     *time.Time
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (GitHubAppRepositorySyncJob) TableName() string {
	return "github_app_repository_sync_jobs"
}

type GitHubAppReconcileJob struct {
	ID        int16 `gorm:"primaryKey"`
	RunAt     time.Time
	Attempts  int
	LockedAt  *time.Time
	LastError string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (GitHubAppReconcileJob) TableName() string {
	return "github_app_reconcile_jobs"
}

type GitHubAppIntegrationBinding struct {
	IntegrationID  uuid.UUID `gorm:"primaryKey"`
	OrganizationID uuid.UUID
	InstallationID int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (GitHubAppIntegrationBinding) TableName() string {
	return "github_app_integration_bindings"
}

type AccessibleGitHubAppRepository struct {
	GitHubAppRepository
	AccountLogin string
	AccountType  string
}

func UpsertGitHubAppInstallation(tx *gorm.DB, installation *GitHubAppInstallation) error {
	if installation == nil || installation.InstallationID <= 0 {
		return errors.New("GitHub App installation id is required")
	}

	installation.AccountLogin = strings.TrimSpace(installation.AccountLogin)
	installation.AccountType = strings.TrimSpace(installation.AccountType)
	installation.HTMLURL = strings.TrimSpace(installation.HTMLURL)
	installation.RepositorySelection = strings.TrimSpace(installation.RepositorySelection)

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "installation_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"account_id",
			"account_login",
			"account_type",
			"html_url",
			"repository_selection",
			"suspended_at",
			"updated_at",
		}),
	}).Create(installation).Error
}

func FindGitHubAppInstallation(tx *gorm.DB, installationID int64) (*GitHubAppInstallation, error) {
	var installation GitHubAppInstallation
	if err := tx.Where("installation_id = ?", installationID).First(&installation).Error; err != nil {
		return nil, err
	}
	return &installation, nil
}

func DeleteGitHubAppInstallation(tx *gorm.DB, installationID int64) error {
	return tx.Where("installation_id = ?", installationID).Delete(&GitHubAppInstallation{}).Error
}

func ReplaceGitHubAppRepositories(tx *gorm.DB, installationID int64, repositories []GitHubAppRepository) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		repositoryIDs := make([]int64, 0, len(repositories))
		for i := range repositories {
			repositories[i].InstallationID = installationID
			repositories[i].FullName = strings.TrimSpace(repositories[i].FullName)
			if repositories[i].RepositoryID <= 0 || repositories[i].FullName == "" {
				return errors.New("GitHub repository id and full name are required")
			}
			repositoryIDs = append(repositoryIDs, repositories[i].RepositoryID)
		}

		deleteQuery := tx.Where("installation_id = ?", installationID)
		if len(repositoryIDs) > 0 {
			deleteQuery = deleteQuery.Where("repository_id NOT IN ?", repositoryIDs)
		}
		if err := deleteQuery.Delete(&GitHubAppRepository{}).Error; err != nil {
			return err
		}

		if len(repositories) == 0 {
			return nil
		}

		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "repository_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"installation_id",
				"full_name",
				"private",
				"default_branch",
				"updated_at",
			}),
		}).Create(&repositories).Error
	})
}

func UpsertGitHubAppRepositories(tx *gorm.DB, installationID int64, repositories []GitHubAppRepository) error {
	if len(repositories) == 0 {
		return nil
	}

	for i := range repositories {
		repositories[i].InstallationID = installationID
		repositories[i].FullName = strings.TrimSpace(repositories[i].FullName)
		if repositories[i].RepositoryID <= 0 || repositories[i].FullName == "" {
			return errors.New("GitHub repository id and full name are required")
		}
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "repository_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"installation_id",
			"full_name",
			"private",
			"default_branch",
			"updated_at",
		}),
	}).Create(&repositories).Error
}

func DeleteGitHubAppRepositories(tx *gorm.DB, installationID int64, repositoryIDs []int64) error {
	if len(repositoryIDs) == 0 {
		return nil
	}
	return tx.
		Where("installation_id = ?", installationID).
		Where("repository_id IN ?", repositoryIDs).
		Delete(&GitHubAppRepository{}).
		Error
}

func FindGitHubAppRepository(tx *gorm.DB, repositoryID int64) (*GitHubAppRepository, error) {
	var repository GitHubAppRepository
	if err := tx.Where("repository_id = ?", repositoryID).First(&repository).Error; err != nil {
		return nil, err
	}
	return &repository, nil
}

func ListGitHubAppRepositories(tx *gorm.DB, installationID int64) ([]GitHubAppRepository, error) {
	var repositories []GitHubAppRepository
	err := tx.
		Where("installation_id = ?", installationID).
		Order("LOWER(full_name) ASC").
		Find(&repositories).
		Error
	return repositories, err
}

func GitHubAppRepositoriesSynchronizing(tx *gorm.DB, repositoryIDs []int64) (bool, error) {
	if len(repositoryIDs) == 0 {
		return false, nil
	}
	var count int64
	err := tx.Model(&GitHubAppRepositorySyncJob{}).
		Where("repository_id IN ?", repositoryIDs).
		Count(&count).
		Error
	return count > 0, err
}

func GitHubAppCatalogSynchronizing(tx *gorm.DB) (bool, error) {
	var count int64
	if err := tx.Model(&GitHubAppRepositorySyncJob{}).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	if err := tx.Model(&GitHubAppReconcileJob{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func EnqueueGitHubAppReconciliation(tx *gorm.DB, runAt time.Time) error {
	job := GitHubAppReconcileJob{ID: 1, RunAt: runAt}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"run_at":     runAt,
			"locked_at":  nil,
			"last_error": "",
			"updated_at": time.Now(),
		}),
	}).Create(&job).Error
}

func ClaimGitHubAppReconciliation(tx *gorm.DB, now, claimableBefore time.Time) (*GitHubAppReconcileJob, error) {
	var claimed *GitHubAppReconcileJob
	err := tx.Transaction(func(tx *gorm.DB) error {
		var job GitHubAppReconcileJob
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("run_at <= ?", now).
			Where("locked_at IS NULL OR locked_at < ?", claimableBefore).
			First(&job).
			Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		job.LockedAt = &now
		job.Attempts++
		if err := tx.Model(&job).Updates(map[string]any{
			"locked_at":  now,
			"attempts":   job.Attempts,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		claimed = &job
		return nil
	})
	return claimed, err
}

func CompleteGitHubAppReconciliation(tx *gorm.DB) error {
	return tx.Where("id = ?", 1).Delete(&GitHubAppReconcileJob{}).Error
}

func RetryGitHubAppReconciliation(tx *gorm.DB, runAt time.Time, reconcileError error) error {
	message := ""
	if reconcileError != nil {
		message = reconcileError.Error()
	}
	return tx.Model(&GitHubAppReconcileJob{}).
		Where("id = ?", 1).
		Updates(map[string]any{
			"run_at":     runAt,
			"locked_at":  nil,
			"last_error": message,
			"updated_at": time.Now(),
		}).
		Error
}

func ReplaceGitHubAppRepositoryCollaborators(
	tx *gorm.DB,
	repositoryID int64,
	collaborators []GitHubAppRepositoryCollaborator,
) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("repository_id = ?", repositoryID).Delete(&GitHubAppRepositoryCollaborator{}).Error; err != nil {
			return err
		}

		if len(collaborators) == 0 {
			return nil
		}

		for i := range collaborators {
			collaborators[i].RepositoryID = repositoryID
			collaborators[i].GitHubLogin = strings.TrimSpace(collaborators[i].GitHubLogin)
			if collaborators[i].GitHubUserID <= 0 {
				return errors.New("GitHub collaborator user id is required")
			}
		}
		return tx.Create(&collaborators).Error
	})
}

func ListAccessibleGitHubAppRepositories(tx *gorm.DB, githubUserID int64) ([]AccessibleGitHubAppRepository, error) {
	var repositories []AccessibleGitHubAppRepository
	err := tx.
		Table("github_app_repositories AS repository").
		Select("repository.*, installation.account_login, installation.account_type").
		Joins("JOIN github_app_repository_collaborators AS collaborator ON collaborator.repository_id = repository.repository_id").
		Joins("JOIN github_app_installations AS installation ON installation.installation_id = repository.installation_id").
		Where("collaborator.github_user_id = ?", githubUserID).
		Where("installation.suspended_at IS NULL").
		Order("LOWER(repository.full_name) ASC").
		Scan(&repositories).
		Error
	return repositories, err
}

func FindAccessibleGitHubAppRepository(tx *gorm.DB, githubUserID, repositoryID int64) (*AccessibleGitHubAppRepository, error) {
	var repository AccessibleGitHubAppRepository
	err := tx.
		Table("github_app_repositories AS repository").
		Select("repository.*, installation.account_login, installation.account_type").
		Joins("JOIN github_app_repository_collaborators AS collaborator ON collaborator.repository_id = repository.repository_id").
		Joins("JOIN github_app_installations AS installation ON installation.installation_id = repository.installation_id").
		Where("collaborator.github_user_id = ?", githubUserID).
		Where("repository.repository_id = ?", repositoryID).
		Where("installation.suspended_at IS NULL").
		First(&repository).
		Error
	if err != nil {
		return nil, err
	}
	return &repository, nil
}

func ReplaceGitHubAppInstallRequests(tx *gorm.DB, requests []GitHubAppInstallRequest) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&GitHubAppInstallRequest{}).Error; err != nil {
			return err
		}
		if len(requests) == 0 {
			return nil
		}
		return tx.Create(&requests).Error
	})
}

func ListGitHubAppInstallRequests(tx *gorm.DB, requesterID int64) ([]GitHubAppInstallRequest, error) {
	var requests []GitHubAppInstallRequest
	err := tx.
		Where("requester_id = ?", requesterID).
		Order("requested_at ASC").
		Find(&requests).
		Error
	return requests, err
}

func DeleteGitHubAppInstallRequestsForAccount(tx *gorm.DB, accountID *int64, accountLogin string) error {
	query := tx.Model(&GitHubAppInstallRequest{})
	if accountID != nil {
		return query.Where("account_id = ?", *accountID).Delete(&GitHubAppInstallRequest{}).Error
	}
	accountLogin = strings.TrimSpace(accountLogin)
	if accountLogin == "" {
		return nil
	}
	return query.Where("LOWER(account_login) = LOWER(?)", accountLogin).Delete(&GitHubAppInstallRequest{}).Error
}

func EnqueueGitHubAppRepositorySync(tx *gorm.DB, repositoryID int64, runAt time.Time) error {
	job := GitHubAppRepositorySyncJob{RepositoryID: repositoryID, RunAt: runAt}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "repository_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"run_at":     runAt,
			"locked_at":  nil,
			"last_error": "",
			"updated_at": time.Now(),
		}),
	}).Create(&job).Error
}

func ClaimGitHubAppRepositorySync(tx *gorm.DB, now, claimableBefore time.Time) (*GitHubAppRepositorySyncJob, error) {
	var claimed *GitHubAppRepositorySyncJob
	err := tx.Transaction(func(tx *gorm.DB) error {
		var job GitHubAppRepositorySyncJob
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("run_at <= ?", now).
			Where("locked_at IS NULL OR locked_at < ?", claimableBefore).
			Order("run_at ASC").
			First(&job).
			Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		job.LockedAt = &now
		job.Attempts++
		if err := tx.Model(&job).Updates(map[string]any{
			"locked_at":  now,
			"attempts":   job.Attempts,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		claimed = &job
		return nil
	})
	return claimed, err
}

func CompleteGitHubAppRepositorySync(tx *gorm.DB, repositoryID int64) error {
	return tx.Where("repository_id = ?", repositoryID).Delete(&GitHubAppRepositorySyncJob{}).Error
}

func RetryGitHubAppRepositorySync(tx *gorm.DB, repositoryID int64, runAt time.Time, syncError error) error {
	message := ""
	if syncError != nil {
		message = syncError.Error()
	}
	return tx.Model(&GitHubAppRepositorySyncJob{}).
		Where("repository_id = ?", repositoryID).
		Updates(map[string]any{
			"run_at":     runAt,
			"locked_at":  nil,
			"last_error": message,
			"updated_at": time.Now(),
		}).
		Error
}

func FindOrCreateHostedGitHubBinding(
	tx *gorm.DB,
	organizationID uuid.UUID,
	installationID int64,
	accountLogin string,
) (*Integration, error) {
	var integration *Integration
	err := tx.Transaction(func(tx *gorm.DB) error {
		// Keep every completed historical integration during cutover, but
		// serialize all new selections so that they reuse one existing binding.
		lockKey := fmt.Sprintf("github-app-binding:%s:%d", organizationID, installationID)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		existing, err := findHostedGitHubBinding(tx, organizationID, installationID)
		if err == nil {
			integration = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		now := time.Now()
		candidate := &Integration{
			ID:               uuid.New(),
			OrganizationID:   organizationID,
			AppName:          "github",
			InstallationName: "github-" + strings.TrimSpace(accountLogin),
			State:            IntegrationStateReady,
			StateDescription: "",
			Configuration:    datatypes.NewJSONType(map[string]any{}),
			Metadata: datatypes.NewJSONType(map[string]any{
				"hostedApp": true,
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if candidate.InstallationName == "github-" {
			candidate.InstallationName = fmt.Sprintf("github-%d", installationID)
		}
		if err := candidate.AssignUniqueInstallationName(tx, candidate.InstallationName); err != nil {
			return err
		}
		if err := tx.Create(candidate).Error; err != nil {
			return err
		}

		binding := GitHubAppIntegrationBinding{
			IntegrationID:  candidate.ID,
			OrganizationID: organizationID,
			InstallationID: installationID,
		}
		if err := tx.Create(&binding).Error; err != nil {
			return err
		}

		integration = candidate
		return nil
	})
	return integration, err
}

func findHostedGitHubBinding(tx *gorm.DB, organizationID uuid.UUID, installationID int64) (*Integration, error) {
	var integration Integration
	err := tx.
		Table("app_installations AS integration").
		Select("integration.*").
		Joins("JOIN github_app_integration_bindings AS binding ON binding.integration_id = integration.id").
		Where("binding.organization_id = ?", organizationID).
		Where("binding.installation_id = ?", installationID).
		Where("integration.deleted_at IS NULL").
		Order("binding.created_at ASC").
		First(&integration).
		Error
	if err != nil {
		return nil, err
	}
	return &integration, nil
}

func FindGitHubAppIntegrationBinding(tx *gorm.DB, integrationID uuid.UUID) (*GitHubAppIntegrationBinding, error) {
	var binding GitHubAppIntegrationBinding
	if err := tx.Where("integration_id = ?", integrationID).First(&binding).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

func ListGitHubAppIntegrationBindings(tx *gorm.DB, installationID int64) ([]GitHubAppIntegrationBinding, error) {
	var bindings []GitHubAppIntegrationBinding
	err := tx.Where("installation_id = ?", installationID).Order("created_at ASC").Find(&bindings).Error
	return bindings, err
}

func ListGitHubAppBoundIntegrations(tx *gorm.DB, installationID int64) ([]Integration, error) {
	var integrations []Integration
	err := tx.
		Table("app_installations AS integration").
		Select("integration.*").
		Joins("JOIN github_app_integration_bindings AS binding ON binding.integration_id = integration.id").
		Where("binding.installation_id = ?", installationID).
		Where("integration.deleted_at IS NULL").
		Find(&integrations).
		Error
	return integrations, err
}
