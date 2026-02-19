package core

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ghodss/yaml"
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
