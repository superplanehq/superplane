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

interface UpdateHeartbeatConfiguration {
  team?: string;
  heartbeat?: string;
  description?: string | null;
  interval?: number | null;
  intervalUnit?: string | null;
  enabled?: boolean | null;
  alertMessage?: string | null;
  alertTags?: unknown[] | null;
  alertPriority?: string | null;
}

const FIELD_LABELS: Record<string, string> = {
  description: "Description",
  interval: "Interval",
  enabled: "Enabled",
  alertMessage: "Alert Message",
  alertTags: "Alert Tags",
  alertPriority: "Alert Priority",
};

interface UpdateHeartbeatNodeMetadata {
  teamName?: string;
}

export const updateHeartbeatMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "jira.updateHeartbeat";

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
    if (!data) {
      return details;
    }
    if (data.name != null) {
      details["Name"] = String(data.name);
    }
    if (data.description != null) {
      details["Description"] = String(data.description);
    }
    if (data.interval != null) {
      const unit = data.intervalUnit != null ? ` ${data.intervalUnit}` : "";
      details["Interval"] = `${data.interval}${unit}`;
    }
    if (data.status != null) {
      details["Status"] = String(data.status);
    }
    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as UpdateHeartbeatConfiguration;
  const nodeMetadata = node.metadata as UpdateHeartbeatNodeMetadata | undefined;
  const items: MetadataItem[] = [];

  const teamLabel = nodeMetadata?.teamName || configuration?.team;
  if (teamLabel) {
    items.push({ icon: "users", label: teamLabel });
  }
  if (configuration?.heartbeat) {
    items.push({ icon: "activity", label: configuration.heartbeat });
  }

  const updated = listUpdatedFields(configuration);
  if (updated.length > 0) {
    items.push({ icon: "edit", label: `Updates: ${updated.join(", ")}` });
  }

  return items;
}

function listUpdatedFields(configuration: UpdateHeartbeatConfiguration | undefined): string[] {
  if (!configuration) return [];
  const skip = new Set(["team", "heartbeat", "intervalUnit"]);
  const updated: string[] = [];
  (Object.keys(configuration) as (keyof UpdateHeartbeatConfiguration)[]).forEach((key) => {
    if (skip.has(key)) return;
    const value = configuration[key];
    if (value === undefined || value === null) return;
    if (typeof value === "string" && value.trim() === "") return;
    if (Array.isArray(value) && value.length === 0) return;
    updated.push(FIELD_LABELS[key] ?? key);
  });
  return updated;
}
