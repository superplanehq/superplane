import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import cloudsmithIcon from "@/assets/icons/integrations/cloudsmith.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { GetRepositoryConfiguration, RepositoryData, RepositoryNodeMetadata } from "./types";

export const getRepositoryMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cloudsmith";

    return {
      iconSrc: cloudsmithIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, unknown> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const repository = outputs?.default?.[0]?.data as RepositoryData | undefined;
    if (!repository) return details;

    addRepositoryDetails(details, repository);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function addRepositoryDetails(details: Record<string, string>, repository: RepositoryData): void {
  details["Name"] = repository.name || "-";
  details["Namespace"] = repository.namespace || "-";
  details["Size"] = repository.size_str || (repository.size != null ? `${repository.size} bytes` : "-");
  details["Packages"] = repository.package_count != null ? String(repository.package_count) : "-";
  details["Downloads"] = repository.num_downloads != null ? String(repository.num_downloads) : "-";

  if (repository.num_quarantined_packages) {
    details["Quarantined Packages"] = String(repository.num_quarantined_packages);
  }

  if (repository.num_policy_violated_packages) {
    details["Policy Violations"] = String(repository.num_policy_violated_packages);
  }
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as RepositoryNodeMetadata | undefined;
  const configuration = node.configuration as GetRepositoryConfiguration | undefined;

  if (nodeMetadata?.repositoryName) {
    metadata.push({ icon: "package", label: nodeMetadata.repositoryName });
  } else if (configuration?.repository) {
    metadata.push({ icon: "package", label: configuration.repository });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.rootEvent || !execution.createdAt || !execution.rootEvent.id) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}
