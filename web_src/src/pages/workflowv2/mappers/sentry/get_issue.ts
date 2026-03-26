import type { ComponentBaseProps } from "@/ui/componentBase";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { addFormattedTimestamp, addOrderedDetails, buildEventSections, getProjectLabel } from "./utils";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

interface GetIssueConfiguration {
  issueId?: string;
}

interface GetIssueNodeMetadata {
  issueTitle?: string;
}

interface SentryIssueOutput {
  shortId?: string;
  title?: string;
  status?: string;
  count?: string;
  permalink?: string;
  web_url?: string;
  assignedTo?: {
    name?: string;
  };
  project?: {
    name?: string;
    slug?: string;
  };
  events?: Array<unknown>;
}

export const getIssueMapper: ComponentBaseMapper = {
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
      eventSections: lastExecution
        ? buildEventSections(context.nodes, lastExecution, componentName, getTriggerRenderer, getState)
        : undefined,
      metadata: buildMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as SentryIssueOutput | undefined;
    const timestamp = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt));
    return [issue?.shortId || issue?.title, timestamp].filter(Boolean).join(" · ");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as SentryIssueOutput | undefined;
    const details: Record<string, string> = {};

    addFormattedTimestamp(details, "Retrieved At", context.execution.createdAt);
    addOrderedDetails(details, [
      { label: "Title", value: issue?.title },
      { label: "Status", value: issue?.status },
      { label: "Assigned To", value: issue?.assignedTo?.name },
      { label: "Frequency", value: getIssueFrequency(issue) },
      { label: "Issue URL", value: getIssueURL(issue) },
      { label: "Project", value: getProjectLabel(issue) },
      { label: "Recent Events", value: getIssueEventCount(issue) },
    ]);

    return details;
  },
};

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as GetIssueConfiguration | undefined;
  const nodeMetadata = node.metadata as GetIssueNodeMetadata | undefined;
  const metadata = [];

  if (nodeMetadata?.issueTitle || configuration?.issueId) {
    metadata.push({ icon: "bug", label: nodeMetadata?.issueTitle || configuration?.issueId });
  }

  return metadata;
}

function getIssueURL(issue: SentryIssueOutput | undefined) {
  return issue?.web_url || issue?.permalink;
}

function getIssueFrequency(issue: SentryIssueOutput | undefined) {
  return issue?.count ? `${issue.count} events` : undefined;
}

function getIssueEventCount(issue: SentryIssueOutput | undefined) {
  return issue?.events?.length ? String(issue.events.length) : undefined;
}
