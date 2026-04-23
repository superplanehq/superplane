import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface DeleteInstanceConfiguration {
  instanceId?: string;
  preserveBootVolume?: boolean;
}

interface InstanceOutputData {
  instanceId?: string;
  lifecycleState?: string;
}

type InstanceOutputPayload = OutputPayload & {
  data?: InstanceOutputData;
};

function getOutputData(context: ExecutionDetailsContext): InstanceOutputData | undefined {
  const outputs = context.execution.outputs as { default?: InstanceOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  if (!payload) return undefined;
  return (payload.data ?? payload) as InstanceOutputData;
}

function executedAt(context: ExecutionDetailsContext): string {
  return context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-";
}

export const deleteInstanceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const props = baseMapper.props(context);
    const config = context.node.configuration as DeleteInstanceConfiguration | undefined;
    const metadata: MetadataItem[] = [];

    if (config?.instanceId) {
      metadata.push({ icon: "trash-2", label: config.instanceId });
    }
    if (config?.preserveBootVolume) {
      metadata.push({ icon: "archive", label: "Preserve boot volume" });
    }

    return {
      ...props,
      metadata,
    };
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const config = context.node.configuration as DeleteInstanceConfiguration | undefined;
    const data = getOutputData(context);
    return {
      "Executed At": executedAt(context),
      "Instance ID": data?.instanceId ?? "-",
      State: data?.lifecycleState ?? "-",
      "Preserve Boot Volume": config?.preserveBootVolume ? "Yes" : "No",
    };
  },
};
