import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getTriggerRenderer, getStateMap } from "..";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { defaultStateFunction } from "../stateRegistry";
import { MetadataItem } from "@/ui/metadataList";

const AGENT_STATUS_PAYLOAD_TYPE = "cursor.agent.status_change";

export const launchAgentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cursor.launchAgent";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Launch Agent",
      iconSlug: context.componentDefinition.icon ?? "bot",
      iconColor: getColorClass(context.componentDefinition?.color ?? "gray"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: lastExecution ? launchAgentEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: launchAgentMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },
  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const payloadData =
      payload?.data && typeof payload.data === "object"
        ? payload.data
        : (context.execution.metadata as Record<string, any> | undefined);

    if (!payloadData || typeof payloadData !== "object") {
      return details;
    }

    if (payload?.type === AGENT_STATUS_PAYLOAD_TYPE && payloadData.status) {
      const status = payloadData.status as Record<string, unknown>;
      if (status.id) details["Agent ID"] = String(status.id);
      if (status.status) details["Status"] = String(status.status);
      if (status.summary) details["Summary"] = String(status.summary);
      const target = status.target as Record<string, unknown> | undefined;
      if (target?.prUrl) details["PR URL"] = String(target.prUrl);
      if (target?.branchName) details["Branch"] = String(target.branchName);
    }

    const agent = payloadData.agent as Record<string, unknown> | undefined;
    if (agent?.id) details["Agent ID"] = details["Agent ID"] ?? String(agent.id);

    if (payload?.timestamp) {
      details["Emitted At"] = new Date(payload.timestamp as string).toLocaleString();
    }

    return details;
  },
};

function launchAgentMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = (node.configuration ?? {}) as Record<string, unknown>;

  if (configuration.repository) {
    metadata.push({ icon: "git-branch", label: String(configuration.repository) });
  }
  if (configuration.ref) {
    metadata.push({ icon: "git-commit", label: String(configuration.ref) });
  }

  return metadata;
}

function launchAgentEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: defaultStateFunction(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
