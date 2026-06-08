import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import gcpIcon from "@/assets/icons/integrations/gcp.compute.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./event_helpers";

interface VMInstanceNodeMetadata {
  instanceName?: string;
  zone?: string;
}

interface UpdateVMInstanceTypeConfiguration {
  instance?: string;
  machineType?: string;
}

interface UpdateVMInstanceTypeOutputData {
  name?: string;
  zone?: string;
  status?: string;
  machineType?: string;
}

export const updateVMInstanceTypeMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "gcp";

    return {
      iconSrc: gcpIcon,
      iconSlug: context.componentDefinition?.icon ?? "cpu",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Update VM Machine Type",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as UpdateVMInstanceTypeOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Instance Name"] = result.name;
    if (result.zone) details["Zone"] = result.zone;
    if (result.machineType) details["Machine Type"] = result.machineType;
    if (result.status) details["Status"] = result.status;

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as VMInstanceNodeMetadata | undefined;
  const configuration = node.configuration as UpdateVMInstanceTypeConfiguration | undefined;

  const instanceName = nodeMetadata?.instanceName || configuration?.instance;
  if (instanceName) {
    metadata.push({ icon: "server", label: instanceName });
  }
  if (nodeMetadata?.zone) {
    metadata.push({ icon: "map-pin", label: nodeMetadata.zone });
  }
  if (configuration?.machineType) {
    metadata.push({ icon: "cpu", label: configuration.machineType });
  }

  return metadata;
}
