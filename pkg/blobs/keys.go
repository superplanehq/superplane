package blobs

import (
	"fmt"
	"path"
	"strings"
)

func objectKey(scope Scope, blobPath string) (string, error) {
	if err := validateScope(scope); err != nil {
		return "", err
	}

	cleaned, err := cleanPath(blobPath)
	if err != nil {
		return "", err
	}

	switch scope.Type {
	case ScopeOrganization:
		return fmt.Sprintf("blobs/organization/%s/%s", scope.OrganizationID, cleaned), nil
	case ScopeCanvas:
		return fmt.Sprintf("blobs/canvas/%s/%s", scope.CanvasID, cleaned), nil
	case ScopeNode:
		return fmt.Sprintf("blobs/node/%s/%s/%s", scope.CanvasID, scope.NodeID, cleaned), nil
	case ScopeExecution:
		return fmt.Sprintf("blobs/execution/%s/%s", scope.ExecutionID, cleaned), nil
	default:
		return "", ErrInvalidScope
	}
}

func scopePrefix(scope Scope) (string, error) {
	if err := validateScope(scope); err != nil {
		return "", err
	}

	switch scope.Type {
	case ScopeOrganization:
		return fmt.Sprintf("blobs/organization/%s/", scope.OrganizationID), nil
	case ScopeCanvas:
		return fmt.Sprintf("blobs/canvas/%s/", scope.CanvasID), nil
	case ScopeNode:
		return fmt.Sprintf("blobs/node/%s/%s/", scope.CanvasID, scope.NodeID), nil
	case ScopeExecution:
		return fmt.Sprintf("blobs/execution/%s/", scope.ExecutionID), nil
	default:
		return "", ErrInvalidScope
	}
}

func cleanPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "/")
	p = path.Clean(p)
	if p == "." {
		return "", nil
	}
	if p == ".." || strings.HasPrefix(p, "../") || strings.Contains(p, "/../") || strings.HasSuffix(p, "/..") {
		return "", ErrInvalidBlobPath
	}
	return p, nil
}

func validateScope(scope Scope) error {
	switch scope.Type {
	case ScopeOrganization:
		if strings.TrimSpace(scope.OrganizationID) == "" {
			return fmt.Errorf("%w: organization scope requires organization_id", ErrInvalidScope)
		}
	case ScopeCanvas:
		if strings.TrimSpace(scope.CanvasID) == "" {
			return fmt.Errorf("%w: canvas scope requires canvas_id", ErrInvalidScope)
		}
	case ScopeNode:
		if strings.TrimSpace(scope.CanvasID) == "" || strings.TrimSpace(scope.NodeID) == "" {
			return fmt.Errorf("%w: node scope requires canvas_id and node_id", ErrInvalidScope)
		}
	case ScopeExecution:
		if strings.TrimSpace(scope.ExecutionID) == "" {
			return fmt.Errorf("%w: execution scope requires execution_id", ErrInvalidScope)
		}
	default:
		return fmt.Errorf("%w: unknown scope type %q", ErrInvalidScope, scope.Type)
	}

	return nil
}
