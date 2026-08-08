package core

import (
	"context"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type Command interface {
	Execute(ctx CommandContext) error
}

type CommandContext struct {
	Context  context.Context
	Cmd      *cobra.Command
	Args     []string
	Logger   *log.Entry
	API      *openapi_client.APIClient
	Renderer Renderer
	Config   ConfigContext
}

/*
 * Interface that allows commands to access
 * and update the current configuration context.
 */
type ConfigContext interface {
	GetActiveApp() string
	SetActiveApp(appID string) error
	GetActiveFactory() string
	SetActiveFactory(factoryID string) error
	GetURL() string
}

// IsInteractive returns true when stdin is a terminal,
// meaning the user can respond to interactive prompts.
func (c CommandContext) IsInteractive() bool {
	if f, ok := c.Cmd.InOrStdin().(*os.File); ok {
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return false
}

type BindOptions struct {
	NewAPIClient        func() *openapi_client.APIClient
	NewConfigContext    func() ConfigContext
	DefaultOutputFormat func() string
	// CLIVersion and ServerVersion feed the version-skew hint that is
	// appended to "not found" API errors, which on self-hosted servers
	// usually mean the CLI is newer than the server. Both are optional;
	// the hint is skipped when either is unset.
	CLIVersion    string
	ServerVersion func(ctx context.Context) (string, error)
}

func NewCommandContext(cmd *cobra.Command, args []string, options BindOptions) (CommandContext, error) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	outputFormat := "text"
	if options.DefaultOutputFormat != nil {
		outputFormat = options.DefaultOutputFormat()
	}

	flagValue, err := cmd.Flags().GetString("output")
	if err == nil && flagValue != "" {
		outputFormat = flagValue
	}

	renderer, err := NewRenderer(outputFormat, cmd.OutOrStdout())
	if err != nil {
		return CommandContext{}, err
	}

	commandContext := CommandContext{
		Context:  ctx,
		Cmd:      cmd,
		Args:     args,
		Logger:   log.WithField("command", cmd.CommandPath()),
		Renderer: renderer,
	}

	if options.NewAPIClient != nil {
		commandContext.API = options.NewAPIClient()
	}
	if options.NewConfigContext != nil {
		commandContext.Config = options.NewConfigContext()
	}

	return commandContext, nil
}

func Bind(cmd *cobra.Command, command Command, options BindOptions) {
	cmd.RunE = func(cobraCmd *cobra.Command, args []string) error {
		ctx, err := NewCommandContext(cobraCmd, args, options)
		if err != nil {
			return err
		}

		execErr := command.Execute(ctx)
		formatted := FormatCommandError(execErr)
		if formatted == nil {
			return nil
		}

		// The skew check runs on the raw error because formatting
		// flattens the API error into a plain string.
		if hint := VersionSkewHint(ctx.Context, execErr, options); hint != "" {
			return fmt.Errorf("%s\n%s", formatted.Error(), hint)
		}

		return formatted
	}
}
