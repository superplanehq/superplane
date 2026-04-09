import { renderTimeAgo } from "@/components/TimeAgo";
import type { EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { getState, getTriggerRenderer } from "..";
import type { DashboardNodeMetadata } from "./types";
import type { ExecutionInfo, NodeInfo } from "../types";

const TEXT_PREVIEW_MAX_LENGTH = 40;

export function buildDashboardMetadata(node: NodeInfo, options?: { dashboardUid?: string }): MetadataItem[] {
  const nodeMetadata = node.metadata as DashboardNodeMetadata | undefined;

  const items: (MetadataItem | undefined)[] = [buildDashboardSelectionMetadata(nodeMetadata, options?.dashboardUid)];

  return items.filter((item): item is MetadataItem => item !== undefined).slice(0, 3);
}

export function buildDashboardSelectionMetadata(
  nodeMetadata: DashboardNodeMetadata | undefined,
  dashboardUid: string | undefined,
): MetadataItem | undefined {
  const label = nodeMetadata?.dashboardTitle?.trim() || dashboardUid?.trim();
  if (!label) {
    return undefined;
  }

  return { icon: "layout-dashboard", label };
}

export function buildPanelMetadata(nodeMetadata: DashboardNodeMetadata | undefined): MetadataItem | undefined {
  const panelLabel = nodeMetadata?.panelLabel?.trim();
  if (panelLabel) {
    return { icon: "hash", label: panelLabel };
  }

  const panelTitle = nodeMetadata?.panelTitle?.trim();
  if (panelTitle) {
    return { icon: "hash", label: panelTitle };
  }

  return undefined;
}

export function buildTimeRangeMetadata(from: string | undefined, to: string | undefined): MetadataItem | undefined {
  const fromLabel = from?.trim();
  const toLabel = to?.trim();

  if (fromLabel && toLabel) {
    return { icon: "clock-3", label: `${fromLabel} -> ${toLabel}` };
  }
  if (fromLabel) {
    return { icon: "clock-3", label: `From: ${fromLabel}` };
  }
  if (toLabel) {
    return { icon: "clock-3", label: `To: ${toLabel}` };
  }

  return undefined;
}

export function previewMetadataItem(
  icon: string,
  prefix: string,
  value: string | number | undefined,
): MetadataItem | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }

  const text = String(value).trim();
  if (!text) {
    return undefined;
  }

  const preview =
    text.length > TEXT_PREVIEW_MAX_LENGTH ? `${text.slice(0, TEXT_PREVIEW_MAX_LENGTH).trimEnd()}...` : text;
  return { icon, label: `${prefix}${preview}` };
}

/** When set, matches the historical Query Data Source node event strip (subtitle + id fallbacks). */
export type BuildGrafanaEventSectionsOptions = {
  legacyQueryDataSource?: boolean;
};

export function buildGrafanaEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
  options?: BuildGrafanaEventSectionsOptions,
): EventSection[] {
  const legacy = options?.legacyQueryDataSource === true;

  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const triggerName = rootTriggerNode?.componentName ?? "";
  const rootTriggerRenderer = getTriggerRenderer(triggerName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const subtitleTimestamp = legacy ? execution.createdAt : execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : legacy ? "-" : "";

  const receivedAtRaw = legacy ? execution.createdAt : execution.createdAt || execution.updatedAt;
  const receivedAt = receivedAtRaw ? new Date(receivedAtRaw) : undefined;

  const eventId = legacy ? (execution.rootEvent?.id ?? "") : (execution.rootEvent?.id ?? execution.id);

  return [
    {
      receivedAt,
      eventTitle: title || "Trigger event",
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId,
    },
  ];
}
