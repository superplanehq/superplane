import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { noopMapper } from "../noop";
import type React from "react";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";

type HetznerConfiguration = {
  serverType?: unknown;
  image?: unknown;
  snapshot?: unknown;
  description?: unknown;
  location?: unknown;
  firewall?: unknown;
  server?: unknown;
  loadBalancer?: unknown;
  loadBalancerType?: unknown;
  algorithm?: unknown;
  sshKeys?: unknown;
};

const metadataFields: Array<{ key: keyof HetznerConfiguration; icon: MetadataItem["icon"]; label: string }> = [
  { key: "serverType", icon: "cpu", label: "Type" },
  { key: "image", icon: "hard-drive", label: "Image" },
  { key: "description", icon: "camera", label: "Snapshot" },
  { key: "snapshot", icon: "hard-drive", label: "Snapshot image" },
  { key: "location", icon: "map-pin", label: "Location" },
  { key: "firewall", icon: "shield", label: "Firewall" },
  { key: "server", icon: "server", label: "Server" },
  { key: "loadBalancer", icon: "route", label: "Load Balancer" },
  { key: "loadBalancerType", icon: "cpu", label: "LB Type" },
  { key: "algorithm", icon: "shuffle", label: "Algorithm" },
];

function getConfigValue(value: unknown): string | undefined {
  if (typeof value === "string") {
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  }

  if (typeof value === "number") {
    return String(value);
  }

  if (!value || typeof value !== "object") {
    return undefined;
  }

  const objectValue = value as Record<string, unknown>;
  const fields = ["label", "displayName", "name", "slug", "value", "id"];

  for (const field of fields) {
    const candidate = objectValue[field];
    if (typeof candidate === "string") {
      const trimmed = candidate.trim();
      if (trimmed.length > 0) {
        return trimmed;
      }
    }
    if (typeof candidate === "number") {
      return String(candidate);
    }
  }

  return undefined;
}

function formatImageName(image: string, execution?: ExecutionInfo): string {
  const output = execution?.outputs as { default?: Array<{ data?: Record<string, unknown> }> } | undefined;
  const outputImageName = getConfigValue(output?.default?.[0]?.data?.imageName);
  return outputImageName && outputImageName !== image ? `${outputImageName} (${image})` : outputImageName || image;
}

function metadataList(node: NodeInfo, execution?: ExecutionInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = (node.configuration as HetznerConfiguration | undefined) ?? {};

  for (const { key, icon, label } of metadataFields) {
    const value = getConfigValue(configuration[key]);
    if (value === undefined) continue;

    const displayValue = key === "image" ? formatImageName(value, execution) : value;
    metadata.push({ icon, label: `${label}: ${displayValue}` });
  }

  const sshKeys = Array.isArray(configuration.sshKeys) ? configuration.sshKeys : [];
  if (sshKeys.length > 0) {
    metadata.push({ icon: "key", label: `SSH keys: ${sshKeys.length}` });
  }

  return metadata;
}

function getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};
  const metadata = context.execution.metadata as Record<string, unknown> | undefined;
  const outputs = context.execution.outputs as { default?: Array<{ data?: Record<string, unknown> }> } | undefined;
  const output = outputs?.default?.[0]?.data;

  const isServerComponent = context.node.componentName.includes("Server");
  const isLoadBalancerComponent = context.node.componentName.includes("LoadBalancer");
  const isSnapshotComponent = context.node.componentName.includes("Snapshot");

  const serverId =
    (isServerComponent ? (output?.serverId ?? output?.id) : undefined) ??
    metadata?.serverId ??
    (metadata?.server as Record<string, unknown> | undefined)?.id;
  const loadBalancerId =
    (isLoadBalancerComponent ? (output?.loadBalancerId ?? output?.id) : undefined) ??
    metadata?.loadBalancerId ??
    (metadata?.loadBalancer as Record<string, unknown> | undefined)?.id;
  const imageId =
    (isSnapshotComponent ? (output?.imageId ?? output?.id) : undefined) ??
    metadata?.imageId ??
    (metadata?.image as Record<string, unknown> | undefined)?.id;
  const imageName = output?.imageName;

  if (serverId !== undefined) {
    details["Server ID"] = String(serverId);
  }

  if (loadBalancerId !== undefined) {
    details["Load Balancer ID"] = String(loadBalancerId);
  }
  if (imageId !== undefined) {
    details["Image ID"] = String(imageId);
  }
  if (imageName !== undefined) {
    details["Image"] = String(imageName);
  }

  if (context.execution.createdAt) {
    details["Started at"] = new Date(context.execution.createdAt).toLocaleString();
  }
  if (context.execution.updatedAt && context.execution.state === "STATE_FINISHED") {
    details["Finished at"] = new Date(context.execution.updatedAt).toLocaleString();
  }

  return details;
}

function props(context: ComponentBaseContext) {
  const base = noopMapper.props(context);
  const latestExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : undefined;
  return {
    ...base,
    metadata: metadataList(context.node, latestExecution),
  };
}

function subtitle(context: SubtitleContext): string | React.ReactNode {
  if (!context.execution.createdAt) return "";
  return renderTimeAgo(new Date(context.execution.createdAt));
}

export const hetznerBaseMapper: ComponentBaseMapper = {
  ...noopMapper,
  props: props,
  getExecutionDetails: getExecutionDetails,
  subtitle: subtitle,
};
