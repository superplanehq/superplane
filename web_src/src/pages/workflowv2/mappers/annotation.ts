import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps } from "@/ui/componentBase";
import { parseBasicMarkdown } from "@/utils/markdown";
import React from "react";

export const annotationMapper: ComponentBaseMapper = {
  props(
    _nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    _lastExecutions: WorkflowsWorkflowNodeExecution[],
    _nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const content = (node.configuration?.content as string) || "";
    const displayText = content || "Configure this component to add your annotation...";
    const hasContent = !!content;

    return {
      iconSlug: componentDefinition.icon || "sticky-note",
      headerColor: "bg-white",
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-gray-100",
      title: node.name!,
      customField: React.createElement(
        "div",
        {
          className: `px-3 py-2 text-sm border-t border-gray-200 text-left leading-snug ${
            hasContent ? "text-gray-900 bg-amber-50" : "text-gray-400 bg-gray-50 italic"
          }`,
          dangerouslySetInnerHTML: hasContent ? { __html: parseBasicMarkdown(content) } : undefined,
        },
        hasContent ? undefined : displayText,
      ),
      includeEmptyState: false, // Never show "No executions received yet" for display-only component
    };
  },
};
