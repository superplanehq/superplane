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
  compartmentId?: string;
  applicationId?: string;
  displayName?: string;
  image?: string;
  memoryInMBs?: number;
  timeoutInSeconds?: number;
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

function getOutputData(context: ExecutionDetailsContext): CreateFunctionOutputData | undefined {
  const outputs = context.execution.outputs as { default?: CreateFunctionOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  if (!payload) return undefined;
  return (payload.data ?? payload) as CreateFunctionOutputData;
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
  const items: MetadataItem[] = [];

  if (config?.displayName) {
    items.push({ icon: "tag", label: config.displayName });
  }
  if (config?.image) {
    items.push({ icon: "box", label: config.image });
  }

  return items;
}
