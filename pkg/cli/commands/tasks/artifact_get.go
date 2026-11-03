package tasks

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type artifactGetCommand struct {
	workspace *string
	taskID    *string
	id        *string
	key       *string
}

func (c *artifactGetCommand) Execute(ctx core.CommandContext) error {
	id := strings.TrimSpace(stringValue(c.id))
	key := strings.TrimSpace(stringValue(c.key))
	if (id == "") == (key == "") {
		return fmt.Errorf("pass --id or --key")
	}
	rawTaskID := taskRef(c.taskID)
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

	response, _, err := ctx.API.FactoryAPI.
		FactoriesListWorkOrderArtifacts(ctx.Context, workspaceID, taskID).
		Execute()
	if err != nil {
		return err
	}
	artifact, ok := findArtifact(response.GetArtifacts(), id, key)
	if !ok {
		return fmt.Errorf("artifact not found")
	}
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(artifact)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderArtifactText(stdout, artifact)
	})
}

func findArtifact(artifacts []openapi_client.FactoriesWorkOrderArtifact, id, key string) (openapi_client.FactoriesWorkOrderArtifact, bool) {
	for _, artifact := range artifacts {
		if id != "" && artifact.GetId() == id {
			return artifact, true
		}
		if key != "" && artifact.GetKey() == key {
			return artifact, true
		}
	}
	return openapi_client.FactoriesWorkOrderArtifact{}, false
}

func renderArtifactText(stdout io.Writer, artifact openapi_client.FactoriesWorkOrderArtifact) error {
	data := artifact.GetData()
	lines := []string{
		"ID: " + artifact.GetId(),
		"Type: " + string(artifact.GetType()),
	}
	if key := artifact.GetKey(); key != "" {
		lines = append(lines, "Key: "+key)
	}
	for _, field := range []string{"title", "name", "url"} {
		if value, _ := data[field].(string); value != "" {
			lines = append(lines, fmt.Sprintf("%s%s: %s", strings.ToUpper(field[:1]), field[1:], value))
		}
	}
	if _, err := fmt.Fprintln(stdout, strings.Join(lines, "\n")); err != nil {
		return err
	}
	if body, _ := data["body"].(string); body != "" {
		_, err := fmt.Fprintf(stdout, "\n%s\n", strings.TrimRight(body, "\n"))
		return err
	}
	return nil
}
