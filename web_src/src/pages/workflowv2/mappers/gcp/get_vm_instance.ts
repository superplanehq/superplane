import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
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
import gcpIcon from "@/assets/icons/integrations/gcp.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

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

function parseInstancePath(value: string | undefined): { zone: string; name: string } | null {
  if (!value) return null;
  const trimmed = value.trim();
  if (!trimmed || trimmed.includes("{{")) return null;
  const match = trimmed.match(/zones\/([^/]+)\/instances\/([^/?#]+)/);
  if (!match) return null;
  return { zone: match[1], name: match[2] };
}

export const getVMInstanceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "gcp";

    return {
      iconSrc: gcpIcon,
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

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.nodeId) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  if (!rootTriggerNode?.componentName) {
    return [];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title, subtitle } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const fallbackSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";
  const eventSubtitle = subtitle || fallbackSubtitle;

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id!,
    },
  ];
}
