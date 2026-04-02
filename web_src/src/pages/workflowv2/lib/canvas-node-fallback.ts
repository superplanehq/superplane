import { Puzzle } from "lucide-react";
import type { ComponentType } from "react";
import type { ComponentsNode, ComponentsComponent, TriggersTrigger } from "@/api-client";
const CANVAS_NODE_FALLBACK_MESSAGE = "Can't display";
import type { CanvasNode } from "@/ui/CanvasPage";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";

function buildMinimalRenderFallback() {
  return {
    source: "mapper" as const,
    message: CANVAS_NODE_FALLBACK_MESSAGE,
  };
}

function buildMinimalEmptyStateProps(icon?: ComponentType<{ size?: number }>) {
  return {
    icon,
    title: CANVAS_NODE_FALLBACK_MESSAGE,
    description: undefined,
  };
}

export function buildTriggerFallbackCanvasNode({
  node,
  displayLabel,
  triggerMetadata,
}: {
  node: ComponentsNode;
  displayLabel: string;
  triggerMetadata?: TriggersTrigger;
}): CanvasNode {
  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "trigger",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: ["default"],
      renderFallback: buildMinimalRenderFallback(),
      trigger: {
        title: displayLabel,
        iconSlug: triggerMetadata?.icon || "bolt",
        metadata: [],
        collapsed: node.isCollapsed,
        includeEmptyState: true,
        emptyStateProps: buildMinimalEmptyStateProps(),
      },
    },
  };
}

export function buildComponentFallbackCanvasNode({
  node,
  displayLabel,
  metadata,
}: {
  node: ComponentsNode;
  displayLabel: string;
  metadata?: ComponentsComponent;
}): CanvasNode {
  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "component",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: metadata?.outputChannels?.map((channel) => channel.name) || ["default"],
      renderFallback: buildMinimalRenderFallback(),
      component: {
        iconSlug: metadata?.icon || "triangle-alert",
        iconColor: getColorClass(metadata?.color || "gray"),
        collapsedBackground: getBackgroundColorClass(metadata?.color || "gray"),
        collapsed: node.isCollapsed,
        title: displayLabel,
        includeEmptyState: true,
        emptyStateProps: buildMinimalEmptyStateProps(Puzzle),
        warning: node.warningMessage,
        paused: !!node.paused,
      },
    },
  };
}
