package provider

import (
	"fmt"
	"path"
	"strings"
)

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

func ValidateCommitOperations(operations []FileOperation) ([]FileOperation, error) {
	if len(operations) == 0 {
		return nil, fmt.Errorf("%w: at least one file operation is required", ErrInvalidCommit)
	}

	normalized := make([]FileOperation, 0, len(operations))
	for _, operation := range operations {
		path, err := ValidateUserPath(operation.Path)
		if err != nil {
			return nil, err
		}
		operation.Path = path

		if operation.Delete {
			normalized = append(normalized, operation)
			continue
		}

		if operation.Content == nil {
			return nil, fmt.Errorf("%w: content is required for %q", ErrInvalidCommit, path)
		}

		if operation.SizeBytes < 0 {
			return nil, fmt.Errorf("%w: size is required for %q", ErrInvalidCommit, path)
		}

		normalized = append(normalized, operation)
	}

	return normalized, nil
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

	// SuperPlaneBotAuthorName and SuperPlaneBotAuthorEmail identify the
	// SuperPlane service in commits it authors on behalf of the platform
	// (initial repository, app install seeding, etc.).
	SuperPlaneBotAuthorName  = "SuperPlane"
	SuperPlaneBotAuthorEmail = "bot@superplane.local"
)

// SuperPlaneBotAuthor returns the canonical author used for commits made by
// SuperPlane itself rather than by an end user.
func SuperPlaneBotAuthor() CommitAuthor {
	return CommitAuthor{
		Name:  SuperPlaneBotAuthorName,
		Email: SuperPlaneBotAuthorEmail,
	}
}

func InitialRepositoryCommitOptions(branch string) CommitOptions {
	return CommitOptions{
		Branch:  DefaultBranch(branch),
		Message: initialRepositoryCommitMessage,
		Author:  SuperPlaneBotAuthor(),
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
