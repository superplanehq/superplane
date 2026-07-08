package canvases

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func effectiveCanvasSpec(
	canvas *models.Canvas,
	liveVersion *models.CanvasVersion,
	organizationID string,
	rows []models.WorkflowStagedFile,
) (*pb.Canvas_Spec, error) {
	spec, err := SerializeCanvasSpecFromVersion(liveVersion)
	if err != nil {
		return nil, err
	}

	canvasYAML, err := effectiveSpecYAML(canvas, liveVersion, organizationID, rows, CanvasYAMLRepositoryPath)
	if err != nil {
		return nil, err
	}
	if err := applyCanvasYAMLToSpec(spec, canvasYAML); err != nil {
		return nil, err
	}

	consoleYAML, err := effectiveSpecYAML(canvas, liveVersion, organizationID, rows, ConsoleYAMLRepositoryPath)
	if err != nil {
		return nil, err
	}
	if err := applyConsoleYAMLToSpec(spec, consoleYAML); err != nil {
		return nil, err
	}

	return spec, nil
}

func applyCanvasYAMLToSpec(spec *pb.Canvas_Spec, yamlText string) error {
	if strings.TrimSpace(yamlText) == "" {
		spec.Nodes = nil
		spec.Edges = nil
		return nil
	}

	pbCanvas, err := canvasFromYAMLText(yamlText)
	if err != nil {
		return err
	}
	if pbCanvas == nil || pbCanvas.Spec == nil {
		return nil
	}

	spec.Nodes = pbCanvas.Spec.Nodes
	spec.Edges = pbCanvas.Spec.Edges
	return nil
}

func applyConsoleYAMLToSpec(spec *pb.Canvas_Spec, yamlText string) error {
	if strings.TrimSpace(yamlText) == "" {
		spec.Panels = nil
		spec.Layout = nil
		return nil
	}

	panels, layout, err := consolePanelsLayoutFromYAMLText(yamlText)
	if err != nil {
		return err
	}

	protoPanels, err := ConsolePanelsToProto(panels)
	if err != nil {
		return err
	}

	spec.Panels = protoPanels
	spec.Layout = ConsoleLayoutToProto(layout)
	return nil
}
