import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps } from "@/ui/componentBase";

export const annotationMapper: ComponentBaseMapper = {
  props(
    _nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    _lastExecutions: WorkflowsWorkflowNodeExecution[],
    _nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const content = node.configuration?.content || "";

    return {
      iconSlug: componentDefinition.icon || "sticky-note",
      headerColor: "bg-white",
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-gray-100",
      title: node.name!,
      specs: content
        ? [
            {
              title: "Content",
              value: content,
              contentType: "text",
            },
          ]
        : undefined,
      includeEmptyState: false, // Never show "No executions received yet" for display-only component
      hideActionsButton: true, // Hide Run/Configure action menu
    };
  },
};
