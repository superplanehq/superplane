package core

import (
	"context"

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
	GetActiveCanvas() string
	SetActiveCanvas(canvasID string) error
}

type BindOptions struct {
	NewAPIClient        func() *openapi_client.APIClient
	NewConfigContext    func() ConfigContext
	DefaultOutputFormat func() string
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

		return command.Execute(ctx)
	}
}
