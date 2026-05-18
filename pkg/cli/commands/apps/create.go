package apps

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type createCommand struct {
	displayName *string
	appSlug     *string
	description *string
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	displayName := strings.TrimSpace(*c.displayName)
	if displayName == "" {
		return fmt.Errorf("--display-name is required")
	}

	appSlug := strings.TrimSpace(*c.appSlug)
	if appSlug == "" {
		return fmt.Errorf("--app-slug is required")
	}

	req := openapi_client.AppsCreateAppRequest{}
	req.SetDisplayName(displayName)
	req.SetAppSlug(appSlug)

	if c.description != nil {
		desc := strings.TrimSpace(*c.description)
		if desc != "" {
			req.SetDescription(desc)
		}
	}

	resp, httpResp, err := ctx.API.AppAPI.AppsCreateApp(ctx.Context).Body(req).Execute()
	return printCreateResponse(ctx, resp, httpResp, err)
}

func printCreateResponse(
	ctx core.CommandContext,
	resp *openapi_client.AppsCreateAppResponse,
	httpResp *http.Response,
	err error,
) error {
	if err != nil {
		return err
	}

	if httpResp != nil && (httpResp.StatusCode < 200 || httpResp.StatusCode >= 300) {
		return fmt.Errorf("unexpected response status: %s", httpResp.Status)
	}

	if resp == nil || resp.App == nil || resp.App.Metadata == nil || resp.App.Metadata.GetId() == "" {
		return fmt.Errorf("failed to create app: the server returned an empty response")
	}

	app := *resp.App
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(buildAppSummary(app))
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "App %q created (ID: %s, slug: %s)\n",
			app.Metadata.GetDisplayName(),
			app.Metadata.GetId(),
			app.Metadata.GetSlug(),
		)
		return err
	})
}
