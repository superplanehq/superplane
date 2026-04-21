import type { ComponentBaseProps, EventSection } from "@/pages/workflowv2/mappers/types";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getState, getStateMap } from "..";
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
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { SnapshotNodeMetadata, DeleteSnapshotConfiguration } from "./types";

export const deleteSnapshotMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "digitalocean";

    return {
      iconSrc: doIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Record<string, any> | undefined;
    if (!result) return details;

    details["Snapshot ID"] = result.snapshotId?.toString() || "-";
    details["Status"] = result.deleted ? "Deleted" : "-";

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as SnapshotNodeMetadata | undefined;
  const configuration = node.configuration as DeleteSnapshotConfiguration;

  if (nodeMetadata?.snapshotName) {
    metadata.push({ icon: "trash", label: nodeMetadata.snapshotName });
  } else if (configuration?.snapshot) {
    metadata.push({ icon: "trash", label: `Snapshot ID: ${configuration.snapshot}` });
  }

  return metadata;
}

function baseEventSections(execution: ExecutionInfo, componentName: string): EventSection[] {
  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
