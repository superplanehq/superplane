import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "../mapperLookup";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import cursorIcon from "@/assets/icons/integrations/cursor.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

type LaunchAgentMetadata = {
  agent?: { id?: string; url?: string; status?: string; summary?: string };
  target?: { prUrl?: string; branchName?: string };
};

type LaunchAgentPayloadData = {
  status?: string;
  agentId?: string;
  prUrl?: string;
  summary?: string;
  branchName?: string;
};

function addDetail(details: Record<string, string>, key: string, value: string | undefined) {
  if (value) {
    details[key] = value;
  }
}

function resolveAgentDetails(
  metadata: LaunchAgentMetadata | undefined,
  data: LaunchAgentPayloadData | undefined,
): Record<string, string> {
  const agent = metadata?.agent ?? {};
  const target = metadata?.target ?? {};
  const payloadData = data ?? {};
  const details: Record<string, string> = {};

  addDetail(details, "Cloud Agent", agent.url);
  addDetail(details, "Pull Request", target.prUrl ?? payloadData.prUrl);
  addDetail(details, "Branch", target.branchName ?? payloadData.branchName);
  addDetail(details, "Status", payloadData.status ?? agent.status);
  addDetail(details, "Summary", payloadData.summary ?? agent.summary);

  return details;
}

export const launchAgentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cursor";

    return {
      iconSrc: cursorIcon,
      iconSlug: context.componentDefinition?.icon ?? "cpu",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Launch Cloud Agent",
      eventSections: lastExecution ? launchAgentEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const metadata = context.execution.metadata as LaunchAgentMetadata | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as LaunchAgentPayloadData | undefined;
    const details = resolveAgentDetails(metadata, data);

    if (payload?.timestamp) {
      details["Emitted At"] = new Date(payload.timestamp).toLocaleString();
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function launchAgentEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";

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
