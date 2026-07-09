package canvas

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/layout"
	appyaml "github.com/superplanehq/superplane/pkg/yaml"
	"gopkg.in/yaml.v3"
)

type updateCommand struct {
	file            *string
	message         *string
	autoLayout      *string
	autoLayoutScope *string
	autoLayoutNodes *[]string
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("update accepts at most one positional argument")
	}

	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}
	if strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("canvas file is required")
	}

	appArg := ""
	if len(ctx.Args) == 1 {
		appArg = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, appArg)
	if err != nil {
		return err
	}

	commitMessage, err := common.RequireCommitMessage(messageValue(c.message))
	if err != nil {
		return fmt.Errorf("%w; use \"superplane apps staging update\" and \"superplane apps staging commit\" to stage changes first", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	resource, err := appyaml.CanvasFromYAML(content)
	if err != nil {
		return fmt.Errorf("invalid canvas yaml in %s: %w", filePath, err)
	}

	resource, err = c.applyLayout(ctx, resource)
	if err != nil {
		return err
	}

	yamlBytes, err := yaml.Marshal(resource)
	if err != nil {
		return fmt.Errorf("marshal canvas yaml: %w", err)
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

	resource, err = appyaml.CanvasFromYAML(canvasYAML)
	if err != nil {
		return fmt.Errorf("invalid canvas yaml in %s: %w", filePath, err)
	}

	if errText := formatNodeSpecErrorsForCLI(resource.Spec.Nodes); errText != "" {
		return fmt.Errorf("%s", errText)
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(version)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		metadata := version.GetMetadata()
		nodeCount := 0
		edgeCount := 0
		if resource.Spec != nil {
			nodeCount = len(resource.Spec.Nodes)
			edgeCount = len(resource.Spec.Edges)
		}

		_, _ = fmt.Fprintf(stdout, "Canvas version updated: %s\n", metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "App ID: %s\n", metadata.GetCanvasId())
		_, _ = fmt.Fprintf(stdout, "Nodes: %d\n", nodeCount)
		_, _ = fmt.Fprintf(stdout, "Edges: %d\n", edgeCount)

		integrations := make(map[string]struct{})
		if resource.Spec != nil {
			for _, node := range resource.Spec.Nodes {
				if node.Integration != nil {
					if id := node.Integration.ID; id != "" {
						integrations[id] = struct{}{}
					}
				}
			}
		}
		_, err := fmt.Fprintf(stdout, "Integrations: %d\n", len(integrations))
		if err != nil {
			return err
		}
		if warnText := formatNodeSpecWarningsForCLI(resource.Spec.Nodes); warnText != "" {
			_, err = fmt.Fprint(stdout, warnText)
		}
		return err
	})
}

func (c *updateCommand) applyLayout(ctx core.CommandContext, resource *appyaml.Canvas) (*appyaml.Canvas, error) {
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

	autoLayout, err := ResolveUpdateAutoLayout(ctx, autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
	if err != nil {
		return nil, err
	}

	if autoLayout == nil {
		return resource, nil
	}

	if resource.Spec == nil {
		return resource, nil
	}

	nodes := []layout.N{}
	for _, node := range resource.Spec.Nodes {
		if node.Type == appyaml.NodeTypeWidget {
			continue
		}
		nodes = append(nodes, layout.N{
			ID:       node.ID,
			Type:     node.Type,
			Position: layout.Position{X: node.Position.X, Y: node.Position.Y},
		})
	}

	edges := []layout.E{}
	for _, edge := range resource.Spec.Edges {
		edges = append(edges, layout.E{
			SourceID: edge.SourceID,
			TargetID: edge.TargetID,
			Channel:  edge.Channel,
		})
	}

	positionedNodes, _, err := layout.ApplyLayout(nodes, edges, autoLayout)
	if err != nil {
		return nil, fmt.Errorf("error applying auto-layout: %w", err)
	}

	for _, positionedNode := range positionedNodes {
		i := slices.IndexFunc(resource.Spec.Nodes, func(node appyaml.Node) bool {
			return node.ID == positionedNode.ID
		})

		if i == -1 {
			continue
		}

		resource.Spec.Nodes[i].Position.X = positionedNode.Position.X
		resource.Spec.Nodes[i].Position.Y = positionedNode.Position.Y
	}

	return resource, nil
}

func messageValue(message *string) string {
	if message == nil {
		return ""
	}
	return *message
}

// formatNodeSpecErrorsForCLI summarizes node error_message from the API response (blocks execution until fixed).
func formatNodeSpecErrorsForCLI(nodes []appyaml.Node) string {
	var lines []string
	for _, node := range nodes {
		if node.ErrorMessage == nil || *node.ErrorMessage == "" {
			continue
		}

		msg := strings.TrimSpace(*node.ErrorMessage)
		if msg == "" {
			continue
		}

		name := strings.TrimSpace(node.Name)
		if name == "" {
			name = node.ID
		}
		lines = append(lines, fmt.Sprintf("node %s (%s): %s", node.ID, name, msg))
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

func formatNodeSpecWarningsForCLI(nodes []appyaml.Node) string {
	var lines []string
	for _, node := range nodes {
		if node.WarningMessage == nil || *node.WarningMessage == "" {
			continue
		}

		msg := strings.TrimSpace(*node.WarningMessage)
		if msg == "" {
			continue
		}

		name := strings.TrimSpace(node.Name)
		if name == "" {
			name = node.ID
		}

		lines = append(lines, fmt.Sprintf("node %s (%s): %s", node.ID, name, msg))
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

// ResolveUpdateAutoLayout picks the auto-layout settings for canvas update.
// Flags take precedence; otherwise a file-level autoLayout field is used.
// When neither is set, horizontal full-canvas layout is applied.
func ResolveUpdateAutoLayout(ctx core.CommandContext, value string, scopeValue string, nodeIDs []string) (*layout.AutoLayout, error) {
	if HasFlags(ctx) {
		return layout.ParseAutoLayout(value, scopeValue, nodeIDs)
	}
	defaultLayout := layout.DefaultAutoLayout()
	return &defaultLayout, nil
}

func HasFlags(ctx core.CommandContext) bool {
	if ctx.Cmd == nil {
		return false
	}

	flags := ctx.Cmd.Flags()
	if flags == nil {
		return false
	}

	return flags.Changed("auto-layout") || flags.Changed("auto-layout-scope") || flags.Changed("auto-layout-node")
}
