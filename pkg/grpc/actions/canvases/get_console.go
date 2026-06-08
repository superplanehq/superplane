package canvases

import "context"

// GetConsole materializes console.yaml for a canvas version.
func GetConsole(ctx context.Context, organizationID, canvasID string, versionID string) (string, error) {
	return ReadRepositorySpecFile(ctx, organizationID, canvasID, versionID, ConsoleYAMLRepositoryPath)
}
