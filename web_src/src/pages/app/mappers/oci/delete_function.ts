import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface DeleteFunctionNodeMetadata {
  applicationId?: string;
  applicationName?: string;
  functionId?: string;
  functionName?: string;
}

interface DeleteFunctionOutputData {
  functionId?: string;
  deleted?: boolean;
}

type DeleteFunctionOutputPayload = OutputPayload & {
  data?: DeleteFunctionOutputData;
};

interface DeleteFunctionExecutionMetadata {
  startedAt?: string;
}

function getExecutedAt(context: ExecutionDetailsContext): string | undefined {
  const metadata = context.execution.metadata as DeleteFunctionExecutionMetadata | undefined;
  const ts = metadata?.startedAt ?? context.execution.createdAt;
  return ts ? new Date(ts).toLocaleString() : undefined;
}

function getOutputData(context: ExecutionDetailsContext): DeleteFunctionOutputData | undefined {
  const outputs = context.execution.outputs as { default?: DeleteFunctionOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  return payload?.data;
}

export const deleteFunctionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return {
      ...baseMapper.props(context),
      metadata: deleteFunctionMetadataList(context.node),
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

    if (data.functionId) {
      details["Function ID"] = data.functionId;
    }
    if (data.deleted != null) {
      details["Deleted"] = String(data.deleted);
    }

    return details;
  },
};

function deleteFunctionMetadataList(node: ComponentBaseContext["node"]): MetadataItem[] {
  const nodeMeta = node.metadata as DeleteFunctionNodeMetadata | undefined;
  const items: MetadataItem[] = [];

  const appLabel = nodeMeta?.applicationName ?? nodeMeta?.applicationId;
  if (appLabel) {
    items.push({ icon: "layout-grid", label: appLabel });
  }

  const fnLabel = nodeMeta?.functionName ?? nodeMeta?.functionId;
  if (fnLabel) {
    items.push({ icon: "zap", label: fnLabel });
  }

  return items;
}
