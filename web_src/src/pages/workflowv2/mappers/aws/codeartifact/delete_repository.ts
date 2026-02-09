import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsCodeArtifactIcon from "@/assets/icons/integrations/aws.codeartifact.svg";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";

interface DeleteRepositoryConfiguration {
  domain?: string;
  repository?: string;
}

interface RepositoryPayload {
  repository?: {
    arn?: string;
    name?: string;
    domainName?: string;
    domainOwner?: string;
    description?: string;
    administratorAccount?: string;
    createdTime?: number;
  };
}

export const deleteRepositoryMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: awsCodeArtifactIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? deleteRepositoryEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: deleteRepositoryMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as RepositoryPayload | undefined;
    const repo = data?.repository;

    if (!repo) {
      return {};
    }

    return {
      Repository: stringOrDash(repo.name),
      Domain: stringOrDash(repo.domainName),
      ARN: stringOrDash(repo.arn),
      Description: stringOrDash(repo.description),
      "Deleted At": context.execution.createdAt ? formatTimestampInUserTimezone(context.execution.createdAt) : "-",
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function deleteRepositoryMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as DeleteRepositoryConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.domain) {
    items.push({ icon: "database", label: config.domain });
  }
  if (config?.repository) {
    items.push({ icon: "boxes", label: config.repository });
  }

  return items;
}

function deleteRepositoryEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}
