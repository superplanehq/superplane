import { EventSection } from "@/ui/componentBase";
import { formatTimeAgo } from "@/utils/date";
import { getState, getTriggerRenderer } from "..";
import { ExecutionInfo, NodeInfo, OutputPayload } from "../types";

// getFirstDefaultPayload returns the first payload emitted on the default channel.
export function getFirstDefaultPayload(execution: ExecutionInfo): OutputPayload | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  if (!outputs?.default || outputs.default.length === 0) {
    return null;
  }

  return outputs.default[0] || null;
}

// buildDash0ExecutionDetails converts Dash0 execution payload data into sidebar details.
export function buildDash0ExecutionDetails(execution: ExecutionInfo, label: string): Record<string, string> {
  const details: Record<string, string> = {};
  const payload = getFirstDefaultPayload(execution);

  if (payload?.timestamp) {
    details["Received At"] = new Date(payload.timestamp).toLocaleString();
  }

  if (!payload?.data) {
    details[label] = "No data returned";
    return details;
  }

  try {
    details[label] = JSON.stringify(payload.data, null, 2);
  } catch {
    details[label] = String(payload.data);
  }

  return details;
}

// buildDash0EventSections creates the event timeline section for Dash0 action mappers.
export function buildDash0EventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
  subtitlePrefix?: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const timeAgo = formatTimeAgo(new Date(execution.createdAt!));

  let subtitle = timeAgo;
  if (subtitlePrefix) {
    subtitle = `${subtitlePrefix} · ${timeAgo}`;
  }

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: subtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
