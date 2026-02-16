import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { formatTimeAgo } from "@/utils/date";

export const updateIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "sentry";

    return {
      iconSrc: sentryIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const data = context.execution.outputs?.["data"] as any;
    const issue = data?.["issue"] as any;
    const details: Record<string, any> = {};

    if (issue) {
      details["Issue"] = issue.shortId || issue.id;
      details["Title"] = issue.title;
      details["Status"] = issue.status;
      details["Level"] = issue.level;
      if (issue.project) {
        details["Project"] = issue.project.name;
      }
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration.issueId) {
    metadata.push({ icon: "alert-circle", label: `Issue: ${configuration.issueId}` });
  }

  // Show which fields are being updated
  const updates: string[] = [];
  if (configuration.status) {
    updates.push(`Status: ${configuration.status}`);
  }
  if (configuration.assignedTo) {
    updates.push("Assigned");
  }

  if (updates.length > 0) {
    metadata.push({ icon: "funnel", label: `Updating: ${updates.join(", ")}` });
  }

  return metadata;
}

function baseEventSections(_nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: execution.rootEvent?.data?.issue?.title || "Issue Event",
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
