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
import gcpComputeIcon from "@/assets/icons/integrations/gcp.compute.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections, parseInstancePath } from "./event_helpers";

interface VMInstanceNodeMetadata {
  instanceName?: string;
  zone?: string;
}

interface GetVMInstanceConfiguration {
  instance?: string;
}

interface GetVMInstanceOutputData {
  instanceId?: string;
  selfLink?: string;
  status?: string;
  zone?: string;
  name?: string;
  machineType?: string;
  internalIP?: string;
  externalIP?: string;
}

export const getVMInstanceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "gcp";

    return {
      iconSrc: gcpComputeIcon,
      iconSlug: context.componentDefinition?.icon ?? "search",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Get VM Instance",
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
    const result = outputs?.default?.[0]?.data as GetVMInstanceOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Instance Name"] = result.name;
    if (result.zone) details["Zone"] = result.zone;
    if (result.status) details["Status"] = result.status;
    if (result.machineType) details["Machine Type"] = result.machineType;
    if (result.internalIP) details["Internal IP"] = result.internalIP;
    if (result.externalIP) details["External IP"] = result.externalIP;
    if (result.selfLink) details["Self Link"] = result.selfLink;

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
  const configuration = node.configuration as GetVMInstanceConfiguration | undefined;

  const parsed = parseInstancePath(configuration?.instance);
  const instanceName = nodeMetadata?.instanceName || parsed?.name || configuration?.instance;
  const zone = nodeMetadata?.zone || parsed?.zone;

  if (instanceName) {
    metadata.push({ icon: "search", label: instanceName });
  }
  if (zone) {
    metadata.push({ icon: "map-pin", label: zone });
  }

  return metadata;
}
