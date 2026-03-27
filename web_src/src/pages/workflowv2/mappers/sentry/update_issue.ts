import type { ComponentBaseProps } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { addDetail, addFormattedTimestamp, buildEventSections, getProjectLabel } from "./utils";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

interface UpdateIssueConfiguration {
  issueId?: string;
  status?: string;
  priority?: string;
  assignedTo?: string;
  hasSeen?: boolean;
  isPublic?: boolean;
  isSubscribed?: boolean;
}

interface UpdateIssueNodeMetadata {
  issueTitle?: string;
  assigneeLabel?: string;
}

interface SentryIssue {
  id?: string;
  shortId?: string;
  title?: string;
  status?: string;
  priority?: string;
  hasSeen?: boolean;
  isPublic?: boolean;
  isSubscribed?: boolean;
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
    const issue = outputs?.default?.[0]?.data as SentryIssue | undefined;
    const timestamp = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt));
    return [issue?.shortId || issue?.title, timestamp].filter(Boolean).join(" · ");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as SentryIssue | undefined;
    const details: Record<string, string> = {};

    addFormattedTimestamp(details, "Triggered At", context.execution.createdAt);

    const orderedDetails: Array<[string, string | undefined]> = [
      ["Title", issue?.title],
      ["Short ID", issue?.shortId],
      ["Status", issue?.status],
      ["Priority", issue?.priority],
      ["Assigned To", issue?.assignedTo?.name],
      ["Project", getProjectLabel(issue)],
    ];

    for (const [label, value] of orderedDetails) {
      if (Object.keys(details).length >= 6) {
        break;
      }

      addDetail(details, label, value);
    }

    return details;
  },
};

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

  if (configuration?.assignedTo) {
    metadata.push({ icon: "user", label: nodeMetadata?.assigneeLabel || configuration.assignedTo });
  }

  if (configuration?.priority) {
    metadata.push({ icon: "flag", label: configuration.priority });
  }

  if (configuration?.hasSeen != null) {
    metadata.push({ icon: "eye", label: `Seen: ${formatBoolean(configuration.hasSeen)}` });
  }

  if (configuration?.isPublic != null) {
    metadata.push({ icon: "globe", label: `Public: ${formatBoolean(configuration.isPublic)}` });
  }

  if (configuration?.isSubscribed != null) {
    metadata.push({ icon: "bell", label: `Subscribed: ${formatBoolean(configuration.isSubscribed)}` });
  }

  return metadata.slice(0, 3);
}

function formatBoolean(value: boolean | undefined): string | undefined {
  if (value === undefined) {
    return undefined;
  }

  return value ? "Yes" : "No";
}
