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

type VCSProviderInstallation struct {
	Provider            string `gorm:"primaryKey"`
	InstallationID      int64  `gorm:"primaryKey"`
	AccountID           *int64
	AccountLogin        string
	AccountType         string
	HTMLURL             string `gorm:"column:html_url"`
	RepositorySelection string
	SuspendedAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (VCSProviderInstallation) TableName() string {
	return "vcs_provider_installations"
}

type VCSProviderRepository struct {
	Provider       string `gorm:"primaryKey"`
	RepositoryID   int64  `gorm:"primaryKey"`
	InstallationID int64
	FullName       string
	Private        bool
	DefaultBranch  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (VCSProviderRepository) TableName() string {
	return "vcs_provider_repositories"
}

type VCSProviderRepositoryCollaborator struct {
	Provider       string `gorm:"primaryKey"`
	RepositoryID   int64  `gorm:"primaryKey"`
	ProviderUserID int64  `gorm:"primaryKey"`
	ProviderLogin  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (VCSProviderRepositoryCollaborator) TableName() string {
	return "vcs_provider_repository_collaborators"
}

type VCSProviderInstallRequest struct {
	Provider       string `gorm:"primaryKey"`
	RequestID      int64  `gorm:"primaryKey"`
	AccountID      *int64
	AccountLogin   string
	AccountType    string
	RequesterID    int64
	RequesterLogin string
	RequestedAt    time.Time
	UpdatedAt      time.Time
}

func (VCSProviderInstallRequest) TableName() string {
	return "vcs_provider_install_requests"
}

type VCSProviderRepositorySyncJob struct {
	Provider     string `gorm:"primaryKey"`
	RepositoryID int64  `gorm:"primaryKey"`
	RunAt        time.Time
	Attempts     int
	LockedAt     *time.Time
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (VCSProviderRepositorySyncJob) TableName() string {
	return "vcs_provider_repository_sync_jobs"
}

type VCSProviderReconcileJob struct {
	Provider  string `gorm:"primaryKey"`
	RunAt     time.Time
	Attempts  int
	LockedAt  *time.Time
	LastError string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (VCSProviderReconcileJob) TableName() string {
	return "vcs_provider_reconcile_jobs"
}

type VCSProviderIntegrationBinding struct {
	IntegrationID  uuid.UUID `gorm:"primaryKey"`
	OrganizationID uuid.UUID
	Provider       string
	InstallationID int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (VCSProviderIntegrationBinding) TableName() string {
	return "vcs_provider_integration_bindings"
}

type AccessibleVCSProviderRepository struct {
	VCSProviderRepository
	AccountLogin string
	AccountType  string
}

func normalizeVCSProvider(provider string) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return "", errors.New("VCS provider is required")
	}
	return provider, nil
}

func UpsertVCSProviderInstallation(tx *gorm.DB, installation *VCSProviderInstallation) error {
	if installation == nil || installation.InstallationID <= 0 {
		return errors.New("VCS provider installation id is required")
	}
	provider, err := normalizeVCSProvider(installation.Provider)
	if err != nil {
		return err
	}
	installation.Provider = provider

	installation.AccountLogin = strings.TrimSpace(installation.AccountLogin)
	installation.AccountType = strings.TrimSpace(installation.AccountType)
	installation.HTMLURL = strings.TrimSpace(installation.HTMLURL)
	installation.RepositorySelection = strings.TrimSpace(installation.RepositorySelection)

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}, {Name: "installation_id"}},
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

func FindVCSProviderInstallation(tx *gorm.DB, provider string, installationID int64) (*VCSProviderInstallation, error) {
	var installation VCSProviderInstallation
	if err := tx.Where("provider = ? AND installation_id = ?", provider, installationID).First(&installation).Error; err != nil {
		return nil, err
	}
	return &installation, nil
}

func DeleteVCSProviderInstallation(tx *gorm.DB, provider string, installationID int64) error {
	return tx.Where("provider = ? AND installation_id = ?", provider, installationID).Delete(&VCSProviderInstallation{}).Error
}

func ReplaceVCSProviderRepositories(
	tx *gorm.DB,
	provider string,
	installationID int64,
	repositories []VCSProviderRepository,
) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}
	return tx.Transaction(func(tx *gorm.DB) error {
		repositoryIDs := make([]int64, 0, len(repositories))
		for i := range repositories {
			repositories[i].Provider = provider
			repositories[i].InstallationID = installationID
			repositories[i].FullName = strings.TrimSpace(repositories[i].FullName)
			if repositories[i].RepositoryID <= 0 || repositories[i].FullName == "" {
				return errors.New("VCS provider repository id and full name are required")
			}
			repositoryIDs = append(repositoryIDs, repositories[i].RepositoryID)
		}

		deleteQuery := tx.Where("provider = ? AND installation_id = ?", provider, installationID)
		if len(repositoryIDs) > 0 {
			deleteQuery = deleteQuery.Where("repository_id NOT IN ?", repositoryIDs)
		}
		if err := deleteQuery.Delete(&VCSProviderRepository{}).Error; err != nil {
			return err
		}

		if len(repositories) == 0 {
			return nil
		}

		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "provider"}, {Name: "repository_id"}},
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

func UpsertVCSProviderRepositories(
	tx *gorm.DB,
	provider string,
	installationID int64,
	repositories []VCSProviderRepository,
) error {
	if len(repositories) == 0 {
		return nil
	}
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}

	for i := range repositories {
		repositories[i].Provider = provider
		repositories[i].InstallationID = installationID
		repositories[i].FullName = strings.TrimSpace(repositories[i].FullName)
		if repositories[i].RepositoryID <= 0 || repositories[i].FullName == "" {
			return errors.New("VCS provider repository id and full name are required")
		}
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}, {Name: "repository_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"installation_id",
			"full_name",
			"private",
			"default_branch",
			"updated_at",
		}),
	}).Create(&repositories).Error
}

func DeleteVCSProviderRepositories(tx *gorm.DB, provider string, installationID int64, repositoryIDs []int64) error {
	if len(repositoryIDs) == 0 {
		return nil
	}
	return tx.
		Where("provider = ? AND installation_id = ?", provider, installationID).
		Where("repository_id IN ?", repositoryIDs).
		Delete(&VCSProviderRepository{}).
		Error
}

func FindVCSProviderRepository(tx *gorm.DB, provider string, repositoryID int64) (*VCSProviderRepository, error) {
	var repository VCSProviderRepository
	if err := tx.Where("provider = ? AND repository_id = ?", provider, repositoryID).First(&repository).Error; err != nil {
		return nil, err
	}
	return &repository, nil
}

func ListVCSProviderRepositories(tx *gorm.DB, provider string, installationID int64) ([]VCSProviderRepository, error) {
	var repositories []VCSProviderRepository
	err := tx.
		Where("provider = ? AND installation_id = ?", provider, installationID).
		Order("LOWER(full_name) ASC").
		Find(&repositories).
		Error
	return repositories, err
}

func VCSProviderRepositoriesSynchronizing(tx *gorm.DB, provider string, repositoryIDs []int64) (bool, error) {
	if len(repositoryIDs) == 0 {
		return false, nil
	}
	var count int64
	err := tx.Model(&VCSProviderRepositorySyncJob{}).
		Where("provider = ? AND repository_id IN ?", provider, repositoryIDs).
		Count(&count).
		Error
	return count > 0, err
}

func VCSProviderCatalogSynchronizing(tx *gorm.DB, provider string) (bool, error) {
	var count int64
	if err := tx.Model(&VCSProviderRepositorySyncJob{}).Where("provider = ?", provider).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	if err := tx.Model(&VCSProviderReconcileJob{}).Where("provider = ?", provider).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func EnqueueVCSProviderReconciliation(tx *gorm.DB, provider string, runAt time.Time) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}
	job := VCSProviderReconcileJob{Provider: provider, RunAt: runAt}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}},
		DoUpdates: clause.Assignments(map[string]any{
			"run_at":     runAt,
			"locked_at":  nil,
			"last_error": "",
			"updated_at": time.Now(),
		}),
	}).Create(&job).Error
}

func ClaimVCSProviderReconciliation(
	tx *gorm.DB,
	provider string,
	now, claimableBefore time.Time,
) (*VCSProviderReconcileJob, error) {
	var claimed *VCSProviderReconcileJob
	err := tx.Transaction(func(tx *gorm.DB) error {
		var job VCSProviderReconcileJob
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("provider = ?", provider).
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

func CompleteVCSProviderReconciliation(tx *gorm.DB, provider string) error {
	return tx.Where("provider = ?", provider).Delete(&VCSProviderReconcileJob{}).Error
}

func RetryVCSProviderReconciliation(tx *gorm.DB, provider string, runAt time.Time, reconcileError error) error {
	message := ""
	if reconcileError != nil {
		message = reconcileError.Error()
	}
	return tx.Model(&VCSProviderReconcileJob{}).
		Where("provider = ?", provider).
		Updates(map[string]any{
			"run_at":     runAt,
			"locked_at":  nil,
			"last_error": message,
			"updated_at": time.Now(),
		}).
		Error
}

func ReplaceVCSProviderRepositoryCollaborators(
	tx *gorm.DB,
	provider string,
	repositoryID int64,
	collaborators []VCSProviderRepositoryCollaborator,
) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("provider = ? AND repository_id = ?", provider, repositoryID).Delete(&VCSProviderRepositoryCollaborator{}).Error; err != nil {
			return err
		}

		if len(collaborators) == 0 {
			return nil
		}

		for i := range collaborators {
			collaborators[i].Provider = provider
			collaborators[i].RepositoryID = repositoryID
			collaborators[i].ProviderLogin = strings.TrimSpace(collaborators[i].ProviderLogin)
			if collaborators[i].ProviderUserID <= 0 {
				return errors.New("VCS provider collaborator user id is required")
			}
		}
		return tx.Create(&collaborators).Error
	})
}

func ListAccessibleVCSProviderRepositories(
	tx *gorm.DB,
	provider string,
	providerUserID int64,
) ([]AccessibleVCSProviderRepository, error) {
	var repositories []AccessibleVCSProviderRepository
	err := tx.
		Table("vcs_provider_repositories AS repository").
		Select("repository.*, installation.account_login, installation.account_type").
		Joins("JOIN vcs_provider_repository_collaborators AS collaborator ON collaborator.provider = repository.provider AND collaborator.repository_id = repository.repository_id").
		Joins("JOIN vcs_provider_installations AS installation ON installation.provider = repository.provider AND installation.installation_id = repository.installation_id").
		Where("repository.provider = ?", provider).
		Where("collaborator.provider_user_id = ?", providerUserID).
		Where("installation.suspended_at IS NULL").
		Order("LOWER(repository.full_name) ASC").
		Scan(&repositories).
		Error
	return repositories, err
}

func FindAccessibleVCSProviderRepository(
	tx *gorm.DB,
	provider string,
	providerUserID, repositoryID int64,
) (*AccessibleVCSProviderRepository, error) {
	var repository AccessibleVCSProviderRepository
	err := tx.
		Table("vcs_provider_repositories AS repository").
		Select("repository.*, installation.account_login, installation.account_type").
		Joins("JOIN vcs_provider_repository_collaborators AS collaborator ON collaborator.provider = repository.provider AND collaborator.repository_id = repository.repository_id").
		Joins("JOIN vcs_provider_installations AS installation ON installation.provider = repository.provider AND installation.installation_id = repository.installation_id").
		Where("repository.provider = ?", provider).
		Where("collaborator.provider_user_id = ?", providerUserID).
		Where("repository.repository_id = ?", repositoryID).
		Where("installation.suspended_at IS NULL").
		First(&repository).
		Error
	if err != nil {
		return nil, err
	}
	return &repository, nil
}

func ReplaceVCSProviderInstallRequests(tx *gorm.DB, provider string, requests []VCSProviderInstallRequest) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("provider = ?", provider).Delete(&VCSProviderInstallRequest{}).Error; err != nil {
			return err
		}
		if len(requests) == 0 {
			return nil
		}
		for i := range requests {
			requests[i].Provider = provider
		}
		return tx.Create(&requests).Error
	})
}

func ListVCSProviderInstallRequests(tx *gorm.DB, provider string, requesterID int64) ([]VCSProviderInstallRequest, error) {
	var requests []VCSProviderInstallRequest
	err := tx.
		Where("provider = ? AND requester_id = ?", provider, requesterID).
		Order("requested_at ASC").
		Find(&requests).
		Error
	return requests, err
}

func DeleteVCSProviderInstallRequestsForAccount(
	tx *gorm.DB,
	provider string,
	accountID *int64,
	accountLogin string,
) error {
	query := tx.Model(&VCSProviderInstallRequest{})
	if accountID != nil {
		return query.Where("provider = ? AND account_id = ?", provider, *accountID).Delete(&VCSProviderInstallRequest{}).Error
	}
	accountLogin = strings.TrimSpace(accountLogin)
	if accountLogin == "" {
		return nil
	}
	return query.
		Where("provider = ? AND LOWER(account_login) = LOWER(?)", provider, accountLogin).
		Delete(&VCSProviderInstallRequest{}).
		Error
}

func EnqueueVCSProviderRepositorySync(tx *gorm.DB, provider string, repositoryID int64, runAt time.Time) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}
	job := VCSProviderRepositorySyncJob{Provider: provider, RepositoryID: repositoryID, RunAt: runAt}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}, {Name: "repository_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"run_at":     runAt,
			"locked_at":  nil,
			"last_error": "",
			"updated_at": time.Now(),
		}),
	}).Create(&job).Error
}

func ClaimVCSProviderRepositorySync(
	tx *gorm.DB,
	provider string,
	now, claimableBefore time.Time,
) (*VCSProviderRepositorySyncJob, error) {
	var claimed *VCSProviderRepositorySyncJob
	err := tx.Transaction(func(tx *gorm.DB) error {
		var job VCSProviderRepositorySyncJob
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("provider = ?", provider).
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

func CompleteVCSProviderRepositorySync(tx *gorm.DB, provider string, repositoryID int64) error {
	return tx.
		Where("provider = ? AND repository_id = ?", provider, repositoryID).
		Delete(&VCSProviderRepositorySyncJob{}).
		Error
}

func RetryVCSProviderRepositorySync(
	tx *gorm.DB,
	provider string,
	repositoryID int64,
	runAt time.Time,
	syncError error,
) error {
	message := ""
	if syncError != nil {
		message = syncError.Error()
	}
	return tx.Model(&VCSProviderRepositorySyncJob{}).
		Where("provider = ? AND repository_id = ?", provider, repositoryID).
		Updates(map[string]any{
			"run_at":     runAt,
			"locked_at":  nil,
			"last_error": message,
			"updated_at": time.Now(),
		}).
		Error
}

func FindOrCreateVCSProviderBinding(
	tx *gorm.DB,
	organizationID uuid.UUID,
	provider string,
	installationID int64,
	accountLogin string,
) (*Integration, error) {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return nil, err
	}
	var integration *Integration
	err = tx.Transaction(func(tx *gorm.DB) error {
		// Keep every completed historical integration during cutover, but
		// serialize all new selections so that they reuse one existing binding.
		lockKey := fmt.Sprintf("vcs-provider-binding:%s:%s:%d", organizationID, provider, installationID)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		existing, err := findVCSProviderBinding(tx, organizationID, provider, installationID)
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
			AppName:          provider,
			InstallationName: provider + "-" + strings.TrimSpace(accountLogin),
			State:            IntegrationStateReady,
			StateDescription: "",
			Configuration:    datatypes.NewJSONType(map[string]any{}),
			Metadata: datatypes.NewJSONType(map[string]any{
				"hostedApp": true,
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if candidate.InstallationName == provider+"-" {
			candidate.InstallationName = fmt.Sprintf("%s-%d", provider, installationID)
		}
		if err := candidate.AssignUniqueInstallationName(tx, candidate.InstallationName); err != nil {
			return err
		}
		if err := tx.Create(candidate).Error; err != nil {
			return err
		}

		binding := VCSProviderIntegrationBinding{
			IntegrationID:  candidate.ID,
			OrganizationID: organizationID,
			Provider:       provider,
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

func findVCSProviderBinding(
	tx *gorm.DB,
	organizationID uuid.UUID,
	provider string,
	installationID int64,
) (*Integration, error) {
	var integration Integration
	err := tx.
		Table("app_installations AS integration").
		Select("integration.*").
		Joins("JOIN vcs_provider_integration_bindings AS binding ON binding.integration_id = integration.id").
		Where("binding.organization_id = ?", organizationID).
		Where("binding.provider = ?", provider).
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

func FindVCSProviderIntegrationBinding(tx *gorm.DB, integrationID uuid.UUID) (*VCSProviderIntegrationBinding, error) {
	var binding VCSProviderIntegrationBinding
	if err := tx.Where("integration_id = ?", integrationID).First(&binding).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

func ListVCSProviderIntegrationBindings(
	tx *gorm.DB,
	provider string,
	installationID int64,
) ([]VCSProviderIntegrationBinding, error) {
	var bindings []VCSProviderIntegrationBinding
	err := tx.
		Where("provider = ? AND installation_id = ?", provider, installationID).
		Order("created_at ASC").
		Find(&bindings).
		Error
	return bindings, err
}

func ListVCSProviderBoundIntegrations(tx *gorm.DB, provider string, installationID int64) ([]Integration, error) {
	var integrations []Integration
	err := tx.
		Table("app_installations AS integration").
		Select("integration.*").
		Joins("JOIN vcs_provider_integration_bindings AS binding ON binding.integration_id = integration.id").
		Where("binding.provider = ?", provider).
		Where("binding.installation_id = ?", installationID).
		Where("integration.deleted_at IS NULL").
		Find(&integrations).
		Error
	return integrations, err
}
