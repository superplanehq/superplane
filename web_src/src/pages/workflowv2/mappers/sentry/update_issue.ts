import type { ComponentBaseProps } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { addDetail, addFormattedTimestamp, getProjectLabel } from "./utils";
import type {
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
}

interface UpdateIssueNodeMetadata {
  issueTitle?: string;
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

    addFormattedTimestamp(details, "Started At", context.execution.createdAt);
    addFormattedTimestamp(details, "Last Updated At", context.execution.updatedAt);

    addDetail(details, "Issue ID", issue?.id);
    addDetail(details, "Short ID", issue?.shortId);
    addDetail(details, "Title", issue?.title);
    addDetail(details, "Status", issue?.status);
    addDetail(details, "Project", getProjectLabel(issue));
    addDetail(details, "Assigned To", issue?.assignedTo?.name);

    return details;
  },
};

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string) {
  const rootEvent = execution.rootEvent;
  const createdAt = execution.createdAt;
  const rootTriggerNode = nodes.find((n) => n.id === rootEvent?.nodeId);
  const rootComponentName = rootTriggerNode?.componentName;

  if (!rootEvent || !createdAt || !rootComponentName) {
    return undefined;
  }

  const rootTriggerRenderer = getTriggerRenderer(rootComponentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(createdAt),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id || "",
    },
  ];
}

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as UpdateIssueConfiguration | undefined;
  const nodeMetadata = node.metadata as UpdateIssueNodeMetadata | undefined;
  const metadata = [];

  const issueLabel = nodeMetadata?.issueTitle || configuration?.issueId;
  if (issueLabel) {
    metadata.push({ icon: "bug", label: issueLabel });
  }

  if (configuration?.status) {
    metadata.push({ icon: "check-circle-2", label: configuration.status });
  }

  return metadata;
}
