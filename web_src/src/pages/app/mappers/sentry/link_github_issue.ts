import type { ComponentBaseProps } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/lib/colors";
import { formatTimeAgo } from "@/lib/date";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { addDetail, addFormattedTimestamp, buildEventSections } from "./utils";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

interface LinkGitHubIssueConfiguration {
  issueId?: string;
  githubIntegrationId?: string;
  repo?: string;
  externalIssue?: string;
  comment?: string;
}

interface LinkGitHubIssueNodeMetadata {
  issueTitle?: string;
  githubIntegrationLabel?: string;
  externalIssueLabel?: string;
}

interface ExternalIssueLink {
  id?: number | string;
  key?: string;
  url?: string;
  integrationId?: number | string;
  displayName?: string;
}

export const linkGitHubIssueMapper: ComponentBaseMapper = {
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
    const link = outputs?.default?.[0]?.data as ExternalIssueLink | undefined;
    const timestamp = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt));
    return [link?.displayName || link?.key, timestamp].filter(Boolean).join(" · ");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const link = outputs?.default?.[0]?.data as ExternalIssueLink | undefined;
    const details: Record<string, string> = {};

    addFormattedTimestamp(details, "Triggered At", context.execution.createdAt);

    const orderedDetails: Array<[string, string | undefined]> = [
      ["Display Name", link?.displayName],
      ["Issue Key", link?.key],
      ["URL", link?.url],
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
  const configuration = node.configuration as LinkGitHubIssueConfiguration | undefined;
  const nodeMetadata = node.metadata as LinkGitHubIssueNodeMetadata | undefined;
  const metadata = [];

  const issueLabel = nodeMetadata?.issueTitle || configuration?.issueId;
  if (issueLabel) {
    metadata.push({ icon: "bug", label: issueLabel });
  }

  const externalIssueLabel =
    nodeMetadata?.externalIssueLabel ||
    (configuration?.repo && configuration?.externalIssue
      ? `${configuration.repo}#${configuration.externalIssue}`
      : configuration?.externalIssue);
  if (externalIssueLabel) {
    metadata.push({ icon: "link", label: externalIssueLabel });
  }

  const integrationLabel = nodeMetadata?.githubIntegrationLabel || configuration?.githubIntegrationId;
  if (integrationLabel) {
    metadata.push({ icon: "github", label: integrationLabel });
  }

  return metadata.slice(0, 3);
}
