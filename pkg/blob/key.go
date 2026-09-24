package blob

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	ScopeApp          = "app"
	ScopeOrganization = "organization"
	ScopeWorkspace    = "workspace"
	ScopeTask         = "task"
)

func ObjectKey(installationID, scope string, organizationID, factoryID, workOrderID, fileID uuid.UUID) (string, error) {
	if installationID == "" {
		return "", fmt.Errorf("installation id is required")
	}
	if fileID == uuid.Nil {
		return "", fmt.Errorf("file id is required")
	}

	switch scope {
	case ScopeApp:
		return fmt.Sprintf("%s/app/%s", installationID, fileID), nil
	case ScopeOrganization:
		if organizationID == uuid.Nil {
			return "", fmt.Errorf("organization id is required")
		}
		return fmt.Sprintf("%s/orgs/%s/%s", installationID, organizationID, fileID), nil
	case ScopeWorkspace:
		if organizationID == uuid.Nil || factoryID == uuid.Nil {
			return "", fmt.Errorf("organization id and factory id are required")
		}
		return fmt.Sprintf("%s/orgs/%s/workspaces/%s/%s", installationID, organizationID, factoryID, fileID), nil
	case ScopeTask:
		if organizationID == uuid.Nil || factoryID == uuid.Nil || workOrderID == uuid.Nil {
			return "", fmt.Errorf("organization id, factory id, and work order id are required")
		}
		return fmt.Sprintf(
			"%s/orgs/%s/workspaces/%s/tasks/%s/%s",
			installationID,
			organizationID,
			factoryID,
			workOrderID,
			fileID,
		), nil
	default:
		return "", fmt.Errorf("unknown file scope %q", scope)
	}
}
