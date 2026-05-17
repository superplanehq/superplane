import { renderTimeAgo } from "@/components/TimeAgo";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getStateMap } from ".";

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

const DISPLAY_BADGE_CLASSES: Record<string, string> = {
  green: "bg-emerald-100 text-emerald-800 ring-emerald-200",
  yellow: "bg-yellow-100 text-yellow-800 ring-yellow-200",
  red: "bg-red-100 text-red-800 ring-red-200",
  blue: "bg-blue-100 text-blue-800 ring-blue-200",
  gray: "bg-gray-100 text-gray-700 ring-gray-200",
};

const DISPLAY_BADGE_DEFAULT_COLOR = "gray";

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

  const message = lastExecution?.metadata?.display_result?.message;
  if (!message) {
    return null;
  }

  return (
    <div className="px-2 py-1.5 text-left text-base max-h-20 truncate">
      <pre>{message}</pre>
    </div>
  );
}

function resolveDisplayResult(execution: ExecutionInfo | null): DisplayResult {
  const metadata = execution?.metadata as { display_result?: DisplayResult } | undefined;
  const fromMetadata = metadata?.display_result;
  if (fromMetadata?.message) {
    return {
      message: String(fromMetadata.message),
      color: normalizeDisplayColor(fromMetadata.color),
    };
  }
  return {};
}

function normalizeDisplayColor(color?: string): string {
  const normalized = (color || DISPLAY_BADGE_DEFAULT_COLOR).toLowerCase().trim();
  return DISPLAY_BADGE_CLASSES[normalized] ? normalized : DISPLAY_BADGE_DEFAULT_COLOR;
}
