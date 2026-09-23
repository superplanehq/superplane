package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBacklogFactoryApp(t *testing.T) {
	componentNode := func(id, component string) Node {
		return Node{ID: id, Ref: NodeRef{Component: &ComponentRef{Name: component}}}
	}
	triggerNode := func(id, trigger string) Node {
		return Node{ID: id, Ref: NodeRef{Trigger: &TriggerRef{Name: trigger}}}
	}

	t.Run("accepts a template marker", func(t *testing.T) {
		node := triggerNode("renamed", "custom")
		node.Metadata = FactoryAppTemplateMetadata(FactoryAppTemplateBacklogID, 1)
		assert.True(t, IsBacklogFactoryApp([]Node{node}, nil))
	})

	t.Run("accepts the exact legacy graph", func(t *testing.T) {
		nodes := []Node{
			triggerNode("trigger", "onWorkOrder"),
			componentNode("analyze", "runnerClaudeCode"),
			componentNode("report-confidence", "reportWorkOrderCheck"),
			componentNode("attach-intent", "addWorkOrderArtifact"),
			componentNode("add-run-error", "addRunError"),
		}
		edges := []Edge{
			{SourceID: "trigger", TargetID: "analyze", Channel: "default"},
			{SourceID: "analyze", TargetID: "report-confidence", Channel: "passed"},
			{SourceID: "analyze", TargetID: "attach-intent", Channel: "passed"},
			{SourceID: "analyze", TargetID: "add-run-error", Channel: "failed"},
		}
		assert.True(t, IsBacklogFactoryApp(nodes, edges))
	})

	t.Run("accepts a customized legacy graph by its stable nodes", func(t *testing.T) {
		nodes := []Node{
			triggerNode("trigger", "onWorkOrder"),
			componentNode("analyze", "runnerCodex"),
			componentNode("report-confidence", "customReporter"),
			componentNode("attach-intent", "customArtifact"),
			componentNode("add-run-error", "customError"),
		}
		assert.True(t, IsBacklogFactoryApp(nodes, nil))
	})

	t.Run("accepts the exact version 2 graph", func(t *testing.T) {
		nodes := []Node{
			triggerNode("trigger", "onWorkOrder"),
			componentNode("task-refinement-enabled", "if"),
			componentNode("analyze", "runnerClaudeCode"),
			componentNode("refine-task", "runnerClaudeCode"),
			componentNode("report-confidence", "reportWorkOrderCheck"),
			componentNode("attach-intent", "addWorkOrderArtifact"),
			componentNode("add-run-error", "addRunError"),
		}
		edges := []Edge{
			{SourceID: "trigger", TargetID: "task-refinement-enabled", Channel: "default"},
			{SourceID: "task-refinement-enabled", TargetID: "analyze", Channel: "false"},
			{SourceID: "task-refinement-enabled", TargetID: "refine-task", Channel: "true"},
			{SourceID: "analyze", TargetID: "report-confidence", Channel: "passed"},
			{SourceID: "analyze", TargetID: "attach-intent", Channel: "passed"},
			{SourceID: "analyze", TargetID: "add-run-error", Channel: "failed"},
			{SourceID: "refine-task", TargetID: "add-run-error", Channel: "failed"},
		}
		assert.True(t, IsBacklogFactoryApp(nodes, edges))
	})

	t.Run("accepts the exact version 3 graph", func(t *testing.T) {
		nodes := []Node{
			triggerNode("trigger", "onWorkOrder"),
			componentNode("task-refinement-enabled", "if"),
			componentNode("refine-task", "runnerClaudeCode"),
			componentNode("add-run-error", "addRunError"),
		}
		edges := []Edge{
			{SourceID: "trigger", TargetID: "task-refinement-enabled", Channel: "default"},
			{SourceID: "task-refinement-enabled", TargetID: "refine-task", Channel: "true"},
			{SourceID: "refine-task", TargetID: "add-run-error", Channel: "failed"},
		}
		assert.True(t, IsBacklogFactoryApp(nodes, edges))
	})

	t.Run("rejects a custom on-work-order canvas", func(t *testing.T) {
		assert.False(t, IsBacklogFactoryApp([]Node{triggerNode("trigger", "onWorkOrder")}, nil))
	})
}

func TestIsLegacyAnalyzeBacklog(t *testing.T) {
	componentNode := func(id, component string) Node {
		return Node{ID: id, Ref: NodeRef{Component: &ComponentRef{Name: component}}}
	}
	triggerNode := func(id, trigger string) Node {
		return Node{ID: id, Ref: NodeRef{Trigger: &TriggerRef{Name: trigger}}}
	}

	t.Run("accepts version 1", func(t *testing.T) {
		nodes := []Node{
			triggerNode("trigger", "onWorkOrder"),
			componentNode("analyze", "runnerClaudeCode"),
			componentNode("report-confidence", "reportWorkOrderCheck"),
			componentNode("attach-intent", "addWorkOrderArtifact"),
			componentNode("add-run-error", "addRunError"),
		}
		assert.True(t, IsLegacyAnalyzeBacklog(nodes))
	})

	t.Run("accepts version 2", func(t *testing.T) {
		nodes := []Node{
			triggerNode("trigger", "onWorkOrder"),
			componentNode("task-refinement-enabled", "if"),
			componentNode("analyze", "runnerClaudeCode"),
			componentNode("refine-task", "runnerClaudeCode"),
			componentNode("report-confidence", "reportWorkOrderCheck"),
			componentNode("attach-intent", "addWorkOrderArtifact"),
			componentNode("add-run-error", "addRunError"),
		}
		assert.True(t, IsLegacyAnalyzeBacklog(nodes))
	})

	t.Run("rejects version 3", func(t *testing.T) {
		nodes := []Node{
			triggerNode("trigger", "onWorkOrder"),
			componentNode("task-refinement-enabled", "if"),
			componentNode("refine-task", "runnerClaudeCode"),
			componentNode("add-run-error", "addRunError"),
		}
		assert.False(t, IsLegacyAnalyzeBacklog(nodes))
	})
}
