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

interface CreateStaticIPConfiguration {
  name?: string;
  region?: string;
}

interface CreateStaticIPOutputData {
  name?: string;
  address?: string;
  region?: string;
  status?: string;
  networkTier?: string;
}

interface DeleteStaticIPConfiguration {
  address?: string;
}

interface DeleteStaticIPOutputData {
  name?: string;
  region?: string;
}

interface ManageStaticIPConfiguration {
  action?: string;
  instance?: string;
}

interface ManageStaticIPOutputData {
  name?: string;
  zone?: string;
  externalIP?: string;
  action?: string;
}

function lastSegment(value: string | undefined): string | undefined {
  if (!value) return undefined;
  const trimmed = value.trim();
  if (!trimmed || trimmed.includes("{{")) return undefined;
  const idx = trimmed.lastIndexOf("/");
  return idx >= 0 ? trimmed.slice(idx + 1).replace(/[?#].*$/, "") : trimmed;
}

function subtitle(context: SubtitleContext): string | React.ReactNode {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
}

function baseProps(
  context: ComponentBaseContext,
  iconSlug: string,
  fallbackTitle: string,
  metadata: MetadataItem[],
): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name ?? "gcp";

  return {
    iconSrc: gcpIcon,
    iconSlug: context.componentDefinition?.icon ?? iconSlug,
    collapsedBackground: "bg-white",
    collapsed: context.node.isCollapsed,
    title: context.node.name || context.componentDefinition?.label || fallbackTitle,
    eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
    metadata,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

export const createStaticIPMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "globe", "Create Static IP", createStaticIPMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as CreateStaticIPOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Name"] = result.name;
    if (result.address) details["Address"] = result.address;
    if (result.region) details["Region"] = result.region;
    if (result.networkTier) details["Network Tier"] = result.networkTier;
    if (result.status) details["Status"] = result.status;
    return details;
  },

  subtitle,
};

export const deleteStaticIPMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "trash-2", "Delete Static IP", deleteStaticIPMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as DeleteStaticIPOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Name"] = result.name;
    if (result.region) details["Region"] = result.region;
    details["Status"] = "Released";
    return details;
  },

  subtitle,
};

export const manageStaticIPMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "globe", "Manage Static IP", manageStaticIPMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as ManageStaticIPOutputData | undefined;
    if (!result) return details;

    if (result.action) details["Action"] = result.action;
    if (result.name) details["Instance"] = result.name;
    if (result.zone) details["Zone"] = result.zone;
    if (result.externalIP) details["External IP"] = result.externalIP;
    return details;
  },

  subtitle,
};

function createStaticIPMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as CreateStaticIPConfiguration | undefined;
  if (config?.name) metadata.push({ icon: "globe", label: config.name });
  if (config?.region) metadata.push({ icon: "map-pin", label: lastSegment(config.region) || config.region });
  return metadata;
}

function deleteStaticIPMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as DeleteStaticIPConfiguration | undefined;
  const name = lastSegment(config?.address);
  if (name) metadata.push({ icon: "trash-2", label: name });
  const region = regionFromSelfLink(config?.address);
  if (region) metadata.push({ icon: "map-pin", label: region });
  return metadata;
}

// regionFromSelfLink extracts the region from an address selfLink/path such as
// ".../regions/us-central1/addresses/web-ip".
function regionFromSelfLink(value: string | undefined): string | undefined {
  if (!value || value.includes("{{")) return undefined;
  const match = value.match(/regions\/([^/]+)\//);
  return match ? match[1] : undefined;
}

function manageStaticIPMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as ManageStaticIPConfiguration | undefined;
  if (config?.action) metadata.push({ icon: "globe", label: config.action });
  const instance = lastSegment(config?.instance);
  if (instance) metadata.push({ icon: "server", label: instance });
  return metadata;
}
