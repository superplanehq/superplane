package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/canvas/yaml"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	grpcactions "github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
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
	git    git.Provider
	app    *models.Canvas
	userID uuid.UUID
}

func IsSpecFilePath(path string) bool {
	return path == CanvasYAMLPath || path == ConsoleYAMLPath
}

func NewAppFileReader(db *gorm.DB, git git.Provider, canvas *models.Canvas, userID uuid.UUID) *AppFileReader {
	return &AppFileReader{db: db, git: git, app: canvas, userID: userID}
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

	//
	// Spec files (canvas.yaml, console.yaml) are not yet written to git,
	// so we still need to take them from the database, and convert to YAML here.
	//
	// NOTE: this should be removed once spec files are also written to git.
	//
	if IsSpecFilePath(path) {
		return r.readSpecFromVersion(ctx, path, v)
	}

	//
	// Arbitrary files are read directly from the git repository.
	// NOTE: once all versions point to git commits, we should use the commit SHA here.
	//
	return r.readFromGit(ctx, path)
}

func (r *AppFileReader) readSpecFromVersion(ctx context.Context, path string, version *models.CanvasVersion) (reader io.ReadCloser, err error) {
	ctx, done := telemetry.Span(ctx, "reader.spec_for_version")
	defer done(&err)

	var content string
	switch path {
	case CanvasYAMLPath:
		pbVersion := &pb.CanvasVersion{
			Spec: &pb.Canvas_Spec{
				Nodes: grpcactions.NodesToProto(version.Nodes),
				Edges: grpcactions.EdgesToProto(version.Edges),
			},
		}
		raw, err := yaml.CanvasResourceYAML(pbVersion, r.app.ID.String(), r.app.Name, r.app.Description)
		if err != nil {
			return nil, err
		}

		content = string(raw)
	default:
		raw, err := models.CanvasVersionToConsoleYML(r.app.Name, version)
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

func (r *AppFileReader) readFromGit(ctx context.Context, path string) (reader io.ReadCloser, err error) {
	ctx, done := telemetry.Span(ctx, "reader.read_from_git")
	defer done(&err)

	repository, err := models.FindRepository(r.app.OrganizationID, r.app.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}

		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	return r.git.GetFile(ctx, repository.RepoID, path, "")
}
