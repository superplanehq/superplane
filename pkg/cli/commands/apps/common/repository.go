package common

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// CurrentUserDraftBranchName returns the default draft branch for the authenticated user.
func CurrentUserDraftBranchName(ctx core.CommandContext) (string, error) {
	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return "", err
	}
	userID := strings.TrimSpace(me.User.GetId())
	if userID == "" {
		return "", fmt.Errorf("current user id not found")
	}
	parsed, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid current user id: %w", err)
	}
	return materialize.DefaultDraftBranchName(parsed), nil
}

// EnsureCurrentUserDraftBranch creates the user's default draft branch when missing.
func EnsureCurrentUserDraftBranch(ctx core.CommandContext, canvasID string) (openapi_client.CanvasesCanvasDraftBranch, error) {
	branchName, err := CurrentUserDraftBranchName(ctx)
	if err != nil {
		return openapi_client.CanvasesCanvasDraftBranch{}, err
	}

	if existing, err := FindDraftBranch(ctx, canvasID, branchName); err == nil {
		return existing, nil
	}

	body := openapi_client.CanvasesCreateDraftBranchBody{}
	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesCreateDraftBranch(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return openapi_client.CanvasesCanvasDraftBranch{}, err
	}
	if response.Branch == nil {
		return openapi_client.CanvasesCanvasDraftBranch{}, fmt.Errorf("draft branch was not returned by the API")
	}
	return *response.Branch, nil
}

// FindDraftBranch returns a draft branch by name.
func FindDraftBranch(ctx core.CommandContext, canvasID, branchName string) (openapi_client.CanvasesCanvasDraftBranch, error) {
	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesListDraftBranches(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return openapi_client.CanvasesCanvasDraftBranch{}, err
	}

	for _, branch := range response.GetBranches() {
		if strings.TrimSpace(branch.GetBranchName()) == branchName {
			return branch, nil
		}
	}
	return openapi_client.CanvasesCanvasDraftBranch{}, fmt.Errorf("draft branch %q not found", branchName)
}

// FindCurrentUserDraftTipSHA returns the tip commit SHA for the user's default draft branch.
func FindCurrentUserDraftTipSHA(ctx core.CommandContext, canvasID string) (string, error) {
	branchName, err := CurrentUserDraftBranchName(ctx)
	if err != nil {
		return "", err
	}
	branch, err := FindDraftBranch(ctx, canvasID, branchName)
	if err != nil {
		return "", err
	}
	tipSHA := strings.TrimSpace(branch.GetTipSha())
	if tipSHA == "" {
		return "", fmt.Errorf("draft branch %q has no tip SHA", branchName)
	}
	return tipSHA, nil
}

// EnsureCurrentUserDraftTipSHA ensures a draft branch exists and returns its tip SHA.
func EnsureCurrentUserDraftTipSHA(ctx core.CommandContext, canvasID string) (string, error) {
	branch, err := EnsureCurrentUserDraftBranch(ctx, canvasID)
	if err != nil {
		return "", err
	}
	tipSHA := strings.TrimSpace(branch.GetTipSha())
	if tipSHA == "" {
		return "", fmt.Errorf("draft branch tip SHA was not returned by the API")
	}
	return tipSHA, nil
}

// CommitRepositoryFiles commits file operations to a draft branch.
func CommitRepositoryFiles(
	ctx core.CommandContext,
	canvasID string,
	branch string,
	expectedHeadSHA string,
	message string,
	operations []openapi_client.CanvasesCanvasRepositoryFileOperation,
) (string, error) {
	body := openapi_client.CanvasesCommitCanvasRepositoryFilesBody{}
	body.SetBranch(branch)
	if expectedHeadSHA != "" {
		body.SetExpectedHeadSha(expectedHeadSHA)
	}
	body.SetMessage(message)
	body.SetOperations(operations)

	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesCommitCanvasRepositoryFiles(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return "", err
	}
	commitSHA := strings.TrimSpace(response.GetCommitSha())
	if commitSHA == "" {
		return "", fmt.Errorf("commit succeeded but server did not return a commit SHA")
	}
	return commitSHA, nil
}

// PublishDraftBranch merges the draft branch to main when change management is disabled.
func PublishDraftBranch(ctx core.CommandContext, canvasID, draftBranch string) error {
	body := openapi_client.CanvasesPublishCanvasBody{}
	if draftBranch != "" {
		body.SetDraftBranch(draftBranch)
	}
	_, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesPublishCanvas(ctx.Context, canvasID).
		Body(body).
		Execute()
	return err
}

// FetchRepositoryFile downloads a file from the app git repository at the given branch or ref.
func FetchRepositoryFile(ctx core.CommandContext, canvasID, path, branch string) ([]byte, error) {
	config := ctx.API.GetConfig()
	if config == nil {
		return nil, fmt.Errorf("api client config is required")
	}

	baseURL, err := config.ServerURLWithContext(ctx.Context, "CanvasRepositoryAPIService.CanvasesListCanvasRepositoryFiles")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("api_url is required")
	}

	values := url.Values{}
	values.Set("path", path)
	if branch != "" {
		values.Set("branch", branch)
	}

	endpoint := fmt.Sprintf(
		"%s/api/v1/canvases/%s/repository/file?%s",
		strings.TrimRight(baseURL, "/"),
		url.PathEscape(canvasID),
		values.Encode(),
	)

	request, err := http.NewRequestWithContext(ctx.Context, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if authorization := strings.TrimSpace(config.DefaultHeader["Authorization"]); authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(body))
		if message != "" {
			return nil, fmt.Errorf("%s", message)
		}
		return nil, fmt.Errorf("failed to read repository file: %s", response.Status)
	}
	return body, nil
}
