package canvases

import (
	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"strings"
)

func canvasYAMLFromVersion(canvas *models.Canvas, version *models.CanvasVersion, organizationID string) (string, error) {
	return canvasyaml.CanvasResourceYAML(SerializeCanvasVersion(canvas, version, organizationID, nil), canvas.ID.String())
}

func consoleYAMLFromVersion(canvas *models.Canvas, version *models.CanvasVersion) (string, error) {
	canvasName, _ := canvasMetadataFromCanvas(canvas)
	raw, err := models.CanvasVersionToConsoleYML(canvasName, version)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

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

func consolePanelsLayoutFromYAMLText(text string) ([]models.ConsolePanel, []models.ConsoleLayoutItem, error) {
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
