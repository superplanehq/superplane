package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const (
	CanvasYAMLPath  = "canvas.yaml"
	ConsoleYAMLPath = "console.yaml"
)

var ErrFileNotFound = errors.New("file not found")
var ErrFileDeleted = errors.New("file deleted")

func NormalizePath(path string) string {
	return strings.TrimLeft(strings.TrimSpace(strings.ReplaceAll(path, "\\", "/")), "/")
}

type AppFileReader struct {
	db     *gorm.DB
	app    *models.Canvas
	userID uuid.UUID
}

func IsSpecFilePath(path string) bool {
	return path == CanvasYAMLPath || path == ConsoleYAMLPath
}

func NewAppFileReader(db *gorm.DB, canvas *models.Canvas, userID uuid.UUID) *AppFileReader {
	return &AppFileReader{db: db, app: canvas, userID: userID}
}

func (r *AppFileReader) Read(ctx context.Context, path string) (reader io.ReadCloser, err error) {
	ctx, done := telemetry.Span(ctx, "reader.read")
	defer done(&err)

	//
	// Read from staging first.
	//
	reader, err = r.ReadFromStaging(ctx, path)
	if err == nil {
		return reader, nil
	}

	if errors.Is(err, ErrFileDeleted) {
		return nil, ErrFileDeleted
	}

	if errors.Is(err, ErrFileNotFound) {
		return r.ReadFromVersion(ctx, path, *r.app.LiveVersionID)
	}

	return nil, err
}

func (r *AppFileReader) ReadFromVersion(ctx context.Context, path string, versionID uuid.UUID) (reader io.ReadCloser, err error) {
	ctx, done := telemetry.Span(ctx, "reader.for_version")
	defer done(&err)

	v, err := models.FindCanvasVersionInTransaction(r.db, r.app.ID, versionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}

		return nil, fmt.Errorf("failed to find canvas version: %w", err)
	}

	path = NormalizePath(path)
	if !IsSpecFilePath(path) {
		return nil, ErrFileNotFound
	}

	return r.readSpecFromVersion(ctx, path, v)
}

func (r *AppFileReader) readSpecFromVersion(ctx context.Context, path string, version *models.CanvasVersion) (reader io.ReadCloser, err error) {
	ctx, done := telemetry.Span(ctx, "reader.spec_for_version")
	defer done(&err)

	var content string
	switch path {
	case CanvasYAMLPath:
		raw, err := yaml.VersionToCanvasYAML(r.app.Name, r.app.Description, version)
		if err != nil {
			return nil, err
		}

		content = string(raw)
	default:
		raw, err := yaml.VersionToConsoleYML(r.app.Name, version)
		if err != nil {
			return nil, err
		}

		content = string(raw)
	}

	return io.NopCloser(strings.NewReader(content)), nil
}

func (r *AppFileReader) ReadFromStaging(ctx context.Context, path string) (reader io.ReadCloser, err error) {
	ctx, done := telemetry.Span(ctx, "reader.from_staging")
	defer done(&err)

	path = NormalizePath(path)
	file, err := models.FindStagedFileForUser(r.db, r.app.ID, r.userID, path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}

		return nil, fmt.Errorf("failed to find staged file: %w", err)
	}

	if file.Deleted {
		return nil, ErrFileDeleted
	}

	return io.NopCloser(strings.NewReader(file.Content)), nil
}
