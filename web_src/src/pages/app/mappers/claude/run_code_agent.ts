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
import { resourceLabel } from "../utils";
import type { MetadataItem } from "@/ui/metadataList";
import claudeIcon from "@/assets/icons/integrations/claude.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

type RunCodeAgentNodeMetadata = {
  repository?: unknown;
  baseBranch?: unknown;
  prUrl?: unknown;
  model?: unknown;
  sourceMode?: string;
};

type RunCodeAgentConfiguration = {
  sourceMode?: string;
  repository?: unknown;
  baseBranch?: unknown;
  prUrl?: unknown;
  model?: unknown;
};

type RunCodeAgentPayloadData = {
  status?: string;
  sessionId?: string;
  prUrl?: string;
  branch?: string;
  lastMessage?: string;
};

function addDetail(details: Record<string, string>, key: string, value: string | undefined) {
  if (value) {
    details[key] = value;
  }
}

export const runCodeAgentMapper: ComponentBaseMapper = {
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
        "Run Code Agent",
      eventSections: lastExecution ? runCodeAgentEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = (outputs?.default?.[0]?.data ?? {}) as RunCodeAgentPayloadData;

    const entries: Array<[string, string | undefined]> = [
      ["Status", data.status],
      ["Pull Request", data.prUrl],
      ["Branch", data.branch],
    ];
    for (const [key, value] of entries) {
      addDetail(details, key, value);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

// metadataList shows the configured target on the component card, independent of
// any execution (names resolved at Setup, with config as a fallback).
function metadataList(node: NodeInfo): MetadataItem[] {
  const items: MetadataItem[] = [];
  const meta = (node.metadata ?? {}) as RunCodeAgentNodeMetadata;
  const config = (node.configuration ?? {}) as RunCodeAgentConfiguration;

  const isPR = (meta.sourceMode ?? config.sourceMode) === "pr";
  if (isPR) {
    const pr = resourceLabel(meta.prUrl) ?? resourceLabel(config.prUrl);
    if (pr) {
      items.push({ icon: "git-pull-request", label: pr });
    }
  } else {
    const repo = resourceLabel(meta.repository) ?? resourceLabel(config.repository);
    if (repo) {
      items.push({ icon: "git-branch", label: repo });
    }
    const baseBranch = resourceLabel(meta.baseBranch) ?? resourceLabel(config.baseBranch);
    if (baseBranch) {
      items.push({ icon: "git-branch", label: baseBranch });
    }
  }

  const model = resourceLabel(meta.model) ?? resourceLabel(config.model);
  if (model) {
    items.push({ icon: "bot", label: model });
  }

  return items;
}

function runCodeAgentEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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
