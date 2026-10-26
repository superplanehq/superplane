package mcpserver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
)

// workspaceURL returns the web app URL of one workspace.
func workspaceURL(baseURL string, organizationID uuid.UUID, workspaceSlug string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return fmt.Sprintf("%s/%s/workspaces/%s", base, organizationID, workspaceSlug)
}

// taskLinks builds the web app URLs of the tasks in one workspace.
type taskLinks struct {
	prefix string
}

func newTaskLinks(baseURL string, factory *models.Factory) taskLinks {
	return taskLinks{prefix: workspaceURL(baseURL, factory.OrganizationID, factory.RouteSegment()) + "/task/"}
}

func (links taskLinks) url(number int64) string {
	return links.prefix + strconv.FormatInt(number, 10)
}
