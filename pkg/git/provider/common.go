package provider

import (
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
)

func ValidateRepositoryID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ReplaceAll(value, "\\", "/"), "/") {
		return "", ErrInvalidRepositoryID
	}

	normalized, err := NormalizePath(value)
	if err != nil {
		return "", ErrInvalidRepositoryID
	}

	segments := strings.Split(normalized, "/")
	if len(segments) != 4 || segments[0] != "orgs" || segments[2] != "canvases" {
		return "", ErrInvalidRepositoryID
	}

	if _, err := uuid.Parse(segments[1]); err != nil {
		return "", ErrInvalidRepositoryID
	}

	if _, err := uuid.Parse(segments[3]); err != nil {
		return "", ErrInvalidRepositoryID
	}

	return normalized, nil
}

func NormalizePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsRune(value, '\x00') {
		return "", ErrInvalidPath
	}

	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.TrimLeft(value, "/")
	if value == "" {
		return "", ErrInvalidPath
	}

	normalized := path.Clean(value)
	if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", ErrInvalidPath
	}

	for _, segment := range strings.Split(normalized, "/") {
		if segment == "" || segment == "." || segment == ".." || segment == ".git" {
			return "", ErrInvalidPath
		}
	}

	return normalized, nil
}

func ValidateUserPath(value string) (string, error) {
	normalized, err := NormalizePath(value)
	if err != nil {
		return "", err
	}

	if normalized == ReservedSuperPlanePath || strings.HasPrefix(normalized, ReservedSuperPlanePath+"/") {
		return "", ErrReservedPath
	}

	return normalized, nil
}

func ValidateCommitOperations(operations []FileOperation) error {
	if len(operations) == 0 {
		return fmt.Errorf("%w: at least one file operation is required", ErrInvalidCommit)
	}

	for _, operation := range operations {
		path, err := ValidateUserPath(operation.Path)
		if err != nil {
			return err
		}

		if operation.Delete {
			continue
		}

		if operation.Content == nil {
			return fmt.Errorf("%w: content is required for %q", ErrInvalidPath, path)
		}

		if operation.SizeBytes < 0 {
			return fmt.Errorf("%w: size is required for %q", ErrInvalidPath, path)
		}
	}

	return nil
}

func DefaultBranch(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "main"
	}
	return branch
}

func RefOrDefault(ref, branch string) string {
	ref = strings.TrimSpace(ref)
	if ref != "" {
		return ref
	}
	return DefaultBranch(branch)
}

//
// All repositories must be initialized with a README.md file.
//

const (
	initialRepositoryFilePath      = "README.md"
	initialRepositoryCommitMessage = "Initialize repository"
	initialRepositoryAuthorName    = "SuperPlane"
	initialRepositoryAuthorEmail   = "bot@superplane.local"
)

func InitialRepositoryCommitOptions(branch string) CommitOptions {
	return CommitOptions{
		Branch:  DefaultBranch(branch),
		Message: initialRepositoryCommitMessage,
		Author: CommitAuthor{
			Name:  initialRepositoryAuthorName,
			Email: initialRepositoryAuthorEmail,
		},
		Operations: []FileOperation{
			{
				Path:      initialRepositoryFilePath,
				Content:   strings.NewReader(""),
				SizeBytes: 0,
			},
		},
	}
}

func ValidateCommitMetadata(message string, author CommitAuthor) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("%w: commit message is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(author.Name) == "" {
		return fmt.Errorf("%w: author name is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(author.Email) == "" {
		return fmt.Errorf("%w: author email is required", ErrInvalidCommit)
	}
	return nil
}
