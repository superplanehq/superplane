package canvas

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/canvas/models"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	file            *string
	message         *string
	autoLayout      *string
	autoLayoutScope *string
	autoLayoutNodes *[]string
}

func resolveCanvasForFileUpdate(filePath string) (string, openapi_client.CanvasesCanvas, error) {
	resource, err := models.ParseCanvasResourceFromFile(filePath, "update")
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	if resource.Metadata == nil {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas metadata is required")
	}

	fileID := ""
	if resource.Metadata.Id != nil {
		fileID = strings.TrimSpace(resource.Metadata.GetId())
	}

	if fileID == "" {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas metadata.id is required in the YAML file")
	}

	canvas := models.CanvasFromCanvas(*resource)
	return fileID, canvas, nil
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}

	autoLayoutValue := ""
	if c.autoLayout != nil {
		autoLayoutValue = strings.TrimSpace(*c.autoLayout)
	}
	autoLayoutScopeValue := ""
	if c.autoLayoutScope != nil {
		autoLayoutScopeValue = strings.TrimSpace(*c.autoLayoutScope)
	}
	_, _ = autoLayoutValue, autoLayoutScopeValue
	autoLayoutNodeIDs := []string{}
	if c.autoLayoutNodes != nil {
		autoLayoutNodeIDs = append(autoLayoutNodeIDs, *c.autoLayoutNodes...)
	}
	_ = autoLayoutNodeIDs

	commitMessage, err := common.RequireCommitMessage(messageValue(c.message))
	if err != nil {
		return fmt.Errorf("%w; use \"superplane apps staging update\" and \"superplane apps staging commit\" to stage changes first", err)
	}

	canvasID, _, err := resolveCanvasForFileUpdate(filePath)
	if err != nil {
		return err
	}

	yamlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read canvas yaml: %w", err)
	}

	if err := common.StageRepositorySpecFile(
		ctx,
		canvasID,
		common.CanvasYAMLRepositoryPath,
		yamlBytes,
	); err != nil {
		return err
	}

	commitResponse, err := common.CommitCanvasStaging(ctx, canvasID, commitMessage)
	if err != nil {
		return fmt.Errorf("canvas was staged but commit failed: %w", err)
	}

	version := commitResponse.GetVersion()
	if version.Metadata == nil {
		return fmt.Errorf("committed version metadata is missing")
	}
	targetVersionID := strings.TrimSpace(version.Metadata.GetId())
	if targetVersionID == "" {
		return fmt.Errorf("updated version metadata is missing")
	}

	canvasYAML, err := common.FetchRepositoryFile(ctx, canvasID, common.CanvasYAMLRepositoryPath, targetVersionID)
	if err != nil {
		return fmt.Errorf("canvas updated but failed to read canvas.yaml: %w", err)
	}

	versionForValidation := versionWithSpecFromYAML(version, string(canvasYAML))
	if errText := formatNodeSpecErrorsForCLI(versionForValidation); errText != "" {
		return fmt.Errorf("%s", errText)
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(version)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		metadata := version.GetMetadata()
		versionForOutput := versionWithSpecFromYAML(version, string(canvasYAML))
		spec, ok := versionForOutput.GetSpecOk()
		nodeCount := 0
		edgeCount := 0
		if ok && spec != nil {
			nodeCount = len(spec.GetNodes())
			edgeCount = len(spec.GetEdges())
		}

		_, _ = fmt.Fprintf(stdout, "Canvas version updated: %s\n", metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "App ID: %s\n", metadata.GetCanvasId())
		_, _ = fmt.Fprintf(stdout, "Nodes: %d\n", nodeCount)
		_, _ = fmt.Fprintf(stdout, "Edges: %d\n", edgeCount)

		integrations := make(map[string]struct{})
		if ok && spec != nil {
			for _, node := range spec.GetNodes() {
				if ref, refOk := node.GetIntegrationOk(); refOk && ref != nil {
					if id := ref.GetId(); id != "" {
						integrations[id] = struct{}{}
					}
				}
			}
		}
		_, err := fmt.Fprintf(stdout, "Integrations: %d\n", len(integrations))
		if err != nil {
			return err
		}
		if warnText := formatNodeSpecWarningsForCLI(versionForOutput); warnText != "" {
			_, err = fmt.Fprint(stdout, warnText)
		}
		return err
	})
}

func messageValue(message *string) string {
	if message == nil {
		return ""
	}
	return *message
}

func versionWithSpecFromYAML(version openapi_client.CanvasesCanvasVersion, canvasYAML string) openapi_client.CanvasesCanvasVersion {
	trimmed := strings.TrimSpace(canvasYAML)
	if trimmed == "" {
		return version
	}

	resource, err := models.ParseCanvas([]byte(trimmed))
	if err != nil || resource.Spec == nil {
		return version
	}

	version.SetSpec(*resource.Spec)
	return version
}

// formatNodeSpecErrorsForCLI summarizes node error_message from the API response (blocks execution until fixed).
func formatNodeSpecErrorsForCLI(version openapi_client.CanvasesCanvasVersion) string {
	spec, ok := version.GetSpecOk()
	if !ok || spec == nil {
		return ""
	}

	var lines []string
	for _, node := range spec.GetNodes() {
		if !node.HasErrorMessage() {
			continue
		}
		msg := strings.TrimSpace(node.GetErrorMessage())
		if msg == "" {
			continue
		}
		id := node.GetId()
		name := strings.TrimSpace(node.GetName())
		if name == "" {
			name = id
		}
		lines = append(lines, fmt.Sprintf("node %s (%s): %s", id, name, msg))
	}
	if len(lines) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("canvas was saved but the following nodes have configuration errors (error_message on each node):\n")
	for _, line := range lines {
		b.WriteString("  - ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func formatNodeSpecWarningsForCLI(version openapi_client.CanvasesCanvasVersion) string {
	spec, ok := version.GetSpecOk()
	if !ok || spec == nil {
		return ""
	}

	var lines []string
	for _, node := range spec.GetNodes() {
		if !node.HasWarningMessage() {
			continue
		}
		msg := strings.TrimSpace(node.GetWarningMessage())
		if msg == "" {
			continue
		}
		id := node.GetId()
		name := strings.TrimSpace(node.GetName())
		if name == "" {
			name = id
		}
		lines = append(lines, fmt.Sprintf("node %s (%s): %s", id, name, msg))
	}
	if len(lines) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\nNode warnings (warning_message):\n")
	for _, line := range lines {
		b.WriteString("  - ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
