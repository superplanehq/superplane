package canvases

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type getCommand struct {
	draft *bool
}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := findCanvasID(ctx, ctx.API, ctx.Args[0])
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return err
	}
	if response.Canvas == nil {
		return fmt.Errorf("canvas %q not found", canvasID)
	}

	canvas := *response.Canvas
	if c.draft != nil && *c.draft {
		if canvas.Metadata == nil || !canvas.Metadata.GetVersioningEnabled() {
			return fmt.Errorf("--draft cannot be used when effective canvas versioning is disabled; remove --draft to get the live canvas directly")
		}

		me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
		if err != nil {
			return err
		}
		currentUserID := strings.TrimSpace(me.GetId())
		if currentUserID == "" {
			return fmt.Errorf("current user id not found")
		}

		versionID, err := findOwnedDraftVersionID(ctx, canvasID, currentUserID)
		if err != nil {
			return err
		}
		if versionID == "" {
			return fmt.Errorf("draft version not found for current user")
		}

		version, err := describeCanvasVersionByID(ctx, canvasID, versionID)
		if err != nil {
			return err
		}
		if version.Spec != nil {
			canvas.SetSpec(*version.Spec)
		}
	}

	resource := models.CanvasResourceFromCanvas(canvas)
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "ID: %s\n", resource.Metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", resource.Metadata.GetName())
		_, _ = fmt.Fprintf(stdout, "Nodes: %d\n", len(resource.Spec.GetNodes()))
		_, err := fmt.Fprintf(stdout, "Edges: %d\n", len(resource.Spec.GetEdges()))
		return err
	})
}

func findOwnedDraftVersionID(ctx core.CommandContext, canvasID string, userID string) (string, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return "", nil
	}

	var before *time.Time
	for {
		req := ctx.API.CanvasVersionAPI.
			CanvasesListCanvasVersions(ctx.Context, canvasID).
			Limit(50)
		if before != nil {
			req = req.Before(*before)
		}

		response, _, err := req.Execute()
		if err != nil {
			return "", err
		}

		for _, version := range response.GetVersions() {
			metadata := version.GetMetadata()
			if metadata.GetIsPublished() {
				continue
			}

			ownerID := ""
			if metadata.Owner != nil {
				ownerID = strings.TrimSpace(metadata.Owner.GetId())
			}
			if ownerID == "" || !strings.EqualFold(ownerID, trimmedUserID) {
				continue
			}

			versionID := strings.TrimSpace(metadata.GetId())
			if versionID == "" {
				continue
			}

			return versionID, nil
		}

		if !response.GetHasNextPage() {
			return "", nil
		}

		last, ok := response.GetLastTimestampOk()
		if !ok || last == nil {
			return "", nil
		}
		before = last
	}
}

func findCanvasID(ctx core.CommandContext, client *openapi_client.APIClient, nameOrID string) (string, error) {
	if _, err := uuid.Parse(nameOrID); err == nil {
		return nameOrID, nil
	}

	return findCanvasIDByName(ctx, client, nameOrID)
}

func findCanvasIDByName(ctx core.CommandContext, client *openapi_client.APIClient, name string) (string, error) {
	response, _, err := client.CanvasAPI.CanvasesListCanvases(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	var matches []openapi_client.CanvasesCanvas
	for _, canvas := range response.GetCanvases() {
		if canvas.Metadata == nil || canvas.Metadata.Name == nil {
			continue
		}
		if *canvas.Metadata.Name == name {
			matches = append(matches, canvas)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("canvas %q not found", name)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple canvases named %q found", name)
	}

	if matches[0].Metadata == nil || matches[0].Metadata.Id == nil {
		return "", fmt.Errorf("canvas %q is missing an id", name)
	}

	return *matches[0].Metadata.Id, nil
}
