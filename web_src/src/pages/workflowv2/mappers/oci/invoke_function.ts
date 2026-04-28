import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface InvokeFunctionConfiguration {
  compartmentId?: string;
  applicationId?: string;
  functionId?: string;
  payload?: string;
}

interface InvokeFunctionOutputData {
  functionId?: string;
  statusCode?: number;
  response?: string;
}

type InvokeFunctionOutputPayload = OutputPayload & {
  data?: InvokeFunctionOutputData;
};

function getOutputData(context: ExecutionDetailsContext): InvokeFunctionOutputData | undefined {
  const outputs = context.execution.outputs as { default?: InvokeFunctionOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  if (!payload) return undefined;
  return (payload.data ?? payload) as InvokeFunctionOutputData;
}

export const invokeFunctionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return {
      ...baseMapper.props(context),
      metadata: invokeFunctionMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const data = getOutputData(context);
    if (!data) return details;

    if (data.statusCode != null) {
      details["Status Code"] = String(data.statusCode);
    }
    if (data.functionId) {
      details["Function ID"] = data.functionId;
    }
    if (data.response) {
      details["Response"] = data.response;
    }

    return details;
  },
};

function invokeFunctionMetadataList(node: ComponentBaseContext["node"]): MetadataItem[] {
  const config = node.configuration as InvokeFunctionConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.functionId) {
    items.push({ icon: "zap", label: config.functionId });
  }

  return items;
}
