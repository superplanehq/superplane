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
  compartment?: string;
  application?: string;
  function?: string;
  payload?: string;
}

interface InvokeFunctionNodeMetadata {
  applicationId?: string;
  applicationName?: string;
  functionId?: string;
  functionName?: string;
}

interface InvokeFunctionOutputData {
  functionId?: string;
  statusCode?: number;
  response?: string;
}

type InvokeFunctionOutputPayload = OutputPayload & {
  data?: InvokeFunctionOutputData;
};

interface InvokeFunctionExecutionMetadata {
  startedAt?: string;
}

function getExecutedAt(context: ExecutionDetailsContext): string | undefined {
  const metadata = context.execution.metadata as InvokeFunctionExecutionMetadata | undefined;
  const ts = metadata?.startedAt ?? context.execution.createdAt;
  return ts ? new Date(ts).toLocaleString() : undefined;
}

function getOutputData(context: ExecutionDetailsContext): InvokeFunctionOutputData | undefined {
  const outputs = context.execution.outputs as { default?: InvokeFunctionOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  return payload?.data;
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

    const executedAt = getExecutedAt(context);
    if (executedAt) {
      details["Executed At"] = executedAt;
    }

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
  const nodeMeta = node.metadata as InvokeFunctionNodeMetadata | undefined;
  const items: MetadataItem[] = [];

  const appLabel = nodeMeta?.applicationName ?? config?.application;
  if (appLabel) {
    items.push({ icon: "layout-grid", label: appLabel });
  }

  const fnLabel = nodeMeta?.functionName ?? config?.function;
  if (fnLabel) {
    items.push({ icon: "zap", label: fnLabel });
  }

  return items;
}
