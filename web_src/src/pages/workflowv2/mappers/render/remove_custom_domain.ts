import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { serviceMetadataLabel, stringOrDash, type RenderServiceNodeMetadata } from "./common";
import { baseProps } from "./base";

interface RemoveCustomDomainConfiguration {
  service?: string;
  domain?: string;
}

interface RemoveCustomDomainOutput {
  name?: string;
  serviceId?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as RemoveCustomDomainConfiguration | undefined;
  const nodeMetadata = node.metadata as RenderServiceNodeMetadata | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${serviceMetadataLabel(nodeMetadata, configuration.service)}` });
  }
  if (configuration?.domain) {
    metadata.push({ icon: "globe", label: configuration.domain });
  }

  return metadata;
}

export const removeCustomDomainMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as RemoveCustomDomainOutput | undefined;

    return {
      "Removed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Domain Name": stringOrDash(result?.name),
      "Service ID": stringOrDash(result?.serviceId),
    };
  },
};
