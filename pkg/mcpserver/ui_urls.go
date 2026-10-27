package mcpserver

import (
    "fmt"
    "os"
    "strings"

    "github.com/google/uuid"
)

// uiBaseURL mirrors the server's base URL resolution so URLs point at the
// SuperPlane UI both in production (BASE_URL) and local development.
func uiBaseURL() string {
    base := strings.TrimRight(strings.TrimSpace(os.Getenv("BASE_URL")), "/")
    if base != "" {
        return base
    }
    port := strings.TrimSpace(os.Getenv("PORT"))
    if port == "" {
        port = "8000"
    }
    return fmt.Sprintf("http://localhost:%s", port)
}

func taskPermalink(orgID uuid.UUID, routeSegment string, number int64) string {
    return fmt.Sprintf("%s/%s/workspaces/%s/work-order/%d", uiBaseURL(), orgID, routeSegment, number)
}

func workspacePermalink(orgID uuid.UUID, routeSegment string) string {
    return fmt.Sprintf("%s/%s/workspaces/%s", uiBaseURL(), orgID, routeSegment)
}
