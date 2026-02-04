import {
  ComponentsNode,
  ComponentsComponent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import { formatTimeAgo } from "@/utils/date";
import { ListNotesResponse, Note } from "./types";

/**
 * Extracts the first payload from execution outputs.
 */
function getFirstPayload(execution: CanvasesCanvasNodeExecution): OutputPayload | null {
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
function getNotes(execution: CanvasesCanvasNodeExecution): Note[] {
  const payload = getFirstPayload(execution);
  if (!payload || !payload.data) return [];

  const responseData = payload.data as ListNotesResponse | undefined;
  if (!responseData || !responseData.notes) return [];

  return responseData.notes;
}

export const listNotesMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    _?: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.component?.name || "unknown";

    return {
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    const timeAgo = formatTimeAgo(new Date(execution.createdAt!));
    const notes = getNotes(execution);

    if (notes.length > 0) {
      return `${notes.length} note${notes.length === 1 ? "" : "s"} · ${timeAgo}`;
    }

    return `no notes · ${timeAgo}`;
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _: ComponentsNode): Record<string, any> {
    const details: Record<string, any> = {};

    // Add "Checked at" timestamp
    if (execution.createdAt) {
      details["Checked at"] = new Date(execution.createdAt).toLocaleString();
    }

    const notes = getNotes(execution);
    details["Notes"] = `${notes.length} note${notes.length === 1 ? "" : "s"} fetched`;

    return details;
  },
};

function metadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration.incidentId) {
    metadata.push({ icon: "alert-triangle", label: `Incident: ${configuration.incidentId}` });
  }

  return metadata;
}

function baseEventSections(
  nodes: ComponentsNode[],
  execution: CanvasesCanvasNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

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
      eventId: execution.rootEvent!.id!,
    },
  ];
}
