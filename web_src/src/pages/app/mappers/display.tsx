import { renderTimeAgo } from "@/components/TimeAgo";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getStateMap } from ".";

import { Message } from "./display/Message";
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "./types";

export const displayMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const componentName = context.componentDefinition.name || "display";
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentDefinition = context.componentDefinition;
    const title = context.node.name || componentDefinition.label || componentDefinition.name || "Unnamed component";

    return {
      iconSlug: "monitor",
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title: title,
      eventSections: [],
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
      customField: <Message lastExecution={lastExecution} />,
      customFieldPosition: "before",
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(_context: ExecutionDetailsContext): Record<string, string> {
    return {};
  },
};
