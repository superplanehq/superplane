package canvases

import (
	"strings"

	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func canvasFromYAMLText(text string) (*pb.Canvas, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, grpcerrors.InvalidArgument(nil, "canvas_yaml is empty")
	}

	canvas, err := canvasyaml.ParseCanvasResource([]byte(trimmed))
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas_yaml")
	}

	return canvas, nil
}

func ParseConsoleYAML(text string) ([]models.ConsolePanel, []models.ConsoleLayoutItem, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, nil, grpcerrors.InvalidArgument(nil, "console_yaml is empty")
	}

	doc, err := models.ConsoleFromYML([]byte(trimmed))
	if err != nil {
		return nil, nil, grpcerrors.InvalidArgument(err, "invalid console_yaml")
	}

	if err := doc.Validate(); err != nil {
		return nil, nil, grpcerrors.InvalidArgument(err, "invalid console_yaml")
	}

	return doc.Spec.Panels, doc.Spec.Layout, nil
}
