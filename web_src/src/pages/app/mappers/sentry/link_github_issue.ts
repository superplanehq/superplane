import type { ComponentBaseProps } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/lib/colors";
import { formatTimeAgo } from "@/lib/date";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getState, getStateMap, getTriggerRenderer } from "../mapperLookup";
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
  const metadata: Array<{ icon: string; label: string }> = [];

  pushMetadataItem(metadata, "bug", nodeMetadata?.issueTitle || configuration?.issueId);
  pushMetadataItem(metadata, "link", githubIssueLinkLabel(nodeMetadata, configuration));
  pushMetadataItem(metadata, "github", nodeMetadata?.githubIntegrationLabel || configuration?.githubIntegrationId);

  return metadata.slice(0, 3);
}

function pushMetadataItem(metadata: Array<{ icon: string; label: string }>, icon: string, label?: string) {
  if (!label) {
    return;
  }

  metadata.push({ icon, label });
}

function githubIssueLinkLabel(
  nodeMetadata: LinkGitHubIssueNodeMetadata | undefined,
  configuration: LinkGitHubIssueConfiguration | undefined,
): string | undefined {
  if (nodeMetadata?.externalIssueLabel) {
    return nodeMetadata.externalIssueLabel;
  }

  if (configuration?.repo && configuration?.externalIssue) {
    return `${configuration.repo}#${configuration.externalIssue}`;
  }

  return configuration?.externalIssue;
}
