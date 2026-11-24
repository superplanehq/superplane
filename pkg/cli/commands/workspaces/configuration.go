package workspaces

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	configurationManifestPath = ".superplane/factory.yaml"
	configurationCoverage     = "All stages and automations"
)

type configurationTarget struct {
	workspace string
}

func (t configurationTarget) validate() error {
	if strings.TrimSpace(t.workspace) == "" {
		return fmt.Errorf("pass --workspace with a workspace key or UUID")
	}
	return nil
}

func (t *configurationTarget) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&t.workspace, "workspace", "", "workspace key or UUID (required)")
	_ = cmd.MarkFlagRequired("workspace")
}

type configurationInput struct {
	configurationTarget
	file string
}

func (c *configurationInput) bind(cmd *cobra.Command) {
	c.configurationTarget.bind(cmd)
	cmd.Flags().StringVarP(&c.file, "file", "f", "", "Factory manifest file (required; - for stdin)")
	_ = cmd.MarkFlagRequired("file")
}

func (c configurationInput) readBundle(ctx core.CommandContext) (openapi_client.FactoriesFactoryConfigurationBundle, error) {
	bundle := openapi_client.NewFactoriesFactoryConfigurationBundle()
	if err := c.validate(); err != nil {
		return *bundle, err
	}
	if strings.TrimSpace(c.file) == "" {
		return *bundle, fmt.Errorf("pass -f with a Factory manifest file")
	}
	var content []byte
	var err error
	if c.file == "-" {
		content, err = io.ReadAll(ctx.Cmd.InOrStdin())
	} else {
		// #nosec G304 -- the CLI user explicitly selects the manifest file.
		content, err = os.ReadFile(c.file)
	}
	if err != nil {
		return *bundle, fmt.Errorf("read Factory manifest: %w", err)
	}
	references, err := configurationManifestReferences(string(content))
	if err != nil {
		return *bundle, err
	}
	file := openapi_client.NewFactoriesFactoryConfigurationFile()
	file.SetPath(configurationManifestPath)
	file.SetContent(string(content))
	bundle.SetManifestPath(configurationManifestPath)
	files := []openapi_client.FactoriesFactoryConfigurationFile{*file}
	if len(references) == 0 {
		bundle.SetFiles(files)
		return *bundle, nil
	}
	if c.file == "-" {
		return *bundle, fmt.Errorf("custom automation references require -f with a manifest file path")
	}
	directory, err := os.OpenRoot(filepath.Dir(c.file))
	if err != nil {
		return *bundle, fmt.Errorf("open manifest directory: %w", err)
	}
	defer directory.Close()
	for _, reference := range references {
		content, err := readConfigurationReference(directory, reference)
		if err != nil {
			return *bundle, fmt.Errorf("read custom automation %q: %w", reference, err)
		}
		file := openapi_client.NewFactoriesFactoryConfigurationFile()
		file.SetPath(".superplane/" + reference)
		file.SetContent(content)
		files = append(files, *file)
	}
	bundle.SetFiles(files)
	return *bundle, nil
}

type configurationPlanCommand struct {
	configurationInput
}

func newConfigurationPlanCommand(options core.BindOptions) *cobra.Command {
	command := &configurationPlanCommand{}
	cmd := &cobra.Command{
		Use: "plan", Short: "Plan stage and automation changes", Args: cobra.NoArgs,
	}
	command.bind(cmd)
	core.Bind(cmd, command, options)
	return cmd
}

func (c *configurationPlanCommand) Execute(ctx core.CommandContext) error {
	bundle, err := c.readBundle(ctx)
	if err != nil {
		return err
	}
	_, err = previewFactoryConfiguration(ctx, c.configurationInput, bundle)
	return err
}

func previewFactoryConfiguration(ctx core.CommandContext, input configurationInput, bundle openapi_client.FactoriesFactoryConfigurationBundle) (*openapi_client.FactoriesPlanFactoryConfigurationResponse, error) {
	body := openapi_client.NewFactoriesPlanFactoryConfigurationBody()
	body.SetBundle(bundle)
	plan, _, err := ctx.API.FactoryAPI.
		FactoriesPlanFactoryConfiguration(ctx.Context, strings.TrimSpace(input.workspace)).Body(*body).Execute()
	if err != nil {
		return nil, err
	}
	if plan == nil || plan.GetCoverage() != configurationCoverage {
		return nil, fmt.Errorf("invalid plan response: expected %s", configurationCoverage)
	}
	if err := renderConfigurationPlan(ctx, plan); err != nil {
		return nil, err
	}
	if len(plan.GetDiagnostics()) != 0 {
		return nil, fmt.Errorf("Factory manifest has diagnostics; correct the listed fields and plan again")
	}
	if len(plan.GetChanges()) != 0 && strings.TrimSpace(plan.GetBaseStateToken()) == "" {
		return nil, fmt.Errorf("invalid plan response: missing base-state token")
	}
	return plan, nil
}

func renderConfigurationPlan(ctx core.CommandContext, plan *openapi_client.FactoriesPlanFactoryConfigurationResponse) error {
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(plan)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if _, err := fmt.Fprintf(stdout, "Coverage: %s (partial workspace configuration).\n", plan.GetCoverage()); err != nil {
			return err
		}
		for _, diagnostic := range plan.GetDiagnostics() {
			if _, err := fmt.Fprintf(stdout, "%s: %s: %s\n", diagnostic.GetPath(), diagnostic.GetField(), diagnostic.GetMessage()); err != nil {
				return err
			}
		}
		for _, change := range plan.GetChanges() {
			if _, err := fmt.Fprintf(stdout, "%s %s\n  Previous: %q\n  Requested: %q\n", change.GetResource(), change.GetFieldPath(), change.GetPreviousValue(), change.GetRequestedValue()); err != nil {
				return err
			}
		}
		if len(plan.GetChanges()) == 0 && len(plan.GetDiagnostics()) == 0 {
			_, err := fmt.Fprintln(stdout, "No changes.")
			return err
		}
		return nil
	})
}

type configurationApplyCommand struct {
	configurationInput
	yes bool
}

func newConfigurationApplyCommand(options core.BindOptions) *cobra.Command {
	command := &configurationApplyCommand{}
	cmd := &cobra.Command{
		Use: "apply", Short: "Apply stage and automation changes", Args: cobra.NoArgs,
	}
	command.bind(cmd)
	cmd.Flags().BoolVar(&command.yes, "yes", false, "apply the displayed plan without confirmation")
	core.Bind(cmd, command, options)
	return cmd
}

func (c *configurationApplyCommand) Execute(ctx core.CommandContext) error {
	bundle, err := c.readBundle(ctx)
	if err != nil {
		return err
	}
	previewCtx := ctx
	if !ctx.Renderer.IsText() {
		previewCtx.Renderer, err = core.NewRenderer(string(ctx.Renderer.Format()), ctx.Cmd.ErrOrStderr())
		if err != nil {
			return err
		}
	}
	plan, err := previewFactoryConfiguration(previewCtx, c.configurationInput, bundle)
	if err != nil {
		return err
	}
	if len(plan.GetChanges()) == 0 {
		if !ctx.Renderer.IsText() {
			return ctx.Renderer.Render(plan)
		}
		return nil
	}
	if !c.yes {
		if err := confirmFactoryConfiguration(ctx); err != nil {
			return err
		}
	}
	body := openapi_client.NewFactoriesApplyFactoryConfigurationBody()
	body.SetBundle(bundle)
	body.SetBaseStateToken(plan.GetBaseStateToken())
	response, _, err := ctx.API.FactoryAPI.
		FactoriesApplyFactoryConfiguration(ctx.Context, strings.TrimSpace(c.workspace)).Body(*body).Execute()
	if err != nil {
		return err
	}
	if response == nil || response.GetCoverage() != configurationCoverage {
		return fmt.Errorf("invalid apply response: expected %s", configurationCoverage)
	}
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if response.GetChanged() {
			_, err := fmt.Fprintln(stdout, "Applied stage and automation configuration.")
			return err
		}
		_, err := fmt.Fprintln(stdout, "No changes.")
		return err
	})
}

func confirmFactoryConfiguration(ctx core.CommandContext) error {
	if !ctx.IsInteractive() || !ctx.Renderer.IsText() {
		return fmt.Errorf("confirmation requires an interactive terminal with text output; pass --yes for noninteractive use")
	}
	if _, err := fmt.Fprint(ctx.Cmd.OutOrStdout(), "Apply these changes? [y/N]: "); err != nil {
		return err
	}
	answer, err := bufio.NewReader(ctx.Cmd.InOrStdin()).ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read confirmation; pass --yes for noninteractive use: %w", err)
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("apply canceled")
	}
	return nil
}
