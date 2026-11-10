package tasks

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type artifactAddCommand struct {
	workspace *string
	taskID    *string
	typ       *string
	title     *string
	body      *string
	file      *string
	url       *string
	name      *string
	fileID    *string
}

// An uploaded file stays on the task when the artifact is rejected, so the
// CLI rejects a title that is too long before it uploads.
const maxFileArtifactTitleBytes = 1024

func (c *artifactAddCommand) Execute(ctx core.CommandContext) error {
	rawTaskID := taskRef(c.taskID)
	artifactType := strings.ToLower(strings.TrimSpace(stringValue(c.typ)))

	if rawTaskID == "" {
		return errTaskRequired
	}

	workspaceID, err := resolveWorkspace(ctx, c.workspace)
	if err != nil {
		return err
	}

	taskID, err := resolveTaskID(rawTaskID)
	if err != nil {
		return err
	}

	var (
		data       map[string]any
		protoType  openapi_client.FactoriesWorkOrderArtifactType
		uploadedID string
	)
	if artifactType == "file" {
		data, uploadedID, err = c.fileData(ctx, workspaceID, taskID)
		protoType = openapi_client.FACTORIESWORKORDERARTIFACTTYPE_TYPE_FILE
	} else {
		data, protoType, err = c.buildData(ctx, artifactType)
	}
	if err != nil {
		return err
	}

	body := openapi_client.NewFactoriesCreateWorkOrderArtifactBody()
	body.SetType(protoType)
	body.SetData(data)

	response, _, err := ctx.API.FactoryAPI.
		FactoriesCreateWorkOrderArtifact(ctx.Context, workspaceID, taskID).
		Body(*body).
		Execute()
	if err != nil && uploadedID != "" {
		return fmt.Errorf("attach uploaded file %s: %w (retry with --file-id %s to skip the upload)", uploadedID, err, uploadedID)
	}
	if err != nil {
		return err
	}

	artifact := response.GetArtifact()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(artifact)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		id := artifact.GetId()
		typ := string(artifact.GetType())
		_, err := fmt.Fprintf(stdout, "Artifact created: %s (%s)\n", id, typ)
		return err
	})
}

func (c *artifactAddCommand) buildData(
	ctx core.CommandContext,
	artifactType string,
) (map[string]any, openapi_client.FactoriesWorkOrderArtifactType, error) {
	data := map[string]any{}

	switch artifactType {
	case "markdown":
		body, err := c.resolveMarkdownBody(ctx)
		if err != nil {
			return nil, "", err
		}
		data["body"] = body
		if title := strings.TrimSpace(stringValue(c.title)); title != "" {
			data["title"] = title
		}
		return data, openapi_client.FACTORIESWORKORDERARTIFACTTYPE_TYPE_MARKDOWN, nil

	case "branch":
		name := strings.TrimSpace(stringValue(c.name))
		if name == "" {
			return nil, "", fmt.Errorf("--name is required for type branch")
		}
		data["name"] = name
		if url := strings.TrimSpace(stringValue(c.url)); url != "" {
			data["url"] = url
		}
		return data, openapi_client.FACTORIESWORKORDERARTIFACTTYPE_TYPE_BRANCH, nil

	case "link":
		url := strings.TrimSpace(stringValue(c.url))
		if url == "" {
			return nil, "", fmt.Errorf("--url is required for type link")
		}
		data["url"] = url
		if title := strings.TrimSpace(stringValue(c.title)); title != "" {
			data["title"] = title
		}
		return data, openapi_client.FACTORIESWORKORDERARTIFACTTYPE_TYPE_LINK, nil

	default:
		return nil, "", fmt.Errorf("unknown artifact type %q (expected markdown, branch, link, or file)", artifactType)
	}
}

// fileData returns the data of a file artifact. It uploads --file, or uses
// the file that --file-id names. uploadedID is set only after an upload.
func (c *artifactAddCommand) fileData(
	ctx core.CommandContext,
	workspaceID, taskID string,
) (data map[string]any, uploadedID string, err error) {
	path := strings.TrimSpace(stringValue(c.file))
	fileID := strings.TrimSpace(stringValue(c.fileID))
	if (path == "") == (fileID == "") || path == "-" {
		return nil, "", fmt.Errorf("pass --file or --file-id for type file")
	}
	title := strings.TrimSpace(stringValue(c.title))
	if len(title) > maxFileArtifactTitleBytes {
		return nil, "", fmt.Errorf("--title must be at most %d bytes", maxFileArtifactTitleBytes)
	}
	if fileID == "" {
		fileID, err = uploadTaskArtifactFile(ctx, workspaceID, taskID, path)
		if err != nil {
			return nil, "", err
		}
		uploadedID = fileID
	}
	data = map[string]any{"fileId": fileID}
	if title != "" {
		data["title"] = title
	}
	return data, uploadedID, nil
}

func (c *artifactAddCommand) resolveMarkdownBody(ctx core.CommandContext) (string, error) {
	bodyInline := stringValue(c.body)
	filePath := stringValue(c.file)

	if bodyInline != "" && filePath != "" {
		return "", fmt.Errorf("specify only one of --body or --file")
	}
	if bodyInline != "" {
		return bodyInline, nil
	}
	if filePath == "" {
		return "", fmt.Errorf("--body or --file is required for type markdown")
	}

	var (
		raw []byte
		err error
	)
	if filePath == "-" {
		raw, err = io.ReadAll(ctx.Cmd.InOrStdin())
	} else {
		raw, err = os.ReadFile(filePath) // #nosec G304 -- CLI user-provided path
	}
	if err != nil {
		return "", fmt.Errorf("read body file: %w", err)
	}
	body := string(raw)
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("markdown body is empty")
	}
	return body, nil
}
