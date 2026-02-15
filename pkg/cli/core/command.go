package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type OutputFormat string

const (
	OutputFormatText OutputFormat = "text"
	OutputFormatJSON OutputFormat = "json"
	OutputFormatYAML OutputFormat = "yaml"
)

type Renderer struct {
	format OutputFormat
	stdout io.Writer
}

func NewRenderer(rawFormat string, stdout io.Writer) (Renderer, error) {
	format := OutputFormat(rawFormat)
	if format == "" {
		format = OutputFormatText
	}

	switch format {
	case OutputFormatText, OutputFormatJSON, OutputFormatYAML:
		return Renderer{format: format, stdout: stdout}, nil
	default:
		return Renderer{}, fmt.Errorf("invalid output format %q, expected one of: text, json, yaml", rawFormat)
	}
}

func (r Renderer) Format() OutputFormat {
	return r.format
}

func (r Renderer) IsText() bool {
	return r.format == OutputFormatText
}

func (r Renderer) Render(value any) error {
	switch r.format {
	case OutputFormatJSON:
		payload, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(r.stdout, string(payload))
		return err
	case OutputFormatYAML:
		payload, err := yaml.Marshal(value)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(r.stdout, string(payload))
		return err
	case OutputFormatText:
		return fmt.Errorf("text output requires RenderText")
	default:
		return fmt.Errorf("unsupported output format %q", r.format)
	}
}

func (r Renderer) RenderText(render func(io.Writer) error) error {
	if r.format != OutputFormatText {
		return fmt.Errorf("RenderText can only be used with text output")
	}

	return render(r.stdout)
}

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
}

type BindOptions struct {
	NewAPIClient        func() *openapi_client.APIClient
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
