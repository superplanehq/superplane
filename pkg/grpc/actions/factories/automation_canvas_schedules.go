package factories

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const customAutomationScheduleLogicalIDKey = "automationTriggerId"

func setCustomAutomationScheduleMetadata(template map[string]any, trigger customAutomationTrigger) {
	if trigger.LogicalID != "" {
		template[customAutomationScheduleLogicalIDKey] = trigger.LogicalID
	}
}

func customAutomationScheduleLogicalID(node models.Node, count int) string {
	template, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
	if id := configString(template, customAutomationScheduleLogicalIDKey); id != "" {
		return id
	}
	if count == 1 {
		return "schedule"
	}
	digest := sha256.Sum256([]byte(node.ID))
	return fmt.Sprintf("schedule-%x", digest[:12])
}

func preserveCustomAutomationScheduleIdentities(live []models.Node, graph *yaml.Canvas) {
	var schedules []models.Node
	for _, node := range live {
		if node.ComponentName() == "schedule" {
			schedules = append(schedules, node)
		}
	}
	if len(schedules) == 0 {
		return
	}
	// Form saves provide no logical IDs. Match equal schedules in saved order.
	slices.SortStableFunc(schedules, func(a, b models.Node) int {
		first, _ := a.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		second, _ := b.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		return configInt(first, "triggerIndex") - configInt(second, "triggerIndex")
	})
	count := len(schedules)
	for i := range graph.Spec.Nodes {
		node := &graph.Spec.Nodes[i]
		if node.Component != "schedule" {
			continue
		}
		template, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		id := configString(template, customAutomationScheduleLogicalIDKey)
		match := slices.IndexFunc(schedules, func(live models.Node) bool {
			return (id == "" || id == customAutomationScheduleLogicalID(live, count)) &&
				reflect.DeepEqual(live.Configuration, node.Configuration)
		})
		if match == -1 {
			continue
		}
		previous := schedules[match]
		renameCustomAutomationNode(graph, node, previous.ID)
		savedTemplate, _ := previous.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		if id == "" {
			if savedID := configString(savedTemplate, customAutomationScheduleLogicalIDKey); savedID != "" {
				template[customAutomationScheduleLogicalIDKey] = savedID
			}
		}
		schedules = slices.Delete(schedules, match, match+1)
	}
}

func renameCustomAutomationNode(graph *yaml.Canvas, node *yaml.Node, nextID string) {
	oldID := node.ID
	node.ID = nextID
	for i := range graph.Spec.Edges {
		edge := &graph.Spec.Edges[i]
		if edge.TargetID == oldID {
			edge.TargetID = nextID
		}
		if edge.SourceID == oldID {
			edge.SourceID = nextID
		}
	}
}

// The caller locks the canvas before retiring deliveries so the scheduler
// cannot claim a request from the version that is about to be replaced.
func retireCustomAutomationSchedules(tx *gorm.DB, canvasID uuid.UUID, live, proposed []models.Node) error {
	retained := make(map[string]bool, len(proposed))
	for _, node := range proposed {
		if node.ComponentName() == "schedule" {
			retained[node.ID] = true
		}
	}
	for _, node := range live {
		if node.ComponentName() != "schedule" || retained[node.ID] {
			continue
		}
		request, err := models.FindPendingRequestForNode(tx, canvasID, node.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if err := request.Complete(tx); err != nil {
			return err
		}
	}
	return nil
}
