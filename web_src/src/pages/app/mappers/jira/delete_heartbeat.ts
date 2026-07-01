import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { jiraBaseEventSections } from "./base";

interface DeleteHeartbeatConfiguration {
  team?: string;
  heartbeat?: string;
}

interface DeleteHeartbeatNodeMetadata {
  teamName?: string;
}

export const deleteHeartbeatMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "jira.deleteHeartbeat";

    return {
      iconSrc: jiraIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? jiraBaseEventSections(context.nodes, lastExecution, componentName) : undefined,
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
    const outputs = context.execution.outputs as { default?: Array<{ data?: unknown }> } | undefined;
    const data = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
    if (data?.deleted === true) {
      details["Deleted"] = "true";
    }
    if (data?.name != null) {
      details["Name"] = String(data.name);
    }
    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as DeleteHeartbeatConfiguration;
  const nodeMetadata = node.metadata as DeleteHeartbeatNodeMetadata | undefined;
  const items: MetadataItem[] = [];

  const teamLabel = nodeMetadata?.teamName || configuration?.team;
  if (teamLabel) {
    items.push({ icon: "users", label: teamLabel });
  }
  if (configuration?.heartbeat) {
    items.push({ icon: "activity", label: configuration.heartbeat });
  }

  return items;
}
