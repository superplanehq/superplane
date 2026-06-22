package codestorage

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	codestorage "github.com/pierrecomputer/sdk/packages/code-storage-go"
	"github.com/superplanehq/superplane/pkg/git/provider"
)

type Provider struct {
	client        *codestorage.Client
	defaultBranch string
}

func NewProvider() (*Provider, error) {
	name := strings.TrimSpace(os.Getenv("GIT_STORAGE_CODE_STORAGE_NAME"))
	if name == "" {
		return nil, fmt.Errorf("GIT_STORAGE_CODE_STORAGE_NAME is required")
	}

	key, err := getPrivateKey()
	if err != nil {
		return nil, err
	}

	client, err := codestorage.NewClient(codestorage.Options{
		Name: name,
		Key:  string(key),
	})

	if err != nil {
		return nil, err
	}

	return &Provider{
		client:        client,
		defaultBranch: "main",
	}, nil
}

func getPrivateKey() ([]byte, error) {
	if key := strings.TrimSpace(os.Getenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY")); key != "" {
		return []byte(key), nil
	}

	privateKeyPath := strings.TrimSpace(os.Getenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH"))
	if privateKeyPath == "" {
		return nil, fmt.Errorf("either GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY or GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH are required")
	}

	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error reading private key for code storage: %w", err)
	}

	return key, nil
}

func (p *Provider) Name() string {
	return provider.CodeStorageProvider
}

func (p *Provider) CreateRepository(ctx context.Context, repoID string) (*provider.Repository, error) {
	repo, err := p.client.CreateRepo(ctx, codestorage.CreateRepoOptions{
		ID:            repoID,
		DefaultBranch: p.defaultBranch,
	})

	if err != nil {
		return nil, err
	}

	//
	// Initialize repository
	//
	initialCommit := provider.InitialRepositoryCommitOptions(p.defaultBranch)
	builder, err := repo.CreateCommit(codestorage.CommitOptions{
		TargetBranch:  p.defaultBranch,
		CommitMessage: initialCommit.Message,
		Author: codestorage.CommitSignature{
			Name:  initialCommit.Author.Name,
			Email: initialCommit.Author.Email,
		},
	})

	if err != nil {
		return nil, err
	}

	for _, operation := range initialCommit.Operations {
		builder.AddFile(operation.Path, operation.Content, nil)
	}

	if err := builder.Err(); err != nil {
		return nil, err
	}

	_, err = builder.Send(ctx)
	if err != nil {
		return nil, err
	}

	return &provider.Repository{
		ID: repo.ID,
	}, nil
}

func (p *Provider) DeleteRepository(ctx context.Context, repoID string) error {
	repo, err := p.client.FindOne(ctx, codestorage.FindOneOptions{ID: repoID})
	if err != nil {
		return err
	}

	//
	// If repo does not exist, do not do anything.
	//
	if repo == nil {
		return nil
	}

	_, err = p.client.DeleteRepo(ctx, codestorage.DeleteRepoOptions{ID: repoID})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) ListFiles(ctx context.Context, repoID, ref string) ([]string, error) {
	repo, err := p.repo(repoID)
	if err != nil {
		return nil, err
	}

	result, err := repo.ListFiles(ctx, codestorage.ListFilesOptions{
		Ref: provider.RefOrDefault(ref, p.defaultBranch),
	})

	if err != nil {
		return nil, err
	}

	return result.Paths, nil
}

func (p *Provider) GetFile(ctx context.Context, repoID, path, ref string) (io.ReadCloser, error) {
	filePath, err := provider.NormalizePath(path)
	if err != nil {
		return nil, err
	}

	repo, err := p.repo(repoID)
	if err != nil {
		return nil, err
	}

	resp, err := repo.FileStream(ctx, codestorage.GetFileOptions{
		Path: filePath,
		Ref:  provider.RefOrDefault(ref, p.defaultBranch),
	})

	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (p *Provider) Commit(ctx context.Context, repoID string, options provider.CommitOptions) (string, error) {
	if err := provider.ValidateCommitMetadata(options.Message, options.Author); err != nil {
		return "", err
	}

	operations, err := provider.ValidateCommitOperations(options.Operations)
	if err != nil {
		return "", err
	}

	repo, err := p.repo(repoID)
	if err != nil {
		return "", err
	}

	builder, err := repo.CreateCommit(codestorage.CommitOptions{
		TargetBranch:    provider.RefOrDefault(options.Branch, p.defaultBranch),
		BaseBranch:      strings.TrimSpace(options.BaseBranch),
		ExpectedHeadSHA: strings.TrimSpace(options.ExpectedHeadSHA),
		CommitMessage:   strings.TrimSpace(options.Message),
		Author: codestorage.CommitSignature{
			Name:  strings.TrimSpace(options.Author.Name),
			Email: strings.TrimSpace(options.Author.Email),
		},
	})

	if err != nil {
		return "", err
	}

	for _, operation := range operations {
		if operation.Delete {
			builder.DeletePath(operation.Path)
			continue
		}
		builder.AddFile(operation.Path, operation.Content, nil)
	}

	if err := builder.Err(); err != nil {
		return "", err
	}

	result, err := builder.Send(ctx)
	if err != nil {
		return "", err
	}

	return result.CommitSHA, nil
}

func (p *Provider) Head(ctx context.Context, repoID, ref string) (string, error) {
	repo, err := p.repo(repoID)
	if err != nil {
		return "", err
	}

	commit, err := repo.GetCommit(ctx, codestorage.GetCommitOptions{
		SHA: provider.RefOrDefault(ref, p.defaultBranch),
	})

	if err != nil {
		return "", err
	}

	return commit.Commit.SHA, nil
}

func (p *Provider) ListBranches(ctx context.Context, repoID, prefix string) ([]string, error) {
	repo, err := p.repo(repoID)
	if err != nil {
		return nil, err
	}

	var names []string
	cursor := ""
	for {
		result, err := repo.ListBranches(ctx, codestorage.ListBranchesOptions{
			Cursor: cursor,
			Limit:  100,
		})
		if err != nil {
			return nil, err
		}

		for _, branch := range result.Branches {
			if prefix == "" || strings.HasPrefix(branch.Name, prefix) {
				names = append(names, branch.Name)
			}
		}

		if !result.HasMore {
			break
		}
		cursor = result.NextCursor
	}

	sort.Strings(names)
	return names, nil
}

func (p *Provider) CreateBranch(ctx context.Context, repoID, branch, fromRef string) error {
	repo, err := p.repo(repoID)
	if err != nil {
		return err
	}

	_, err = repo.CreateBranch(ctx, codestorage.CreateBranchOptions{
		TargetBranch: strings.TrimSpace(branch),
		BaseRef:      provider.RefOrDefault(fromRef, p.defaultBranch),
	})
	return err
}

func (p *Provider) MergeBranch(ctx context.Context, repoID, sourceBranch, targetBranch, message string, author provider.CommitAuthor) (string, error) {
	if err := provider.ValidateCommitMetadata(message, author); err != nil {
		return "", err
	}

	repo, err := p.repo(repoID)
	if err != nil {
		return "", err
	}

	result, err := repo.Merge(ctx, codestorage.MergeOptions{
		SourceBranch:  strings.TrimSpace(sourceBranch),
		TargetBranch:  provider.RefOrDefault(targetBranch, p.defaultBranch),
		CommitMessage: strings.TrimSpace(message),
		Author: &codestorage.CommitSignature{
			Name:  strings.TrimSpace(author.Name),
			Email: strings.TrimSpace(author.Email),
		},
		Strategy: codestorage.MergeStrategyFFPrefer,
	})
	if err != nil {
		return "", err
	}

	return result.CommitSHA, nil
}

func (p *Provider) DeleteBranch(ctx context.Context, repoID, branch string) error {
	repo, err := p.repo(repoID)
	if err != nil {
		return err
	}

	_, err = repo.DeleteBranch(ctx, codestorage.DeleteBranchOptions{
		Name: strings.TrimSpace(branch),
	})
	return err
}

func (p *Provider) repo(repoID string) (*codestorage.Repo, error) {
	return p.client.Repo(codestorage.RepoOptions{
		ID:            strings.TrimSpace(repoID),
		DefaultBranch: p.defaultBranch,
	})
}

func (p *Provider) GetRepositoryID(options provider.RepositoryOptions) string {
	return fmt.Sprintf("orgs/%s/canvases/%s", options.OrganizationID.String(), options.CanvasID.String())
}
