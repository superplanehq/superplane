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

type VCSProviderRepositorySyncPriority int16

const (
	VCSProviderRepositorySyncPriorityBackground  VCSProviderRepositorySyncPriority = 0
	VCSProviderRepositorySyncPriorityInteractive VCSProviderRepositorySyncPriority = 100
)

type VCSProviderRepositorySyncJob struct {
	Provider     string `gorm:"primaryKey"`
	RepositoryID int64  `gorm:"primaryKey"`
	RunAt        time.Time
	Priority     VCSProviderRepositorySyncPriority
	Attempts     int
	LockedAt     *time.Time
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (VCSProviderRepositorySyncJob) TableName() string {
	return "vcs_provider_repository_sync_jobs"
}

type VCSProviderInstallationReconcileJob struct {
	Provider       string `gorm:"primaryKey"`
	InstallationID int64  `gorm:"primaryKey"`
	RunAt          time.Time
	Attempts       int
	LockedAt       *time.Time
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (VCSProviderInstallationReconcileJob) TableName() string {
	return "vcs_provider_installation_reconcile_jobs"
}

type VCSProviderInstallationReconcileRequester struct {
	Provider       string    `gorm:"primaryKey"`
	InstallationID int64     `gorm:"primaryKey"`
	OrganizationID uuid.UUID `gorm:"primaryKey"`
}

func (VCSProviderInstallationReconcileRequester) TableName() string {
	return "vcs_provider_installation_reconcile_requesters"
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

type VCSProviderIntegrationRepository struct {
	IntegrationID uuid.UUID `gorm:"primaryKey"`
	Provider      string    `gorm:"primaryKey"`
	RepositoryID  int64     `gorm:"primaryKey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (VCSProviderIntegrationRepository) TableName() string {
	return "vcs_provider_integration_repositories"
}

type AccessibleVCSProviderRepository struct {
	VCSProviderRepository
	AccountLogin string
	AccountType  string
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

func DeleteVCSProviderInstallationsMissingFromSnapshot(
	tx *gorm.DB,
	provider string,
	installationIDs []int64,
	observedBefore time.Time,
) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}

	query := tx.Where("provider = ? AND updated_at <= ?", provider, observedBefore)
	if len(installationIDs) > 0 {
		query = query.Where("installation_id NOT IN ?", installationIDs)
	}
	return query.Delete(&VCSProviderInstallation{}).Error
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

func VCSProviderCatalogSynchronizing(
	tx *gorm.DB,
	provider string,
	providerUserID int64,
	organizationID uuid.UUID,
) (bool, error) {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return false, err
	}

	var synchronizing bool
	err = tx.Raw(`
		WITH relevant_installations AS (
			SELECT installation.provider, installation.installation_id
			FROM vcs_provider_installations AS installation
			WHERE installation.provider = ?
				AND LOWER(installation.account_type) = 'user'
				AND installation.account_id = ?
			UNION
			SELECT repository.provider, repository.installation_id
			FROM vcs_provider_repositories AS repository
			JOIN vcs_provider_repository_collaborators AS collaborator
				ON collaborator.provider = repository.provider
				AND collaborator.repository_id = repository.repository_id
			WHERE repository.provider = ?
				AND collaborator.provider_user_id = ?
		)
		SELECT EXISTS (
			SELECT 1
			FROM vcs_provider_repository_sync_jobs AS job
			JOIN vcs_provider_repositories AS repository
				ON repository.provider = job.provider
				AND repository.repository_id = job.repository_id
			JOIN relevant_installations AS relevant
				ON relevant.provider = repository.provider
				AND relevant.installation_id = repository.installation_id
			WHERE job.provider = ?
			UNION ALL
			SELECT 1
			FROM vcs_provider_installation_reconcile_jobs AS job
			WHERE job.provider = ?
				AND (
					EXISTS (
						SELECT 1
						FROM relevant_installations AS relevant
						WHERE relevant.provider = job.provider
							AND relevant.installation_id = job.installation_id
					)
					OR EXISTS (
						SELECT 1
						FROM vcs_provider_installation_reconcile_requesters AS requester
						WHERE requester.provider = job.provider
							AND requester.installation_id = job.installation_id
							AND requester.organization_id = ?
					)
				)
		)
	`, provider, providerUserID, provider, providerUserID, provider, provider, organizationID).Scan(&synchronizing).Error
	return synchronizing, err
}

func EnqueueVCSProviderInstallationReconciliation(
	tx *gorm.DB,
	provider string,
	installationID int64,
	organizationID uuid.UUID,
	runAt time.Time,
) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}
	if installationID <= 0 {
		return errors.New("VCS provider installation id is required")
	}

	job := VCSProviderInstallationReconcileJob{
		Provider:       provider,
		InstallationID: installationID,
		RunAt:          runAt,
	}
	updatedAt := time.Now()
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "provider"}, {Name: "installation_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"run_at": gorm.Expr(
					`CASE
					WHEN vcs_provider_installation_reconcile_jobs.locked_at IS NULL
						THEN LEAST(vcs_provider_installation_reconcile_jobs.run_at, ?)
					WHEN vcs_provider_installation_reconcile_jobs.updated_at <= vcs_provider_installation_reconcile_jobs.locked_at
						THEN ?
					ELSE LEAST(vcs_provider_installation_reconcile_jobs.run_at, ?)
				END`,
					runAt,
					runAt,
					runAt,
				),
				"last_error": "",
				"updated_at": gorm.Expr(
					"GREATEST(?, COALESCE(vcs_provider_installation_reconcile_jobs.locked_at + INTERVAL '1 microsecond', ?))",
					updatedAt,
					updatedAt,
				),
			}),
		}).Create(&job).Error; err != nil {
			return err
		}
		if organizationID == uuid.Nil {
			return nil
		}
		requester := VCSProviderInstallationReconcileRequester{
			Provider:       provider,
			InstallationID: installationID,
			OrganizationID: organizationID,
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&requester).Error
	})
}

func ClaimVCSProviderInstallationReconciliation(
	tx *gorm.DB,
	provider string,
	now, claimableBefore time.Time,
) (*VCSProviderInstallationReconcileJob, error) {
	var claimed *VCSProviderInstallationReconcileJob
	err := tx.Transaction(func(tx *gorm.DB) error {
		var job VCSProviderInstallationReconcileJob
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

func CompleteVCSProviderInstallationReconciliation(
	tx *gorm.DB,
	provider string,
	installationID int64,
	claimedAt time.Time,
) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where(
				"provider = ? AND installation_id = ? AND locked_at = ? AND updated_at <= ?",
				provider,
				installationID,
				claimedAt,
				claimedAt,
			).
			Delete(&VCSProviderInstallationReconcileJob{})
		if result.Error != nil || result.RowsAffected > 0 {
			return result.Error
		}
		return tx.Model(&VCSProviderInstallationReconcileJob{}).
			Where("provider = ? AND installation_id = ? AND locked_at = ?", provider, installationID, claimedAt).
			Update("locked_at", nil).
			Error
	})
}

func RetryVCSProviderInstallationReconciliation(
	tx *gorm.DB,
	provider string,
	installationID int64,
	claimedAt, runAt time.Time,
	reconcileError error,
) error {
	message := ""
	if reconcileError != nil {
		message = reconcileError.Error()
	}
	return tx.Model(&VCSProviderInstallationReconcileJob{}).
		Where("provider = ? AND installation_id = ? AND locked_at = ?", provider, installationID, claimedAt).
		Updates(map[string]any{
			"run_at":     gorm.Expr("CASE WHEN updated_at > ? THEN run_at ELSE ? END", claimedAt, runAt),
			"locked_at":  nil,
			"last_error": gorm.Expr("CASE WHEN updated_at > ? THEN last_error ELSE ? END", claimedAt, message),
			"updated_at": time.Now(),
		}).
		Error
}

func EnqueueVCSProviderReconciliation(tx *gorm.DB, provider string, runAt time.Time) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}
	job := VCSProviderReconcileJob{Provider: provider, RunAt: runAt}
	updatedAt := time.Now()
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}},
		DoUpdates: clause.Assignments(map[string]any{
			"run_at":     runAt,
			"last_error": "",
			"updated_at": gorm.Expr(
				"GREATEST(?, COALESCE(vcs_provider_reconcile_jobs.locked_at + INTERVAL '1 microsecond', ?))",
				updatedAt,
				updatedAt,
			),
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

func CompleteVCSProviderReconciliation(tx *gorm.DB, provider string, claimedAt time.Time) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where("provider = ? AND locked_at = ? AND updated_at <= ?", provider, claimedAt, claimedAt).
			Delete(&VCSProviderReconcileJob{})
		if result.Error != nil || result.RowsAffected > 0 {
			return result.Error
		}
		return tx.Model(&VCSProviderReconcileJob{}).
			Where("provider = ? AND locked_at = ?", provider, claimedAt).
			Update("locked_at", nil).
			Error
	})
}

func RetryVCSProviderReconciliation(
	tx *gorm.DB,
	provider string,
	claimedAt, runAt time.Time,
	reconcileError error,
) error {
	message := ""
	if reconcileError != nil {
		message = reconcileError.Error()
	}
	return tx.Model(&VCSProviderReconcileJob{}).
		Where("provider = ? AND locked_at = ?", provider, claimedAt).
		Updates(map[string]any{
			"run_at":     gorm.Expr("CASE WHEN updated_at > ? THEN run_at ELSE ? END", claimedAt, runAt),
			"locked_at":  nil,
			"last_error": gorm.Expr("CASE WHEN updated_at > ? THEN last_error ELSE ? END", claimedAt, message),
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

func FindAccessibleVCSProviderRepositoryByName(
	tx *gorm.DB,
	provider string,
	providerUserID int64,
	fullName string,
) (*AccessibleVCSProviderRepository, error) {
	var repository AccessibleVCSProviderRepository
	err := tx.
		Table("vcs_provider_repositories AS repository").
		Select("repository.*, installation.account_login, installation.account_type").
		Joins("JOIN vcs_provider_repository_collaborators AS collaborator ON collaborator.provider = repository.provider AND collaborator.repository_id = repository.repository_id").
		Joins("JOIN vcs_provider_installations AS installation ON installation.provider = repository.provider AND installation.installation_id = repository.installation_id").
		Where("repository.provider = ?", provider).
		Where("collaborator.provider_user_id = ?", providerUserID).
		Where("LOWER(repository.full_name) = LOWER(?)", strings.TrimSpace(fullName)).
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
	_, err := ConsumeVCSProviderInstallRequestsForAccount(tx, provider, accountID, accountLogin)
	return err
}

func ConsumeVCSProviderInstallRequestsForAccount(
	tx *gorm.DB,
	provider string,
	accountID *int64,
	accountLogin string,
) (bool, error) {
	query := tx.Model(&VCSProviderInstallRequest{})
	if accountID != nil {
		result := query.Where("provider = ? AND account_id = ?", provider, *accountID).Delete(&VCSProviderInstallRequest{})
		return result.RowsAffected > 0, result.Error
	}
	accountLogin = strings.TrimSpace(accountLogin)
	if accountLogin == "" {
		return false, nil
	}
	result := query.
		Where("provider = ? AND LOWER(account_login) = LOWER(?)", provider, accountLogin).
		Delete(&VCSProviderInstallRequest{})
	return result.RowsAffected > 0, result.Error
}

func EnqueueVCSProviderRepositorySync(
	tx *gorm.DB,
	provider string,
	repositoryID int64,
	runAt time.Time,
	priority VCSProviderRepositorySyncPriority,
) error {
	return enqueueVCSProviderRepositorySync(
		tx,
		provider,
		repositoryID,
		runAt,
		priority,
		vcsProviderRepositoryScheduleEarliest,
	)
}

func DelayVCSProviderRepositorySync(
	tx *gorm.DB,
	provider string,
	repositoryID int64,
	runAt time.Time,
	priority VCSProviderRepositorySyncPriority,
) error {
	return enqueueVCSProviderRepositorySync(
		tx,
		provider,
		repositoryID,
		runAt,
		priority,
		vcsProviderRepositoryScheduleDelayed,
	)
}

type vcsProviderRepositorySchedule int

const (
	vcsProviderRepositoryScheduleEarliest vcsProviderRepositorySchedule = iota
	vcsProviderRepositoryScheduleDelayed
)

func enqueueVCSProviderRepositorySync(
	tx *gorm.DB,
	provider string,
	repositoryID int64,
	runAt time.Time,
	priority VCSProviderRepositorySyncPriority,
	schedule vcsProviderRepositorySchedule,
) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}
	job := VCSProviderRepositorySyncJob{
		Provider:     provider,
		RepositoryID: repositoryID,
		RunAt:        runAt,
		Priority:     priority,
	}
	updatedAt := time.Now()
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}, {Name: "repository_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"run_at": repositorySyncRunAtExpression(schedule, runAt),
			"priority": gorm.Expr(
				`CASE
					WHEN vcs_provider_repository_sync_jobs.locked_at IS NOT NULL
						AND vcs_provider_repository_sync_jobs.updated_at <= vcs_provider_repository_sync_jobs.locked_at
						THEN ?
					ELSE GREATEST(vcs_provider_repository_sync_jobs.priority, ?)
				END`,
				priority,
				priority,
			),
			"last_error": "",
			"updated_at": gorm.Expr(
				"GREATEST(?, COALESCE(vcs_provider_repository_sync_jobs.locked_at + INTERVAL '1 microsecond', ?))",
				updatedAt,
				updatedAt,
			),
		}),
	}).Create(&job).Error
}

func repositorySyncRunAtExpression(schedule vcsProviderRepositorySchedule, runAt time.Time) clause.Expr {
	if schedule == vcsProviderRepositoryScheduleDelayed {
		return gorm.Expr(
			`CASE
				WHEN vcs_provider_repository_sync_jobs.locked_at IS NULL
					THEN GREATEST(vcs_provider_repository_sync_jobs.run_at, ?)
				WHEN vcs_provider_repository_sync_jobs.updated_at <= vcs_provider_repository_sync_jobs.locked_at
					THEN ?
				ELSE GREATEST(vcs_provider_repository_sync_jobs.run_at, ?)
			END`,
			runAt,
			runAt,
			runAt,
		)
	}
	return gorm.Expr(
		`CASE
				WHEN vcs_provider_repository_sync_jobs.locked_at IS NULL
					THEN LEAST(vcs_provider_repository_sync_jobs.run_at, ?)
				WHEN vcs_provider_repository_sync_jobs.updated_at <= vcs_provider_repository_sync_jobs.locked_at
					THEN ?
				ELSE LEAST(vcs_provider_repository_sync_jobs.run_at, ?)
			END`,
		runAt,
		runAt,
		runAt,
	)
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
			Order("priority DESC, run_at ASC").
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

func CompleteVCSProviderRepositorySync(
	tx *gorm.DB,
	provider string,
	repositoryID int64,
	claimedAt time.Time,
) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where(
				"provider = ? AND repository_id = ? AND locked_at = ? AND updated_at <= ?",
				provider,
				repositoryID,
				claimedAt,
				claimedAt,
			).
			Delete(&VCSProviderRepositorySyncJob{})
		if result.Error != nil || result.RowsAffected > 0 {
			return result.Error
		}
		return tx.Model(&VCSProviderRepositorySyncJob{}).
			Where("provider = ? AND repository_id = ? AND locked_at = ?", provider, repositoryID, claimedAt).
			Update("locked_at", nil).
			Error
	})
}

func RetryVCSProviderRepositorySync(
	tx *gorm.DB,
	provider string,
	repositoryID int64,
	claimedAt, runAt time.Time,
	syncError error,
) error {
	message := ""
	if syncError != nil {
		message = syncError.Error()
	}
	return tx.Model(&VCSProviderRepositorySyncJob{}).
		Where("provider = ? AND repository_id = ? AND locked_at = ?", provider, repositoryID, claimedAt).
		Updates(map[string]any{
			"run_at":     gorm.Expr("CASE WHEN updated_at > ? THEN run_at ELSE ? END", claimedAt, runAt),
			"locked_at":  nil,
			"last_error": gorm.Expr("CASE WHEN updated_at > ? THEN last_error ELSE ? END", claimedAt, message),
			"updated_at": time.Now(),
		}).
		Error
}

func GrantVCSProviderBindingRepository(
	tx *gorm.DB,
	integrationID uuid.UUID,
	provider string,
	repositoryID int64,
) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}
	binding, err := FindVCSProviderIntegrationBinding(tx, integrationID)
	if err != nil {
		return err
	}
	if binding.Provider != provider {
		return fmt.Errorf("VCS provider binding uses provider %q", binding.Provider)
	}
	repository, err := FindVCSProviderRepository(tx, provider, repositoryID)
	if err != nil {
		return err
	}
	if repository.InstallationID != binding.InstallationID {
		return errors.New("VCS provider repository is not part of the bound installation")
	}

	grant := VCSProviderIntegrationRepository{
		IntegrationID: integrationID,
		Provider:      provider,
		RepositoryID:  repositoryID,
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&grant).Error
}

func ListVCSProviderBindingRepositories(tx *gorm.DB, integrationID uuid.UUID) ([]VCSProviderRepository, error) {
	var repositories []VCSProviderRepository
	err := tx.
		Table("vcs_provider_repositories AS repository").
		Select("repository.*").
		Joins("JOIN vcs_provider_integration_repositories AS access ON access.provider = repository.provider AND access.repository_id = repository.repository_id").
		Where("access.integration_id = ?", integrationID).
		Order("LOWER(repository.full_name) ASC").
		Find(&repositories).
		Error
	return repositories, err
}

// SyncVCSProviderBindingRepositories limits a binding to repositories selected
// by its active Factory workspaces. The transaction lock prevents concurrent
// workspace updates from replacing each other's grants.
func SyncVCSProviderBindingRepositories(tx *gorm.DB, integrationID uuid.UUID, provider string) error {
	provider, err := normalizeVCSProvider(provider)
	if err != nil {
		return err
	}
	binding, err := FindVCSProviderIntegrationBinding(tx, integrationID)
	if err != nil {
		return err
	}
	if binding.Provider != provider {
		return fmt.Errorf("VCS provider binding uses provider %q", binding.Provider)
	}

	return tx.Transaction(func(tx *gorm.DB) error {
		lockKey := fmt.Sprintf("vcs-provider-binding-repositories:%s", integrationID)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		var factories []Factory
		if err := tx.
			Select("onboarding_config").
			Where("deleted_at IS NULL").
			Where("onboarding_config ->> 'vcs_integration_id' = ?", integrationID.String()).
			Find(&factories).
			Error; err != nil {
			return err
		}

		selected := map[int64]struct{}{}
		for i := range factories {
			config := factories[i].OnboardingConfigValue()
			if config.AppRepositoryID > 0 {
				selected[config.AppRepositoryID] = struct{}{}
			}
			if config.BacklogRepositoryID > 0 {
				selected[config.BacklogRepositoryID] = struct{}{}
			}
		}

		if err := tx.Where("integration_id = ?", integrationID).Delete(&VCSProviderIntegrationRepository{}).Error; err != nil {
			return err
		}
		if len(selected) == 0 {
			return nil
		}

		repositoryIDs := make([]int64, 0, len(selected))
		for repositoryID := range selected {
			repositoryIDs = append(repositoryIDs, repositoryID)
		}
		var repositories []VCSProviderRepository
		if err := tx.
			Where("provider = ? AND installation_id = ? AND repository_id IN ?", provider, binding.InstallationID, repositoryIDs).
			Find(&repositories).
			Error; err != nil {
			return err
		}

		grants := make([]VCSProviderIntegrationRepository, 0, len(repositories))
		for _, repository := range repositories {
			grants = append(grants, VCSProviderIntegrationRepository{
				IntegrationID: integrationID,
				Provider:      provider,
				RepositoryID:  repository.RepositoryID,
			})
		}
		if len(grants) == 0 {
			return nil
		}
		return tx.Create(&grants).Error
	})
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

func normalizeVCSProvider(provider string) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return "", errors.New("VCS provider is required")
	}
	return provider, nil
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
