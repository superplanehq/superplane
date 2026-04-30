import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface CreateApplicationConfiguration {
  compartment?: string;
  displayName?: string;
  vcn?: string;
  subnet?: string;
  shape?: string;
}

interface CreateApplicationOutputData {
  applicationId?: string;
  displayName?: string;
  lifecycleState?: string;
  compartmentId?: string;
  timeCreated?: string;
}

type CreateApplicationOutputPayload = OutputPayload & {
  data?: CreateApplicationOutputData;
};

interface CreateApplicationExecutionMetadata {
  startedAt?: string;
}

function getExecutedAt(context: ExecutionDetailsContext): string | undefined {
  const metadata = context.execution.metadata as CreateApplicationExecutionMetadata | undefined;
  const ts = metadata?.startedAt ?? context.execution.createdAt;
  return ts ? new Date(ts).toLocaleString() : undefined;
}

function getOutputData(context: ExecutionDetailsContext): CreateApplicationOutputData | undefined {
  const outputs = context.execution.outputs as { default?: CreateApplicationOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  return payload?.data;
}

export const createApplicationMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return {
      ...baseMapper.props(context),
      metadata: createApplicationMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const data = getOutputData(context);

    const executedAt = getExecutedAt(context);
    if (executedAt) {
      details["Executed At"] = executedAt;
    }

    if (!data) return details;

    if (data.displayName) {
      details["Display Name"] = data.displayName;
    }
    if (data.lifecycleState) {
      details["State"] = data.lifecycleState;
    }
    if (data.applicationId) {
      details["Application ID"] = data.applicationId;
    }

    return details;
  },
};

interface CreateApplicationNodeMetadata {
  subnetId?: string;
  subnetName?: string;
  shape?: string;
}

function createApplicationMetadataList(node: ComponentBaseContext["node"]): MetadataItem[] {
  const config = node.configuration as CreateApplicationConfiguration | undefined;
  const nodeMeta = node.metadata as CreateApplicationNodeMetadata | undefined;
  const items: MetadataItem[] = [];

  if (config?.displayName) {
    items.push({ icon: "tag", label: config.displayName });
  }

  const subnetLabel = nodeMeta?.subnetName ?? config?.subnet;
  if (subnetLabel) {
    items.push({ icon: "network", label: subnetLabel });
  }

  const shape = nodeMeta?.shape ?? config?.shape;
  if (shape) {
    items.push({ icon: "cpu", label: shape });
  }

  return items;
}
