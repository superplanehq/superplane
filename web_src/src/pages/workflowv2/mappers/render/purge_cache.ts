import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "./common";
import { baseProps } from "./base";

interface PurgeCacheConfiguration {
  service?: string;
}

interface PurgeCacheOutput {
  serviceId?: string;
  status?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as PurgeCacheConfiguration | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${configuration.service}` });
  }

  return metadata;
}

export const purgeCacheMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as PurgeCacheOutput | undefined;

    return {
      "Requested At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Service ID": stringOrDash(result?.serviceId),
      Status: stringOrDash(result?.status),
    };
  },
};
