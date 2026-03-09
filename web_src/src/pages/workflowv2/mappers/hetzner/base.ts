import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { noopMapper } from "../noop";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";

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

function metadataList(node: NodeInfo, execution?: ExecutionInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = (node.configuration as HetznerConfiguration | undefined) ?? {};
  const output = execution?.outputs as { default?: Array<{ data?: Record<string, unknown> }> } | undefined;
  const outputData = output?.default?.[0]?.data;

  const serverType = getConfigValue(configuration.serverType);
  const image = getConfigValue(configuration.image);
  const outputImageName = getConfigValue(outputData?.imageName);
  const snapshot = getConfigValue(configuration.snapshot);
  const description = getConfigValue(configuration.description);
  const location = getConfigValue(configuration.location);
  const firewall = getConfigValue(configuration.firewall);
  const server = getConfigValue(configuration.server);
  const loadBalancer = getConfigValue(configuration.loadBalancer);
  const loadBalancerType = getConfigValue(configuration.loadBalancerType);
  const algorithm = getConfigValue(configuration.algorithm);
  const sshKeys = Array.isArray(configuration.sshKeys) ? configuration.sshKeys : [];

  if (serverType) {
    metadata.push({ icon: "cpu", label: `Type: ${serverType}` });
  }
  if (image) {
    const imageLabel =
      outputImageName && outputImageName !== image ? `${outputImageName} (${image})` : outputImageName || image;
    metadata.push({ icon: "hard-drive", label: `Image: ${imageLabel}` });
  }
  if (description) {
    metadata.push({ icon: "camera", label: `Snapshot: ${description}` });
  }
  if (snapshot) {
    metadata.push({ icon: "hard-drive", label: `Snapshot image: ${snapshot}` });
  }
  if (location) {
    metadata.push({ icon: "map-pin", label: `Location: ${location}` });
  }
  if (firewall) {
    metadata.push({ icon: "shield", label: `Firewall: ${firewall}` });
  }
  if (server) {
    metadata.push({ icon: "server", label: `Server: ${server}` });
  }
  if (loadBalancer) {
    metadata.push({ icon: "route", label: `Load Balancer: ${loadBalancer}` });
  }
  if (loadBalancerType) {
    metadata.push({ icon: "cpu", label: `LB Type: ${loadBalancerType}` });
  }
  if (algorithm) {
    metadata.push({ icon: "shuffle", label: `Algorithm: ${algorithm}` });
  }
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

  if (context.execution.resultMessage) {
    details["Error"] = context.execution.resultMessage;
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

function subtitle(context: SubtitleContext): string {
  if (!context.execution.createdAt) return "";
  return formatTimeAgo(new Date(context.execution.createdAt));
}

export const hetznerBaseMapper: ComponentBaseMapper = {
  ...noopMapper,
  props: props,
  getExecutionDetails: getExecutionDetails,
  subtitle: subtitle,
};
