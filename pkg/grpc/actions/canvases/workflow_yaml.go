package canvases

import (
	"strings"

	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func canvasYAMLFromVersion(canvas *models.Canvas, version *models.CanvasVersion, organizationID string) (string, error) {
	return canvasyaml.CanvasResourceYAML(SerializeCanvasVersion(version, organizationID), canvas.ID.String(), canvas.IsTemplate)
}

func consoleYAMLFromVersion(version *models.CanvasVersion) (string, error) {
	raw, err := models.CanvasVersionToConsoleYML(version)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func canvasFromYAMLText(text string) (*pb.Canvas, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, status.Error(codes.InvalidArgument, "canvas_yaml is empty")
	}

	canvas, err := canvasyaml.ParseCanvasResource([]byte(trimmed))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas_yaml: %v", err)
	}

	return canvas, nil
}

func consolePanelsLayoutFromYAMLText(text string) ([]models.ConsolePanel, []models.ConsoleLayoutItem, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "console_yaml is empty")
	}

	doc, err := models.ConsoleFromYML([]byte(trimmed))
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid console_yaml: %v", err)
	}

	if err := doc.Validate(); err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid console_yaml: %v", err)
	}

	return doc.Spec.Panels, doc.Spec.Layout, nil
}
