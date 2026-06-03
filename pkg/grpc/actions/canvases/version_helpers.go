package canvases

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func parseVersionSHA(versionID string) (string, error) {
	trimmed := strings.TrimSpace(versionID)
	if trimmed == "" {
		return "", status.Error(codes.InvalidArgument, "version id is required")
	}
	if len(trimmed) != 40 {
		return "", status.Errorf(codes.InvalidArgument, "invalid version id: must be a 40-character commit SHA")
	}
	for _, c := range trimmed {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return "", status.Errorf(codes.InvalidArgument, "invalid version id: must be a 40-character commit SHA")
	}
	return trimmed, nil
}
