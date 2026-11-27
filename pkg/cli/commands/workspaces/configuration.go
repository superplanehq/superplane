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
	"github.com/superplanehq/superplane/pkg/workspaceconfig"
)

const (
	configurationManifestPath = workspaceconfig.ManifestPath
	configurationCoverage     = workspaceconfig.Coverage
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
	root string
}

func (c *configurationInput) bind(cmd *cobra.Command) {
	c.configurationTarget.bind(cmd)
	cmd.Flags().StringVarP(&c.file, "file", "f", configurationManifestPath, "Factory manifest file (- for stdin)")
	cmd.Flags().StringVar(&c.root, "root", "", "bundle root for the manifest and automation files")
}

func (c configurationInput) readBundle(ctx core.CommandContext) (openapi_client.FactoriesFactoryConfigurationBundle, error) {
	bundle := openapi_client.NewFactoriesFactoryConfigurationBundle()
	if err := c.validate(); err != nil {
		return *bundle, err
	}
	if strings.TrimSpace(c.file) == "" {
		return *bundle, fmt.Errorf("pass -f with a Factory manifest file")
	}
	manifestPath := configurationManifestPath
	var content string
	var directory *os.Root
	var err error
	if c.file == "-" {
		var input []byte
		input, err = io.ReadAll(ctx.Cmd.InOrStdin())
		content = string(input)
	} else {
		manifestFile, absErr := filepath.Abs(c.file)
		if absErr != nil {
			return *bundle, absErr
		}
		rootPath := filepath.Dir(manifestFile)
		manifestPath = filepath.Base(manifestFile)
		if c.root != "" {
			rootPath, err = filepath.Abs(c.root)
			if err != nil {
				return *bundle, err
			}
			relative, relErr := filepath.Rel(rootPath, manifestFile)
			if relErr != nil {
				return *bundle, fmt.Errorf("resolve manifest inside --root: %w", relErr)
			}
			manifestPath, err = workspaceconfig.NormalizePath(filepath.ToSlash(relative))
			if err != nil {
				return *bundle, fmt.Errorf("manifest must be inside --root: %w", err)
			}
		}
		manifestPath, err = workspaceconfig.NormalizePath(manifestPath)
		if err != nil {
			return *bundle, fmt.Errorf("invalid manifest path: %w", err)
		}
		directory, err = openConfigurationInputDirectory(rootPath)
		if err != nil {
			return *bundle, fmt.Errorf("open manifest root for %q: %w", c.file, err)
		}
		defer directory.Close()
		content, err = readConfigurationReference(directory, manifestPath)
	}
	if err != nil {
		return *bundle, fmt.Errorf("read Factory manifest %q: %w", c.file, err)
	}
	references, err := configurationManifestReferences(content, manifestPath)
	if err != nil {
		return *bundle, fmt.Errorf("%s: %w", manifestPath, err)
	}
	if c.file == "-" && len(references) != 0 {
		return *bundle, fmt.Errorf("custom automation references require -f with a manifest file path")
	}
	file := openapi_client.NewFactoriesFactoryConfigurationFile()
	file.SetPath(manifestPath)
	file.SetContent(content)
	bundle.SetManifestPath(manifestPath)
	files := []openapi_client.FactoriesFactoryConfigurationFile{*file}
	for _, reference := range references {
		content, err := readConfigurationReference(directory, reference)
		if err != nil {
			return *bundle, fmt.Errorf("read custom automation %q: %w", reference, err)
		}
		file := openapi_client.NewFactoriesFactoryConfigurationFile()
		file.SetPath(reference)
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
		Use: "plan", Short: "Plan workspace name, stage, and automation changes",
		Long: "Plan workspace name, stage, and automation changes. Repository settings must match the workspace. Change repositories in workspace settings and export again.", Args: cobra.NoArgs,
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
		Use: "apply", Short: "Apply workspace name, stage, and automation changes",
		Long: "Apply workspace name, stage, and automation changes. Repository settings must match the workspace. Change repositories in workspace settings and export again.", Args: cobra.NoArgs,
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
			_, err := fmt.Fprintln(stdout, "Applied workspace configuration.")
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
