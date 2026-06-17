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
import type { GetPackageStatusConfiguration, PackageNodeMetadata, PackageStatusData } from "./types";

export const getPackageStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cloudsmith";

    return {
      iconSrc: cloudsmithIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: buildMetadata(context.node),
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
    const statusData = outputs?.default?.[0]?.data as PackageStatusData | undefined;
    if (!statusData) return details;

    if (statusData.stage_str) details["Stage"] = statusData.stage_str;
    if (statusData.status_str) details["Status"] = statusData.status_str;
    if (statusData.sync_progress != null) details["Sync Progress"] = `${statusData.sync_progress}%`;
    if (statusData.is_sync_completed != null) details["Sync Completed"] = statusData.is_sync_completed ? "Yes" : "No";
    if (statusData.is_quarantined != null) details["Quarantined"] = statusData.is_quarantined ? "Yes" : "No";

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function buildMetadata(node: NodeInfo): MetadataItem[] {
  const items: MetadataItem[] = [];
  const nodeMetadata = node.metadata as PackageNodeMetadata | undefined;
  const configuration = node.configuration as GetPackageStatusConfiguration | undefined;

  if (nodeMetadata?.repositoryName) {
    items.push({ icon: "package", label: nodeMetadata.repositoryName });
  } else if (configuration?.repository) {
    items.push({ icon: "package", label: configuration.repository });
  }

  if (nodeMetadata?.packageName) {
    items.push({ icon: "archive", label: nodeMetadata.packageName });
  } else if (configuration?.package) {
    items.push({ icon: "archive", label: configuration.package });
  }

  return items;
}

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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
