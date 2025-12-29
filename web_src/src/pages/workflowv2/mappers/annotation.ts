import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps } from "@/ui/componentBase";
import React from "react";

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
      customField: content
        ? React.createElement(
            "div",
            { className: "px-3 py-2 text-sm text-gray-700 whitespace-pre-wrap border-t border-gray-200 text-left bg-amber-50 font-bold" },
            content,
          )
        : undefined,
      includeEmptyState: false, // Never show "No executions received yet" for display-only component
    };
  },
};
