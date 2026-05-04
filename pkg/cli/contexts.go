package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ContextsCommand struct{}

func (c *ContextsCommand) Execute(ctx core.CommandContext) error {
	contexts := GetContexts()
	if len(contexts) == 0 {
		return fmt.Errorf("no contexts configured; run superplane connect [BASE_URL] [API_TOKEN]")
	}

	if len(ctx.Args) == 2 {
		selected, err := SwitchContext(ctx.Args[0], ctx.Args[1])
		if err != nil {
			return err
		}
		return c.renderContext(ctx, *selected)
	}

	if !ctx.Renderer.IsText() || !ctx.IsInteractive() {
		return c.renderContexts(ctx, contexts)
	}

	selected, err := c.selectContextInteractively(ctx, contexts)
	if err != nil {
		return err
	}

	org := selected.OrganizationID
	if org == "" {
		org = selected.Organization
	}
	selected, err = SwitchContext(selected.URL, org)
	if err != nil {
		return err
	}

	return c.renderContext(ctx, *selected)
}

/*
 * Do not render the API token in the output for security reasons.
 */
func (c *ContextsCommand) renderContexts(ctx core.CommandContext, contexts []ConfigContext) error {
	if !ctx.Renderer.IsText() {
		ctxs := make([]map[string]any, 0, len(contexts))
		for _, context := range contexts {
			ctxs = append(ctxs, contextToMap(context))
		}

		return ctx.Renderer.Render(map[string]any{
			"contexts": ctxs,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		for i, context := range contexts {
			_, _ = fmt.Fprintf(stdout, "%d. %s (%s)\n", i+1, context.Organization, context.URL)
		}
		return nil
	})
}

func (c *ContextsCommand) renderContext(ctx core.CommandContext, context ConfigContext) error {
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(contextToMap(context))
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Current context: %q (%s)\n", context.Organization, context.URL)
		return nil
	})
}

func contextToMap(context ConfigContext) map[string]any {
	m := map[string]any{
		"organization": context.Organization,
		"url":          context.URL,
	}
	if context.OrganizationID != "" {
		m["organizationId"] = context.OrganizationID
	}
	return m
}

func (c *ContextsCommand) selectContextInteractively(ctx core.CommandContext, contexts []ConfigContext) (*ConfigContext, error) {
	currentContext, hasCurrentContext := GetCurrentContext()
	currentSelector := ContextSelector(currentContext)

	//
	// Render prompt for context selection
	//
	err := ctx.Renderer.RenderText(func(stdout io.Writer) error {
		for i, context := range contexts {
			currentPrefix := " "
			if hasCurrentContext && ContextSelector(context) == currentSelector {
				currentPrefix = "*"
			}
			_, _ = fmt.Fprintf(stdout, "%s %d. %s (%s)\n", currentPrefix, i+1, context.Organization, context.URL)
		}
		_, _ = fmt.Fprint(stdout, "Select a context number: ")
		return nil
	})

	if err != nil {
		return nil, err
	}

	//
	// Validate selection
	//
	reader := bufio.NewReader(ctx.Cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read selected context: %w", err)
	}

	selectedIndex, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return nil, fmt.Errorf("invalid context selection %q", strings.TrimSpace(input))
	}

	if selectedIndex < 1 || selectedIndex > len(contexts) {
		return nil, fmt.Errorf("context selection must be between 1 and %d", len(contexts))
	}

	context := contexts[selectedIndex-1]
	return &context, nil
}

var contextsCmd = &cobra.Command{
	Use:   "contexts [BASE_URL] [ORGANIZATION]",
	Short: "List and switch CLI contexts",
	Long: "Without arguments, lists available contexts and prompts for a selection. " +
		"With BASE_URL and ORGANIZATION, switches directly. " +
		"ORGANIZATION can be the organization name or ID.",
	Args: cobra.MatchAll(
		func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || len(args) == 2 {
				return nil
			}
			return fmt.Errorf("accepts 0 or 2 args, received %d", len(args))
		},
	),
}

func init() {
	core.Bind(contextsCmd, &ContextsCommand{}, defaultBindOptions())
	RootCmd.AddCommand(contextsCmd)
}
