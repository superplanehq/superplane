import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ListNotesResponse, Note } from "./types";

/**
 * Extracts the first payload from execution outputs.
 */
function getFirstPayload(execution: ExecutionInfo): OutputPayload | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  if (!outputs) return null;

  if (outputs.default && outputs.default.length > 0) {
    return outputs.default[0];
  }

  return null;
}

/**
 * Extracts notes from the execution payload.
 */
function getNotes(execution: ExecutionInfo): Note[] {
  const payload = getFirstPayload(execution);
  if (!payload || !payload.data) return [];

  const responseData = payload.data as ListNotesResponse | undefined;
  if (!responseData || !responseData.notes) return [];

  return responseData.notes;
}

export const listNotesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution =
      context.lastExecutions && context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition?.name || "unknown";

    return {
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition?.color),
      collapsed: context.node?.isCollapsed ?? false,
      title:
        context.node?.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const timeAgo = formatTimeAgo(new Date(context.execution.createdAt!));
    const notes = getNotes(context.execution);

    if (notes.length > 0) {
      return `${notes.length} note${notes.length === 1 ? "" : "s"} · ${timeAgo}`;
    }

    return `no notes · ${timeAgo}`;
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};

    // Add "Checked at" timestamp
    if (context.execution.createdAt) {
      details["Checked at"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const notes = getNotes(context.execution);
    details["Notes"] = `${notes.length} note${notes.length === 1 ? "" : "s"} fetched`;

    return details;
  },
};

function metadataList(node: { configuration?: unknown }): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  if (!node) return metadata;
  const configuration = node.configuration as any;

  if (configuration.incidentId) {
    metadata.push({ icon: "alert-triangle", label: `Incident: ${configuration.incidentId}` });
  }

  return metadata;
}

function baseEventSections(nodes: { id: string }[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer((rootTriggerNode as any)?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  const notes = getNotes(execution);
  const timeAgo = formatTimeAgo(new Date(execution.createdAt!));

  let eventSubtitle: string;
  if (notes.length > 0) {
    eventSubtitle = `${notes.length} note${notes.length === 1 ? "" : "s"} · ${timeAgo}`;
  } else {
    eventSubtitle = `no notes · ${timeAgo}`;
  }

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id,
    },
  ];
}
