import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "./types";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from ".";
import { renderTimeAgo } from "@/components/TimeAgo";

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
    const muted = !!context.node.paused || !lastExecution;

    return {
      iconSlug: context.componentDefinition.icon || "tag",
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution
        ? getDisplayEventSections(context.nodes, lastExecution, componentName, displayResult, muted)
        : undefined,
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

function renderDisplayBadge(result: DisplayResult, muted: boolean): React.ReactNode {
  if (!result.value) {
    return null;
  }

  const color = normalizeDisplayColor(result.color);
  const badgeClasses = DISPLAY_BADGE_CLASSES[color] || DISPLAY_BADGE_CLASSES.gray;
  const displayValue = truncate(result.value, DISPLAY_BADGE_MAX_CHARS);

  return (
    <span
      title={result.value}
      className={`inline-flex max-w-full items-center rounded-md px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${badgeClasses} ${muted ? "opacity-60" : ""}`}
    >
      <span className="truncate">{displayValue}</span>
    </span>
  );
}

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

function getDisplayEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
  displayResult: DisplayResult,
  muted: boolean,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";
  const displayBadge = renderDisplayBadge(displayResult, muted);

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: displayBadge ?? title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
