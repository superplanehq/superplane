package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

// RepositorySeedFile stores a file extracted from an installable app
// repository so the canvas repository provisioner can commit it as part of
// the repository's initial content. Rows are deleted after a successful
// commit (see workers/repository_provisioner.go).
type RepositorySeedFile struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	RepositoryID uuid.UUID
	Path         string
	Content      []byte
	CreatedAt    time.Time
}

func NewRepositorySeedFile(repositoryID uuid.UUID, path string, content []byte) *RepositorySeedFile {
	return &RepositorySeedFile{
		RepositoryID: repositoryID,
		Path:         path,
		Content:      content,
		CreatedAt:    time.Now(),
	}
}

func (RepositorySeedFile) TableName() string {
	return "repository_seed_files"
}

func CreateRepositorySeedFiles(repositoryID uuid.UUID, files []RepositorySeedFile) error {
	return CreateRepositorySeedFilesInTransaction(database.Conn(), repositoryID, files)
}

func ListRepositorySeedFiles(repositoryID uuid.UUID) ([]RepositorySeedFile, error) {
	return ListRepositorySeedFilesInTransaction(database.Conn(), repositoryID)
}

func DeleteRepositorySeedFiles(repositoryID uuid.UUID) error {
	return DeleteRepositorySeedFilesInTransaction(database.Conn(), repositoryID)
}

func CreateRepositorySeedFilesInTransaction(tx *gorm.DB, repositoryID uuid.UUID, files []RepositorySeedFile) error {
	if len(files) == 0 {
		return nil
	}

	rows := make([]RepositorySeedFile, 0, len(files))
	now := time.Now()
	for _, file := range files {
		rows = append(rows, RepositorySeedFile{
			RepositoryID: repositoryID,
			Path:         file.Path,
			Content:      file.Content,
			CreatedAt:    now,
		})
	}

	return tx.Create(&rows).Error
}

func ListRepositorySeedFilesInTransaction(tx *gorm.DB, repositoryID uuid.UUID) ([]RepositorySeedFile, error) {
	var files []RepositorySeedFile
	err := tx.
		Where("repository_id = ?", repositoryID).
		Order("path ASC").
		Find(&files).
		Error
	if err != nil {
		return nil, err
	}

	return files, nil
}

func DeleteRepositorySeedFilesInTransaction(tx *gorm.DB, repositoryID uuid.UUID) error {
	return tx.
		Where("repository_id = ?", repositoryID).
		Delete(&RepositorySeedFile{}).
		Error
}
