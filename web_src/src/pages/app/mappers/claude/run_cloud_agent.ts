import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import claudeIcon from "@/assets/icons/integrations/claude.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

type RunCloudAgentNodeMetadata = {
  agentId?: string;
  agentName?: string;
  environmentId?: string;
  environmentName?: string;
};

type RunCloudAgentConfiguration = {
  agent?: string;
  environmentId?: string;
  repository?: string;
  branch?: string;
};

type RunCloudAgentExecutionMetadata = {
  repository?: string;
  branch?: string;
  session?: { id?: string; status?: string };
};

type RunCloudAgentPayloadData = {
  status?: string;
  sessionId?: string;
  lastMessage?: string;
};

function addDetail(details: Record<string, string>, key: string, value: string | undefined) {
  if (value) {
    details[key] = value;
  }
}

export const runCloudAgentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "claude";

    return {
      iconSrc: claudeIcon,
      iconSlug: context.componentDefinition?.icon ?? "bot",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Run Claude Cloud Agent",
      eventSections: lastExecution
        ? runCloudAgentEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const metadata = (context.execution.metadata ?? {}) as RunCloudAgentExecutionMetadata;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = (payload?.data ?? {}) as RunCloudAgentPayloadData;
    const session = metadata.session ?? {};

    const entries: Array<[string, string | undefined]> = [
      ["Repository", metadata.repository],
      ["Branch", metadata.branch],
      ["Status", data.status || session.status],
      ["Last Message", data.lastMessage],
    ];
    for (const [key, value] of entries) {
      addDetail(details, key, value);
    }

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

// metadataList renders the configured agent and environment (resolved to their
// names at Setup time) on the component card, independent of any execution.
function metadataList(node: NodeInfo): MetadataItem[] {
  const items: MetadataItem[] = [];
  const meta = (node.metadata ?? {}) as RunCloudAgentNodeMetadata;
  const config = (node.configuration ?? {}) as RunCloudAgentConfiguration;

  const agent = meta.agentName || meta.agentId || config.agent;
  if (agent) {
    items.push({ icon: "bot", label: agent });
  }

  const environment = meta.environmentName || meta.environmentId || config.environmentId;
  if (environment) {
    items.push({ icon: "box", label: environment });
  }

  if (config.repository) {
    items.push({ icon: "git-branch", label: config.repository });
  }

  return items;
}

function runCloudAgentEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
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
