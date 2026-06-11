package drafts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	"google.golang.org/grpc/codes"
)

type deleteCommand struct{}

func (c *deleteCommand) Execute(ctx core.CommandContext) error {
	draftID := strings.TrimSpace(ctx.Args[0])

	appArg := ""
	if len(ctx.Args) == 2 {
		appArg = strings.TrimSpace(ctx.Args[1])
	}

	appID, err := common.ResolveAppNameOrIDArg(ctx, appArg)
	if err != nil {
		return err
	}

	_, _, err = ctx.API.CanvasVersionAPI.
		CanvasesDeleteCanvasVersion(ctx.Context, appID, draftID).
		Execute()
	if err != nil {
		return mapDeleteDraftError(err)
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "Draft deleted: %s\n", draftID)
			return err
		})
	}

	return ctx.Renderer.Render(map[string]string{
		"id":      draftID,
		"deleted": "true",
	})
}

func mapDeleteDraftError(err error) error {
	var apiErr *openapi_client.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		return err
	}

	status := extractRPCStatus(apiErr)
	if status == nil {
		return err
	}

	switch codes.Code(status.GetCode()) {
	case codes.PermissionDenied:
		return fmt.Errorf("you can only delete your own drafts")
	case codes.FailedPrecondition:
		return fmt.Errorf("only draft versions can be deleted")
	case codes.NotFound:
		return fmt.Errorf("draft not found")
	default:
		return core.FormatCommandError(err)
	}
}

func extractRPCStatus(apiErr *openapi_client.GenericOpenAPIError) *openapi_client.GooglerpcStatus {
	switch model := apiErr.Model().(type) {
	case *openapi_client.GooglerpcStatus:
		if model != nil {
			return model
		}
	case openapi_client.GooglerpcStatus:
		return &model
	}

	body := apiErr.Body()
	if len(body) == 0 {
		return nil
	}

	var decoded openapi_client.GooglerpcStatus
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil
	}
	return &decoded
}
