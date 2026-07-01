import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface DeleteApplicationOutputData {
  applicationId?: string;
  displayName?: string;
  deleted?: boolean;
}

type DeleteApplicationOutputPayload = OutputPayload & {
  data?: DeleteApplicationOutputData;
};

interface DeleteApplicationExecutionMetadata {
  startedAt?: string;
}

function getExecutedAt(context: ExecutionDetailsContext): string | undefined {
  const metadata = context.execution.metadata as DeleteApplicationExecutionMetadata | undefined;
  const ts = metadata?.startedAt ?? context.execution.createdAt;
  return ts ? new Date(ts).toLocaleString() : undefined;
}

function getOutputData(context: ExecutionDetailsContext): DeleteApplicationOutputData | undefined {
  const outputs = context.execution.outputs as { default?: DeleteApplicationOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  return payload?.data;
}

export const deleteApplicationMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return {
      ...baseMapper.props(context),
      metadata: deleteApplicationMetadataList(context),
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

    if (data.displayName) {
      details["Display Name"] = data.displayName;
    }
    if (data.deleted != null) {
      details["Deleted"] = String(data.deleted);
    }

    return details;
  },
};

interface DeleteApplicationNodeMetadata {
  applicationId?: string;
  applicationName?: string;
}

function deleteApplicationMetadataList(context: ComponentBaseContext): MetadataItem[] {
  const items: MetadataItem[] = [];
  const nodeMetadata = context.node.metadata as DeleteApplicationNodeMetadata | undefined;
  const configuration = context.node.configuration as { application?: string } | undefined;

  if (nodeMetadata?.applicationName) {
    items.push({ icon: "trash-2", label: nodeMetadata.applicationName });
  } else if (configuration?.application) {
    items.push({ icon: "trash-2", label: configuration.application });
  }

  return items;
}
