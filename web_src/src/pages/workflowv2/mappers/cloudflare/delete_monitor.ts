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

interface DeleteMonitorConfiguration {
  monitor?: string;
  force?: boolean;
}

interface DeleteMonitorOutput {
  accountId?: string;
  monitorId?: string;
  deleted?: boolean;
  references?: unknown[];
}

export const deleteMonitorMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseMapper.props(context),
      metadata: metadataList(context.node),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details = baseMapper.getExecutionDetails(context) as Record<string, string>;
    const output = firstOutputData(context.execution.outputs) as DeleteMonitorOutput | undefined;

    if (!output) {
      return details;
    }

    details["Monitor"] = output.monitorId
      ? getCloudflareMonitorDisplayLabel(context.node.metadata, output.monitorId)
      : "-";
    details["Deleted"] = output.deleted ? "Yes" : "No";

    if (output.references) {
      details["References"] = String(output.references.length);
    }

    return details;
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as DeleteMonitorConfiguration | undefined;
  const metadata: MetadataItem[] = [];

  const monitorId = configuration?.monitor?.trim();
  if (monitorId) {
    metadata.push({
      icon: "trash-2",
      label: getCloudflareMonitorDisplayLabel(node.metadata, monitorId),
    });
  }

  if (configuration?.force) {
    metadata.push({ icon: "shield-alert", label: "Force delete" });
  }

  return metadata;
}
