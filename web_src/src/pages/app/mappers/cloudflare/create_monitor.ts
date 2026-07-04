import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionInfo,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { baseMapper, firstOutputData } from "./base";
import { getCloudflarePoolName } from "./metadata";

interface CreateMonitorConfiguration {
  description?: string;
  type?: string;
  path?: string;
  port?: number;
  pool?: string;
  advanced?: Record<string, unknown>;
  /** Legacy flat fields saved before nested `advanced` was standard */
  interval?: number;
  timeout?: number;
  retries?: number;
  consecutiveUp?: number;
  consecutiveDown?: number;
}

interface CreateMonitorOutput {
  accountId?: string;
  monitorId?: string;
  poolId?: string;
  monitor?: {
    id?: string;
    type?: string;
    description?: string;
    path?: string;
    port?: number;
  };
  pool?: {
    id?: string;
    name?: string;
  };
}

export const createMonitorMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseMapper.props(context),
      metadata: metadataList(context.node, context.lastExecutions[0]),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details = baseMapper.getExecutionDetails(context) as Record<string, string>;
    const output = firstOutputData(context.execution.outputs) as CreateMonitorOutput | undefined;

    return output ? { ...details, ...outputDetails(output) } : details;
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },
};

function metadataList(node: NodeInfo, lastExecution?: ExecutionInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as CreateMonitorConfiguration | undefined;

  if (configuration?.description) {
    metadata.push({ icon: "activity", label: configuration.description });
  }

  if (configuration?.type) {
    metadata.push({ icon: "radio", label: configuration.type.toUpperCase() });
  }

  const target = monitorTarget(configuration);
  if (target) {
    metadata.push({ icon: "link", label: target });
  }

  const poolId = configuration?.pool?.trim();
  if (poolId) {
    const poolLabel = getCloudflarePoolName(node.metadata) || getPoolNameFromExecution(lastExecution) || poolId;
    metadata.push({ icon: "server", label: `Pool: ${poolLabel}` });
  }

  if (createMonitorShowsAdvancedBadge(configuration)) {
    metadata.push({ icon: "settings", label: "Advanced health check settings" });
  }

  return metadata;
}

function getPoolNameFromExecution(execution?: ExecutionInfo): string | undefined {
  const output = firstOutputData(execution?.outputs) as CreateMonitorOutput | undefined;
  const name = output?.pool?.name?.trim();
  return name || undefined;
}

function outputDetails(output: CreateMonitorOutput): Record<string, string> {
  return compactDetails({
    Name: output.monitor?.description || "-",
    Type: output.monitor?.type?.toUpperCase() || "-",
    Path: output.monitor?.path,
    Port: output.monitor?.port != null ? String(output.monitor.port) : undefined,
    Pool: output.pool?.name || output.poolId,
  });
}

function compactDetails(values: Record<string, string | undefined>): Record<string, string> {
  const details: Record<string, string> = {};

  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined) {
      details[key] = value;
    }
  }

  return details;
}

function monitorTarget(configuration?: CreateMonitorConfiguration): string {
  if (!configuration) {
    return "";
  }

  const path = configuration.path?.trim();
  const port = configuration.port;

  if (path && port != null) {
    return `${path} · Port ${port}`;
  }

  if (path) {
    return path;
  }

  if (port != null) {
    return `Port ${port}`;
  }

  return "";
}

function createMonitorShowsAdvancedBadge(configuration?: CreateMonitorConfiguration): boolean {
  if (advancedObjectHasSettings(configuration?.advanced)) {
    return true;
  }
  if (!configuration) {
    return false;
  }
  const legacyKeys: (keyof CreateMonitorConfiguration)[] = [
    "interval",
    "timeout",
    "retries",
    "consecutiveUp",
    "consecutiveDown",
  ];
  return legacyKeys.some(
    (key) => typeof configuration[key] === "number" && !Number.isNaN(configuration[key] as number),
  );
}

function advancedObjectHasSettings(adv: Record<string, unknown> | undefined): boolean {
  if (!adv || typeof adv !== "object") {
    return false;
  }

  return Object.values(adv).some(advancedValuePresent);
}

function advancedValuePresent(value: unknown): boolean {
  if (value === undefined || value === null || value === "") {
    return false;
  }

  if (Array.isArray(value)) {
    return value.length > 0;
  }

  return true;
}
