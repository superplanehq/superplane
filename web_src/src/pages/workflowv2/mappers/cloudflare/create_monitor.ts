import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { baseMapper, firstOutputData } from "./base";

interface CreateMonitorConfiguration {
  description?: string;
  type?: string;
  path?: string;
  port?: number;
  pool?: string;
  advanced?: Record<string, unknown>;

  // Legacy flat fields.
  method?: string;
  expectedCodes?: string;
  headers?: unknown[];
  expectedBody?: string;
  followRedirects?: boolean;
  allowInsecure?: boolean;
  probeZone?: string;
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
    expected_codes?: string;
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
      metadata: metadataList(context.node.configuration as CreateMonitorConfiguration | undefined),
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

function metadataList(configuration?: CreateMonitorConfiguration): MetadataItem[] {
  const metadata: MetadataItem[] = [];

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

  if (configuration?.pool) {
    metadata.push({ icon: "server", label: `Pool: ${configuration.pool}` });
  }

  if (hasAdvancedSettings(configuration)) {
    metadata.push({ icon: "settings", label: "Advanced health check settings" });
  }

  return metadata;
}

function outputDetails(output: CreateMonitorOutput): Record<string, string> {
  return compactDetails({
    "Monitor ID": output.monitorId || output.monitor?.id || "-",
    Name: output.monitor?.description || "-",
    Type: output.monitor?.type?.toUpperCase() || "-",
    Path: output.monitor?.path,
    Port: output.monitor?.port != null ? String(output.monitor.port) : undefined,
    "Expected Codes": output.monitor?.expected_codes,
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

function hasAdvancedSettings(configuration?: CreateMonitorConfiguration): boolean {
  if (!configuration) {
    return false;
  }

  if (configuration.advanced && Object.keys(configuration.advanced).length > 0) {
    return true;
  }

  const adv = configuration.advanced;
  return Boolean(
    configuration.method ||
      configuration.expectedCodes ||
      configuration.expectedBody ||
      configuration.headers?.length ||
      configuration.followRedirects != null ||
      configuration.allowInsecure != null ||
      configuration.probeZone ||
      configuration.consecutiveUp != null ||
      configuration.consecutiveDown != null ||
      typeof adv?.interval === "number" ||
      typeof adv?.timeout === "number" ||
      typeof adv?.retries === "number",
  );
}
