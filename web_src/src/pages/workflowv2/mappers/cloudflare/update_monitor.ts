import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { baseMapper, firstOutputData } from "./base";
import { getCloudflareMonitorDisplayLabel } from "./metadata";

interface UpdateMonitorConfiguration {
  monitor?: string;
  description?: string;
  type?: string;
  path?: string;
  port?: number;
  advanced?: Record<string, unknown>;
}

interface UpdateMonitorOutput {
  accountId?: string;
  monitorId?: string;
  monitor?: {
    id?: string;
    type?: string;
    description?: string;
    path?: string;
    port?: number;
    interval?: number;
    timeout?: number;
    retries?: number;
  };
}

export const updateMonitorMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseMapper.props(context),
      metadata: metadataList(context.node),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details = baseMapper.getExecutionDetails(context) as Record<string, string>;
    const output = firstOutputData(context.execution.outputs) as UpdateMonitorOutput | undefined;

    if (!output?.monitor) {
      return details;
    }

    const m = output.monitor;
    if (m.description) details["Name"] = m.description;
    if (m.type) details["Type"] = m.type.toUpperCase();
    if (m.path) details["Path"] = m.path;
    if (m.port != null) details["Port"] = String(m.port);
    if (m.interval != null) details["Interval"] = `${m.interval}s`;
    if (m.timeout != null) details["Timeout"] = `${m.timeout}s`;

    return details;
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as UpdateMonitorConfiguration | undefined;
  const metadata: MetadataItem[] = [];

  const monitorId = configuration?.monitor?.trim();
  if (monitorId) {
    metadata.push({
      icon: "activity",
      label: getCloudflareMonitorDisplayLabel(node.metadata, monitorId),
    });
  }

  if (configuration?.description) {
    metadata.push({ icon: "pencil", label: configuration.description });
  }

  if (configuration?.type) {
    metadata.push({ icon: "radio", label: configuration.type.toUpperCase() });
  }

  return metadata;
}
