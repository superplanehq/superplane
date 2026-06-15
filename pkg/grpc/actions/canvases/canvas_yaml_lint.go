package canvases

import (
	canvaslint "github.com/superplanehq/superplane/pkg/lint/canvasyaml"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func lintCanvasConfigurationFieldNames(text string) error {
	issues, err := canvaslint.LintConfigurationFieldNames([]byte(text))
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid canvas_yaml: %v", err)
	}
	if len(issues) == 0 {
		return nil
	}

	return status.Error(codes.InvalidArgument, canvaslint.FormatIssues(issues))
}
