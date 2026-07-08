package canvases

import (
	"strings"

	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func canvasYAMLFromVersion(canvas *models.Canvas, version *models.CanvasVersion, organizationID string) (string, error) {
	name, description := canvasMetadataFromCanvas(canvas)

	serializedVersion, err := SerializeCanvasVersion(version, organizationID, nil)
	if err != nil {
		return "", err
	}

	return canvasyaml.CanvasResourceYAML(
		serializedVersion,
		canvas.ID.String(),
		name,
		description,
	)
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
