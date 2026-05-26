package canvases

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const gitCredentialHelperCommand = "!superplane canvases git-credential-helper"

type repositoryGitURLCommand struct{}

func (c *repositoryGitURLCommand) Execute(ctx core.CommandContext) error {
	repository, err := getCanvasRepository(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	gitURL := strings.TrimSpace(repository.GetGitUrl())
	if gitURL == "" {
		return fmt.Errorf("canvas repository does not expose a Git URL")
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintln(stdout, gitURL)
			return err
		})
	}

	return ctx.Renderer.Render(map[string]string{"gitUrl": gitURL})
}

type repositoryCredentialHelperCommand struct {
	readOnly       *bool
	ttlSeconds     *int64
	allowForcePush *bool
}

func (c *repositoryCredentialHelperCommand) Execute(ctx core.CommandContext) error {
	operation := "get"
	if len(ctx.Args) > 0 {
		operation = strings.TrimSpace(ctx.Args[0])
	}

	if operation != "get" {
		return nil
	}

	canvasID, err := canvasIDFromGitCredentialInput(ctx.Cmd.InOrStdin())
	if err != nil {
		return err
	}

	credentials, err := generateCanvasRepositoryCredentials(
		ctx,
		canvasID,
		boolValue(c.readOnly),
		int64Value(c.ttlSeconds),
		boolValue(c.allowForcePush),
	)
	if err != nil {
		return err
	}

	username := strings.TrimSpace(credentials.GetUsername())
	password := strings.TrimSpace(credentials.GetPassword())
	if username == "" || password == "" {
		return fmt.Errorf("server returned empty Git credentials")
	}

	_, _ = fmt.Fprintf(ctx.Cmd.OutOrStdout(), "username=%s\n", username)
	_, _ = fmt.Fprintf(ctx.Cmd.OutOrStdout(), "password=%s\n\n", password)
	return nil
}

func getCanvasRepository(ctx core.CommandContext, nameOrID string) (*openapi_client.CanvasesCanvasRepository, error) {
	canvasID, err := findCanvasID(ctx, ctx.API, nameOrID)
	if err != nil {
		return nil, err
	}

	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesGetCanvasRepository(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return nil, err
	}

	if response == nil || response.Repository == nil {
		return nil, fmt.Errorf("canvas repository %q not found", canvasID)
	}

	return response.Repository, nil
}

func generateCanvasRepositoryCredentials(
	ctx core.CommandContext,
	canvasID string,
	readOnly bool,
	ttlSeconds int64,
	allowForcePush bool,
) (*openapi_client.CanvasesGenerateCanvasRepositoryCredentialsResponse, error) {
	body := openapi_client.CanvasesGenerateCanvasRepositoryCredentialsBody{
		ReadOnly:       openapi_client.PtrBool(readOnly),
		AllowForcePush: openapi_client.PtrBool(allowForcePush),
	}
	if ttlSeconds > 0 {
		body.TtlSeconds = openapi_client.PtrString(strconv.FormatInt(ttlSeconds, 10))
	}

	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesGenerateCanvasRepositoryCredentials(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, fmt.Errorf("server returned empty Git credentials response")
	}

	return response, nil
}

func canvasIDFromGitCredentialInput(input io.Reader) (string, error) {
	values := map[string]string{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			break
		}

		key, value, ok := strings.Cut(line, "=")
		if ok {
			values[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	rawPath := strings.TrimSpace(values["path"])
	if rawPath == "" {
		return "", fmt.Errorf("Git credential request is missing path; set credential.useHttpPath=true")
	}

	unescapedPath, err := url.PathUnescape(rawPath)
	if err != nil {
		return "", fmt.Errorf("invalid Git credential path: %w", err)
	}

	segments := strings.Split(strings.Trim(unescapedPath, "/"), "/")
	for i, segment := range segments {
		if segment != "canvases" || i+1 >= len(segments) {
			continue
		}

		canvasID := strings.TrimSuffix(segments[i+1], ".git")
		if _, err := uuid.Parse(canvasID); err != nil {
			return "", fmt.Errorf("invalid canvas id in Git credential path")
		}

		return canvasID, nil
	}

	return "", fmt.Errorf("Git credential path does not contain a canvas id")
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}

	return *value
}
