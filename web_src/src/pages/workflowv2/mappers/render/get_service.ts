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
import { formatTimestamp, stringOrDash } from "./common";
import { baseProps } from "./base";

interface GetServiceConfiguration {
  service?: string;
}

interface GetServiceOutput {
  serviceId?: string;
  serviceName?: string;
  type?: string;
  suspended?: string;
  dashboardUrl?: string;
  createdAt?: string;
  updatedAt?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetServiceConfiguration | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${configuration.service}` });
  }

  return metadata;
}

export const getServiceMapper: ComponentBaseMapper = {
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
    const result = outputs?.default?.[0]?.data as GetServiceOutput | undefined;

    return {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Service ID": stringOrDash(result?.serviceId),
      "Service Name": stringOrDash(result?.serviceName),
      Type: stringOrDash(result?.type),
      Suspended: result?.suspended === undefined ? "-" : result.suspended === "suspended" ? "Yes" : "No",
      "Dashboard URL": stringOrDash(result?.dashboardUrl),
      "Created At": formatTimestamp(result?.createdAt),
      "Updated At": formatTimestamp(result?.updatedAt),
    };
  },
};
