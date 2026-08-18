package factories

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type artifactAddCommand struct {
	factory *string
	orderID *string
	typ     *string
	title   *string
	body    *string
	file    *string
	url     *string
	number  *int64
	name    *string
}

func (c *artifactAddCommand) Execute(ctx core.CommandContext) error {
	orderID := strings.TrimSpace(stringValue(c.orderID))
	artifactType := strings.ToLower(strings.TrimSpace(stringValue(c.typ)))

	if _, err := uuid.Parse(orderID); err != nil {
		return fmt.Errorf("--order-id must be a UUID")
	}

	factoryID, err := ResolveFactoryID(ctx, stringValue(c.factory))
	if err != nil {
		return err
	}

	data, protoType, err := c.buildData(ctx, artifactType)
	if err != nil {
		return err
	}

	body := openapi_client.NewFactoriesCreateWorkOrderArtifactBody()
	body.SetType(protoType)
	body.SetData(data)

	response, _, err := ctx.API.FactoryAPI.
		FactoriesCreateWorkOrderArtifact(ctx.Context, factoryID, orderID).
		Body(*body).
		Execute()
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

	case "pr":
		url := strings.TrimSpace(stringValue(c.url))
		if url == "" {
			return nil, "", fmt.Errorf("--url is required for type pr")
		}
		data["url"] = url
		if title := strings.TrimSpace(stringValue(c.title)); title != "" {
			data["title"] = title
		}
		if ctx.Cmd.Flags().Changed("number") {
			data["number"] = *c.number
		}
		return data, openapi_client.FACTORIESWORKORDERARTIFACTTYPE_TYPE_PR, nil

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
		return nil, "", fmt.Errorf("unknown artifact type %q (expected pr, markdown, branch, or link)", artifactType)
	}
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
