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
import { jiraBaseEventSections, buildOpsAlertReferenceMetadata, opsAlertCoreExecutionPayloadDetails } from "./base";

interface UpdateAlertConfiguration {
  alert?: string;
}

export const updateAlertMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "jira.updateAlert";

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
    Object.assign(details, opsAlertCoreExecutionPayloadDetails(data));
    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

interface UpdateAlertNodeMeta {
  updateSummaries?: string[];
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as UpdateAlertConfiguration;
  const items = buildOpsAlertReferenceMetadata(node, configuration?.alert);

  const nodeMeta = node.metadata as UpdateAlertNodeMeta | undefined;
  const summaries = nodeMeta?.updateSummaries;
  if (Array.isArray(summaries) && summaries.length > 0) {
    for (const raw of summaries) {
      const line = typeof raw === "string" ? raw.trim() : "";
      if (line !== "") {
        items.push({ icon: "corner-down-right", label: line });
      }
    }
  }
  return items;
}
