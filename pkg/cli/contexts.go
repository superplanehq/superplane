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

	if len(ctx.Args) == 1 {
		selected, err := SaveCurrentContextBySelector(ctx.Args[0])
		if err != nil {
			return err
		}

		return c.renderContext(ctx, *selected)
	}

	//
	// If not in text mode, render the contexts as a list.
	//
	if !ctx.Renderer.IsText() {
		return c.renderContexts(ctx, contexts)
	}

	//
	// If in text mode, select the context interactively.
	//
	selected, err := c.selectContextInteractively(ctx, contexts)
	if err != nil {
		return err
	}

	selected, err = SaveCurrentContextBySelector(ContextSelector(*selected))
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
			ctxs = append(ctxs, map[string]any{
				"organization": context.Organization,
				"url":          context.URL,
			})
		}

		return ctx.Renderer.Render(map[string]any{
			"contexts": ctxs,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		for i, context := range contexts {
			_, _ = fmt.Fprintf(stdout, "%d. %s (%s)\n", i+1, context.Organization, ContextSelector(context))
		}
		return nil
	})
}

/*
 * Do not render the API token in the output for security reasons.
 */
func (c *ContextsCommand) renderContext(ctx core.CommandContext, context ConfigContext) error {
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"organization": context.Organization,
			"url":          context.URL,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Current context: %q (%s)\n", context.Organization, context.URL)
		return nil
	})
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
			_, _ = fmt.Fprintf(stdout, "%s %d. %s (%s)\n", currentPrefix, i+1, context.Organization, ContextSelector(context))
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
	Use:   "contexts [BASE_URL/ORGANIZATION]",
	Short: "List and switch CLI contexts",
	Long:  "Without arguments, shows available contexts and prompts for a selection. With a BASE_URL/ORGANIZATION selector, switches directly.",
	Args:  cobra.MaximumNArgs(1),
}

func init() {
	core.Bind(contextsCmd, &ContextsCommand{}, defaultBindOptions())
	RootCmd.AddCommand(contextsCmd)
}
