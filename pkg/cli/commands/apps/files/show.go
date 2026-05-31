package files

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ShowCommand struct{}

func (c *ShowCommand) Execute(ctx core.CommandContext) error {
	path, canvasTarget, err := c.parseArgs(ctx.Args)
	if err != nil {
		return err
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, canvasTarget)
	if err != nil {
		return err
	}

	content, err := c.fetchFile(ctx, canvasID, path)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"canvasId": canvasID,
			"path":     path,
			"content":  string(content),
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := stdout.Write(content)
		return err
	})
}

func (c *ShowCommand) parseArgs(args []string) (path string, canvasTarget string, err error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("path is required")
	}
	if len(args) > 2 {
		return "", "", fmt.Errorf("show accepts at most two positional arguments")
	}

	path = c.normalizePath(args[0])
	if path == "" {
		return "", "", fmt.Errorf("path is required")
	}

	if len(args) == 2 {
		canvasTarget = strings.TrimSpace(args[1])
	}

	return path, canvasTarget, nil
}

func (c *ShowCommand) normalizePath(path string) string {
	return strings.TrimLeft(strings.TrimSpace(strings.ReplaceAll(path, "\\", "/")), "/")
}

func (c *ShowCommand) fetchFile(ctx core.CommandContext, canvasID string, path string) ([]byte, error) {
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
