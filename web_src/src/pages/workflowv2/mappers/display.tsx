import { renderTimeAgo } from "@/components/TimeAgo";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
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
  value?: string;
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
const DISPLAY_BADGE_MAX_CHARS = 60;

export const displayMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const componentName = context.componentDefinition.name || "display";
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const displayResult = resolveDisplayResult(lastExecution);
    const title =
      context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Unnamed component";

    return {
      iconSlug: context.componentDefinition.icon || "tag",
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title,
      eventSections: lastExecution ? getDisplayEventSections(displayResult.value ?? "") : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const result = resolveDisplayResult(context.execution);
    if (!result.value) {
      return {};
    }

    return {
      Value: result.value,
      Color: normalizeDisplayColor(result.color),
    };
  },
};

function resolveDisplayResult(execution: ExecutionInfo | null): DisplayResult {
  const metadata = execution?.metadata as { display_result?: DisplayResult } | undefined;
  const fromMetadata = metadata?.display_result;
  if (fromMetadata?.value) {
    return {
      value: String(fromMetadata.value),
      color: normalizeDisplayColor(fromMetadata.color),
    };
  }
  return {};
}

function normalizeDisplayColor(color?: string): string {
  const normalized = (color || DISPLAY_BADGE_DEFAULT_COLOR).toLowerCase().trim();
  return DISPLAY_BADGE_CLASSES[normalized] ? normalized : DISPLAY_BADGE_DEFAULT_COLOR;
}

function truncate(value: string, maxLength: number): string {
  if (value.length <= maxLength) {
    return value;
  }

  return `${value.slice(0, maxLength - 1)}…`;
}

function getDisplayEventSections(value: string): EventSection[] {
  const message = truncate(value, DISPLAY_BADGE_MAX_CHARS);

  return [
    {
      eventTitle: <pre>{message}</pre>,
      eventId: "",
    },
  ];
}
