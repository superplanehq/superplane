import { ComponentBaseProps } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

interface UpdateIssueConfiguration {
  issueId?: string;
  status?: string;
  assignedTo?: string;
}

interface SentryIssue {
  id?: string;
  shortId?: string;
  title?: string;
  status?: string;
  project?: {
    name?: string;
    slug?: string;
  };
  assignedTo?: {
    name?: string;
  };
}

export const updateIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: sentryIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: buildMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as SentryIssue | undefined;
    const timestamp = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt));
    return [issue?.shortId || issue?.title, timestamp].filter(Boolean).join(" · ");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as SentryIssue | undefined;
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Started At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (context.execution.updatedAt) {
      details["Last Updated At"] = new Date(context.execution.updatedAt).toLocaleString();
    }

    if (issue) {
      if (issue.id) details["Issue ID"] = issue.id;
      if (issue.shortId) details["Short ID"] = issue.shortId;
      if (issue.title) details["Title"] = issue.title;
      if (issue.status) details["Status"] = issue.status;
      if (issue.project?.name || issue.project?.slug)
        details["Project"] = issue.project?.name || issue.project?.slug || "";
      if (issue.assignedTo?.name) details["Assigned To"] = issue.assignedTo.name;
    }

    return details;
  },
};

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string) {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as UpdateIssueConfiguration | undefined;
  const metadata = [];

  if (configuration?.issueId) {
    metadata.push({ icon: "hash", label: configuration.issueId });
  }

  if (configuration?.status) {
    metadata.push({ icon: "check-circle-2", label: configuration.status });
  }

  if (configuration?.assignedTo) {
    metadata.push({ icon: "user", label: configuration.assignedTo });
  }

  return metadata;
}
