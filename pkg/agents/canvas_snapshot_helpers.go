package agents

import (
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
)

type superPlaneCanvasNodeSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Component string `json:"component,omitempty"`
	Issue     string `json:"issue,omitempty"`
}

func ownedDraftVersion(canvasID, userID uuid.UUID) (*models.CanvasVersion, error) {
	drafts, err := models.ListDraftCanvasVersions(canvasID)
	if err != nil {
		return nil, err
	}

	for i := range drafts {
		if models.IsUserOwnedDraftVersion(&drafts[i], userID) && models.IsRegisteredDraftVersion(&drafts[i]) {
			return &drafts[i], nil
		}
	}

	return nil, nil
}

func selectedVersion(canvas *models.Canvas, draft *models.CanvasVersion, source string) (*models.CanvasVersion, error) {
	if source == "draft" {
		return draft, nil
	}

	if canvas == nil || canvas.LiveVersionID == nil {
		return nil, nil
	}

	version, err := models.FindCanvasVersion(canvas.ID, *canvas.LiveVersionID)
	if err != nil {
		return nil, err
	}

	return version, nil
}

func summarizeNodes(nodes []models.Node, limit int) []superPlaneCanvasNodeSummary {
	count := len(nodes)
	if count > limit {
		count = limit
	}

	result := make([]superPlaneCanvasNodeSummary, 0, count)
	for i := 0; i < count; i++ {
		result = append(result, superPlaneCanvasNodeSummary{
			ID:        nodes[i].ID,
			Name:      nodes[i].Name,
			Type:      nodes[i].Type,
			Component: nodeRefName(nodes[i].Ref),
			Issue:     firstNonEmptyString(stringPtrValue(nodes[i].ErrorMessage), stringPtrValue(nodes[i].WarningMessage)),
		})
	}

	return result
}

func nodeRefName(ref models.NodeRef) string {
	switch {
	case ref.Component != nil:
		return ref.Component.Name
	case ref.Trigger != nil:
		return ref.Trigger.Name
	case ref.Widget != nil:
		return ref.Widget.Name
	default:
		return ""
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
