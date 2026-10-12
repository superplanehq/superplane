package licensing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// LicensePathEnv configures a license file, for example a mounted Kubernetes
// secret. When it is set, the file is the only license source.
const LicensePathEnv = "SUPERPLANE_LICENSE_PATH"

type SourceKind string

const (
	SourceNone     SourceKind = "none"
	SourceFile     SourceKind = "file"
	SourceDatabase SourceKind = "database"
)

// ErrNotInstalled tells the service that a source has no license. It is not a
// verification failure.
var ErrNotInstalled = errors.New("no license installed")

// Source supplies a raw license artifact.
type Source interface {
	Kind() SourceKind
	Read(ctx context.Context) ([]byte, error)
}

// WritableSource is a Source that installation administrators can change.
type WritableSource interface {
	Source
	Write(ctx context.Context, raw []byte, installedBy uuid.UUID) error
	Clear(ctx context.Context) error
}

type FileSource struct {
	path string
}

func NewFileSource(path string) *FileSource {
	return &FileSource{path: path}
}

func (f *FileSource) Kind() SourceKind {
	return SourceFile
}

func (f *FileSource) Read(_ context.Context) ([]byte, error) {
	file, err := os.Open(f.path)
	if err != nil {
		return nil, fmt.Errorf("open license file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, MaxLicenseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read license file: %w", err)
	}

	if len(data) > MaxLicenseBytes {
		return nil, invalid(ReasonMalformed)
	}

	return data, nil
}

// licenseAssociatedData binds the ciphertext to its purpose so another
// encrypted value cannot be substituted for the license.
var licenseAssociatedData = []byte("superplane:installation-license:v1")

// DatabaseSource stores the license encrypted at rest with the installation
// encryptor.
type DatabaseSource struct {
	encryptor crypto.Encryptor
}

func NewDatabaseSource(encryptor crypto.Encryptor) *DatabaseSource {
	return &DatabaseSource{encryptor: encryptor}
}

func (d *DatabaseSource) Kind() SourceKind {
	return SourceDatabase
}

func (d *DatabaseSource) Read(ctx context.Context) ([]byte, error) {
	record, err := models.FindInstallationLicense(database.DB(ctx))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotInstalled
	}

	if err != nil {
		return nil, fmt.Errorf("find installation license: %w", err)
	}

	raw, err := d.encryptor.Decrypt(ctx, record.EncryptedLicense, licenseAssociatedData)
	if err != nil {
		return nil, invalid(ReasonUnreadable)
	}

	return raw, nil
}

func (d *DatabaseSource) Write(ctx context.Context, raw []byte, installedBy uuid.UUID) error {
	encrypted, err := d.encryptor.Encrypt(ctx, raw, licenseAssociatedData)
	if err != nil {
		return fmt.Errorf("encrypt installation license: %w", err)
	}

	return models.SaveInstallationLicense(database.DB(ctx), encrypted, &installedBy)
}

func (d *DatabaseSource) Clear(ctx context.Context) error {
	return models.DeleteInstallationLicense(database.DB(ctx))
}
