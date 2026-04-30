import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface CreateFunctionConfiguration {
  compartment?: string;
  application?: string;
  displayName?: string;
  imageRepository?: string;
  image?: string;
  memoryInMBs?: number;
  timeoutInSeconds?: number;
}

interface CreateFunctionNodeMetadata {
  applicationId?: string;
  applicationName?: string;
}

interface CreateFunctionOutputData {
  functionId?: string;
  displayName?: string;
  applicationId?: string;
  image?: string;
  memoryInMBs?: number;
  invokeEndpoint?: string;
  lifecycleState?: string;
  timeCreated?: string;
}

type CreateFunctionOutputPayload = OutputPayload & {
  data?: CreateFunctionOutputData;
};

interface CreateFunctionExecutionMetadata {
  startedAt?: string;
}

function getExecutedAt(context: ExecutionDetailsContext): string | undefined {
  const metadata = context.execution.metadata as CreateFunctionExecutionMetadata | undefined;
  const ts = metadata?.startedAt ?? context.execution.createdAt;
  return ts ? new Date(ts).toLocaleString() : undefined;
}

function getOutputData(context: ExecutionDetailsContext): CreateFunctionOutputData | undefined {
  const outputs = context.execution.outputs as { default?: CreateFunctionOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  return payload?.data;
}

export const createFunctionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return {
      ...baseMapper.props(context),
      metadata: createFunctionMetadataList(context.node),
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
      details["Function Name"] = data.displayName;
    }
    if (data.image) {
      details["Image"] = data.image;
    }
    if (data.memoryInMBs != null) {
      details["Memory (MB)"] = String(data.memoryInMBs);
    }
    if (data.lifecycleState) {
      details["State"] = data.lifecycleState;
    }
    if (data.invokeEndpoint) {
      details["Invoke Endpoint"] = data.invokeEndpoint;
    }

    return details;
  },
};

function createFunctionMetadataList(node: ComponentBaseContext["node"]): MetadataItem[] {
  const config = node.configuration as CreateFunctionConfiguration | undefined;
  const nodeMeta = node.metadata as CreateFunctionNodeMetadata | undefined;
  const items: MetadataItem[] = [];

  if (config?.displayName) {
    items.push({ icon: "tag", label: config.displayName });
  }

  const appLabel = nodeMeta?.applicationName ?? config?.application;
  if (appLabel) {
    items.push({ icon: "layout-grid", label: appLabel });
  }

  if (config?.image) {
    items.push({ icon: "box", label: config.image });
  }

  return items;
}
