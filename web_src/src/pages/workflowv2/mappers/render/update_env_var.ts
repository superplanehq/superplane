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

interface UpdateEnvVarConfiguration {
  service?: string;
  key?: string;
  valueStrategy?: string;
  emitValue?: boolean;
}

interface UpdateEnvVarOutput {
  serviceId?: string;
  key?: string;
  valueGenerated?: boolean;
  value?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpdateEnvVarConfiguration | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${configuration.service}` });
  }
  if (configuration?.key) {
    metadata.push({ icon: "key-round", label: configuration.key });
  }

  if (configuration?.valueStrategy) {
    metadata.push({ icon: "sliders-horizontal", label: `Strategy: ${configuration.valueStrategy}` });
  }

  if (configuration?.emitValue) {
    metadata.push({ icon: "shield-alert", label: "Emits value" });
  }

  return metadata;
}

export const updateEnvVarMapper: ComponentBaseMapper = {
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
    const result = outputs?.default?.[0]?.data as UpdateEnvVarOutput | undefined;

    const details: Record<string, string> = {
      "Updated At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Service ID": stringOrDash(result?.serviceId),
      Key: stringOrDash(result?.key),
      "Value Generated": result?.valueGenerated === undefined ? "-" : result.valueGenerated ? "Yes" : "No",
    };

    if (result?.value !== undefined) {
      details.Value = result.value;
    }

    return details;
  },
};
