package canvas

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/canvas/models"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/cli/layout"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	file            *string
	draft           *bool
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
	autoLayoutNodeIDs := []string{}
	if c.autoLayoutNodes != nil {
		autoLayoutNodeIDs = append(autoLayoutNodeIDs, *c.autoLayoutNodes...)
	}
	draftMode := c.draft != nil && *c.draft

	canvasID, _, err := resolveCanvasForFileUpdate(filePath)
	if err != nil {
		return err
	}

	changeManagementEnabled, err := common.ChangeManagementEnabled(ctx, canvasID)
	if err != nil {
		return err
	}

	if changeManagementEnabled && !draftMode {
		return fmt.Errorf("change management is enabled for this canvas; use --draft to commit to your draft branch, then publish with `superplane apps change-requests create`")
	}

	resource, err := models.ParseCanvasResourceFromFile(filePath, "update")
	if err != nil {
		return err
	}
	if layout.HasFlags(ctx) {
		autoLayout, parseErr := layout.ParseAutoLayout(autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
		if parseErr != nil {
			return parseErr
		}
		if err := applyAutoLayoutToCanvasResource(resource, autoLayout); err != nil {
			return err
		}
	}

	yamlBytes, err := yaml.Marshal(resource)
	if err != nil {
		return fmt.Errorf("marshal canvas yaml: %w", err)
	}

	branch, err := common.EnsureCurrentUserDraftBranch(ctx, canvasID)
	if err != nil {
		return err
	}
	branchName := strings.TrimSpace(branch.GetBranchName())
	expectedHead := strings.TrimSpace(branch.GetTipSha())

	canvasOp := openapi_client.NewCanvasesCanvasRepositoryFileOperation()
	canvasOp.SetPath(materialize.CanvasFileName)
	canvasOp.SetContent(base64.StdEncoding.EncodeToString(yamlBytes))

	commitSHA, err := common.CommitRepositoryFiles(
		ctx,
		canvasID,
		branchName,
		expectedHead,
		"Update canvas.yaml",
		[]openapi_client.CanvasesCanvasRepositoryFileOperation{*canvasOp},
	)
	if err != nil {
		return err
	}

	version, err := common.DescribeAppVersionByID(ctx, canvasID, commitSHA)
	if err != nil {
		return err
	}
	if errText := formatNodeSpecErrorsForCLI(version); errText != "" {
		return fmt.Errorf("%s", errText)
	}

	if !draftMode {
		if err := common.PublishDraftBranch(ctx, canvasID, branchName); err != nil {
			return fmt.Errorf("draft was committed but publish failed: %w", err)
		}
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(version)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		metadata := version.GetMetadata()
		spec := version.GetSpec()

		_, _ = fmt.Fprintf(stdout, "Canvas committed: %s\n", metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "Branch: %s\n", branchName)
		_, _ = fmt.Fprintf(stdout, "App ID: %s\n", metadata.GetCanvasId())
		_, _ = fmt.Fprintf(stdout, "Nodes: %d\n", len(spec.GetNodes()))
		_, _ = fmt.Fprintf(stdout, "Edges: %d\n", len(spec.GetEdges()))

		integrations := make(map[string]struct{})
		for _, node := range spec.GetNodes() {
			if ref, ok := node.GetIntegrationOk(); ok && ref != nil {
				if id := ref.GetId(); id != "" {
					integrations[id] = struct{}{}
				}
			}
		}
		_, err := fmt.Fprintf(stdout, "Integrations: %d\n", len(integrations))
		if err != nil {
			return err
		}
		if warnText := formatNodeSpecWarningsForCLI(version); warnText != "" {
			_, err = fmt.Fprint(stdout, warnText)
		}
		return err
	})
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
