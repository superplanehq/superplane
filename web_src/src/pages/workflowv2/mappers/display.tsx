import { renderTimeAgo } from "@/components/TimeAgo";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getStateMap } from ".";
import { getBackgroundColorClass } from "../../../lib/colors";

import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  SubtitleContext,
} from "./types";

type DisplayResult = {
  message?: string;
  color?: string;
};

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

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const result = resolveDisplayResult(context.execution);
    if (!result.message) {
      return {};
    }

    return {
      Message: result.message,
      Color: normalizeDisplayColor(result.color),
    };
  },
};

function Message({ lastExecution }: { lastExecution: ExecutionInfo | null }): React.ReactNode {
  if (!lastExecution) {
    return null;
  }

  const metadata = lastExecution?.metadata as Record<string, unknown>;

  const message = metadata["message"] as string | undefined;
  if (!message) {
    return null;
  }

  const colorClass = getBackgroundColorClass(normalizeDisplayColor(metadata["color"] as string | undefined));

  return (
    <div className={`px-2 py-1.5 text-left text-base max-h-20 truncate ${colorClass}`}>
      <pre className="break-all whitespace-pre-wrap">{message}</pre>
    </div>
  );
}

const DISPLAY_COLORS = new Set(["green", "yellow", "red", "blue", "gray"]);

function resolveDisplayResult(execution: ExecutionInfo | null): DisplayResult {
  const metadata = execution?.metadata as Record<string, unknown> | undefined;
  const message = metadata?.message;
  if (typeof message !== "string" || !message) {
    return {};
  }

  return {
    message,
    color: normalizeDisplayColor(typeof metadata?.color === "string" ? metadata.color : undefined),
  };
}

function normalizeDisplayColor(color?: string): string {
  const normalized = (color || "gray").toLowerCase().trim();
  return DISPLAY_COLORS.has(normalized) ? normalized : "gray";
}
