package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const APIBaseURL = "https://api.bitbucket.org/2.0"

// InstallationRef identifies one Forge install whose system token can list
// repositories. The token itself stays with the caller.
type InstallationRef struct {
	ID            string
	WorkspaceUUID string
	WorkspaceSlug string
}

// VisibleRepository is a Bitbucket repository the linked account can use.
type VisibleRepository struct {
	InstallationID string
	WorkspaceSlug  string
	FullName       string
	UUID           string
	DefaultBranch  string
	Private        bool
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Directory calls the Bitbucket REST API with a Forge system token.
type Directory struct {
	BaseURL string
	HTTP    httpDoer
}

var activeDirectory = Directory{
	BaseURL: APIBaseURL,
	HTTP:    &http.Client{Timeout: 20 * time.Second},
}

func CurrentDirectory() Directory {
	return activeDirectory
}

// UseDirectory points repository listing at another Bitbucket API. The
// returned function restores the previous directory.
func UseDirectory(directory Directory) func() {
	previous := activeDirectory
	activeDirectory = directory
	return func() {
		activeDirectory = previous
	}
}

type permissionPage struct {
	Values []permissionRecord `json:"values"`
	Next   string             `json:"next"`
}

type permissionRecord struct {
	Permission string `json:"permission"`
	Repository struct {
		UUID     string `json:"uuid"`
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// pushPermissions are the Bitbucket effective permissions that allow a push.
// GitHub keeps only push-capable collaborators; this is the same rule.
var pushPermissions = map[string]struct{}{
	"write": {},
	"admin": {},
}

type repositoryPage struct {
	Values []repositoryRecord `json:"values"`
	Next   string             `json:"next"`
}

type repositoryRecord struct {
	UUID       string  `json:"uuid"`
	FullName   string  `json:"full_name"`
	IsPrivate  bool    `json:"is_private"`
	Mainbranch *Branch `json:"mainbranch"`
}

// RepositoriesVisibleTo lists repositories from each installation where the
// Bitbucket account can push. A stale token is skipped. An API error is
// returned only when no installation could be read.
func (d Directory) RepositoriesVisibleTo(
	ctx context.Context,
	accountUUID string,
	installations []InstallationRef,
	tokenFor func(installationID string) (string, error),
) ([]VisibleRepository, error) {
	var repositories []VisibleRepository
	seen := map[string]struct{}{}
	sawRead := false
	var readErr error
	for _, installation := range installations {
		workspaceRef := strings.TrimSpace(installation.WorkspaceSlug)
		if workspaceRef == "" {
			workspaceRef = strings.TrimSpace(installation.WorkspaceUUID)
		}
		if installation.ID == "" || workspaceRef == "" {
			continue
		}
		token, err := tokenFor(installation.ID)
		if err != nil {
			continue
		}
		visible, err := d.VisibleRepositories(ctx, token, workspaceRef, accountUUID)
		if err != nil {
			readErr = err
			continue
		}
		sawRead = true
		for _, repository := range visible {
			repository.InstallationID = installation.ID
			if repository.WorkspaceSlug == "" {
				repository.WorkspaceSlug = workspaceRef
			}
			key := strings.ToLower(repository.FullName)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			repositories = append(repositories, repository)
		}
	}
	if !sawRead && readErr != nil {
		return nil, readErr
	}
	return repositories, nil
}

// VisibleRepositories reads the account's effective repository permissions in
// the workspace, then returns the repositories the account can push to.
// Workspace membership alone does not make a repository visible. An account
// without a push permission, or a workspace Bitbucket does not know, yields
// an empty list.
func (d Directory) VisibleRepositories(ctx context.Context, token, workspaceRef, accountUUID string) ([]VisibleRepository, error) {
	grants, err := d.listPushGrants(ctx, token, workspaceRef, accountUUID)
	if err != nil {
		return nil, err
	}
	if len(grants) == 0 {
		return nil, nil
	}
	slug := workspaceSlugFromGrants(grants, workspaceRef)
	records, err := d.listRepositories(ctx, token, slug)
	if err != nil {
		return nil, err
	}
	repositories := make([]VisibleRepository, 0, len(grants))
	for _, record := range records {
		fullName := strings.TrimSpace(record.FullName)
		if fullName == "" {
			continue
		}
		repositoryUUID, err := NormalizeAccountID(record.UUID)
		if err != nil {
			continue
		}
		if _, granted := grants[repositoryUUID]; !granted {
			continue
		}
		repositories = append(repositories, VisibleRepository{
			WorkspaceSlug: slug,
			FullName:      fullName,
			UUID:          repositoryUUID,
			DefaultBranch: defaultBranch(record.Mainbranch),
			Private:       record.IsPrivate,
		})
	}
	return repositories, nil
}

// listPushGrants returns the repositories, by uuid, where the account holds a
// push permission. Bitbucket reports effective permissions, so grants through
// groups and projects are included. A 404 means the workspace is unknown to
// this token and yields no grants.
func (d Directory) listPushGrants(ctx context.Context, token, workspaceRef, accountUUID string) (map[string]permissionRecord, error) {
	nextURL, err := d.permissionsURL(workspaceRef, accountUUID)
	if err != nil {
		return nil, err
	}
	grants := map[string]permissionRecord{}
	for nextURL != "" {
		var page permissionPage
		status, err := d.getJSON(ctx, token, nextURL, &page)
		if err != nil {
			return nil, err
		}
		if status == http.StatusNotFound {
			return nil, nil
		}
		for _, record := range page.Values {
			if _, ok := pushPermissions[strings.ToLower(strings.TrimSpace(record.Permission))]; !ok {
				continue
			}
			repositoryUUID, err := NormalizeAccountID(record.Repository.UUID)
			if err != nil {
				continue
			}
			grants[repositoryUUID] = record
		}
		nextURL = d.allowedNext(page.Next)
	}
	return grants, nil
}

// workspaceSlugFromGrants reads the workspace slug from a granted repository's
// full name. The installation may only know the workspace uuid.
func workspaceSlugFromGrants(grants map[string]permissionRecord, fallback string) string {
	for _, grant := range grants {
		slug, _, ok := strings.Cut(strings.TrimSpace(grant.Repository.FullName), "/")
		if ok && slug != "" {
			return slug
		}
	}
	return fallback
}

func defaultBranch(branch *Branch) string {
	if branch == nil || strings.TrimSpace(branch.Name) == "" {
		return "main"
	}
	return strings.TrimSpace(branch.Name)
}

func (d Directory) listRepositories(ctx context.Context, token, workspace string) ([]repositoryRecord, error) {
	nextURL, err := d.repositoryURL(workspace)
	if err != nil {
		return nil, err
	}
	var records []repositoryRecord
	for nextURL != "" {
		var page repositoryPage
		status, err := d.getJSON(ctx, token, nextURL, &page)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("bitbucket returned status %d", status)
		}
		records = append(records, page.Values...)
		nextURL = d.allowedNext(page.Next)
	}
	return records, nil
}

func (d Directory) getJSON(ctx context.Context, token, rawURL string, dest any) (int, error) {
	if d.HTTP == nil {
		return 0, fmt.Errorf("bitbucket http client is required")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	response, err := d.HTTP.Do(request)
	if err != nil {
		return 0, fmt.Errorf("call bitbucket: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return response.StatusCode, err
	}
	if response.StatusCode == http.StatusNotFound {
		return response.StatusCode, nil
	}
	if response.StatusCode != http.StatusOK {
		return response.StatusCode, fmt.Errorf("bitbucket returned status %d", response.StatusCode)
	}
	if dest != nil {
		if err := json.Unmarshal(body, dest); err != nil {
			return response.StatusCode, fmt.Errorf("decode bitbucket response: %w", err)
		}
	}
	return response.StatusCode, nil
}

// permissionsURL filters the workspace repository permissions to one account.
// The app bot may call this endpoint with read:repository:bitbucket.
func (d Directory) permissionsURL(workspaceRef, accountUUID string) (string, error) {
	account, err := NormalizeAccountID(accountUUID)
	if err != nil {
		return "", err
	}
	workspace, err := workspacePath(workspaceRef)
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("q", fmt.Sprintf(`user.uuid="{%s}"`, account))
	query.Set("pagelen", "100")
	return fmt.Sprintf("%s/workspaces/%s/permissions/repositories?%s", strings.TrimRight(d.base(), "/"), workspace, query.Encode()), nil
}

func (d Directory) repositoryURL(workspace string) (string, error) {
	workspace, err := workspacePath(workspace)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/repositories/%s?pagelen=100", strings.TrimRight(d.base(), "/"), workspace), nil
}

func (d Directory) base() string {
	if strings.TrimSpace(d.BaseURL) == "" {
		return APIBaseURL
	}
	return d.BaseURL
}

func (d Directory) allowedNext(next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return ""
	}
	parsed, err := url.Parse(next)
	if err != nil {
		return ""
	}
	base, err := url.Parse(d.base())
	if err != nil {
		return ""
	}
	if parsed.Host == "" {
		return base.ResolveReference(parsed).String()
	}
	if !strings.EqualFold(parsed.Host, base.Host) {
		return ""
	}
	return parsed.String()
}

// NormalizeAccountID accepts a Bitbucket UUID with or without braces.
func NormalizeAccountID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.Trim(strings.TrimSpace(value), "{}"))
	if err != nil || parsed == uuid.Nil {
		return "", fmt.Errorf("invalid bitbucket account id")
	}
	return parsed.String(), nil
}

func workspacePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if normalized, err := NormalizeAccountID(value); err == nil {
		return url.PathEscape("{" + normalized + "}"), nil
	}
	if value == "" {
		return "", fmt.Errorf("bitbucket workspace is required")
	}
	return url.PathEscape(value), nil
}
