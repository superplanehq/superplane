import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  OutputPayload,
  NodeInfo,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import { MetadataItem } from "@/ui/metadataList";

interface WorkflowUsageOutput {
  minutes_used?: number;
  minutes_used_breakdown?: Record<string, number>;
  net_amount?: number;
  repositories?: string[];
  usage_items?: Array<{
    repositoryName?: string;
    sku?: string;
    quantity?: number;
    unitType?: string;
  }>;
}

interface GetWorkflowUsageConfiguration {
  repositories?: string[];
  year?: string;
  month?: string;
  day?: string;
  product?: string;
  sku?: string;
}

function getWorkflowUsageMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetWorkflowUsageConfiguration | undefined;

  if (configuration?.repositories && configuration.repositories.length > 0) {
    metadata.push({ icon: "book", label: `${configuration.repositories.length} repositories` });
  } else {
    metadata.push({ icon: "book", label: "All repositories" });
  }

  if (configuration?.product) {
    metadata.push({ icon: "tag", label: configuration.product });
  }

  if (configuration?.year || configuration?.month || configuration?.day) {
    const year = configuration?.year || "current year";
    const month = configuration?.month || "current month";
    const day = configuration?.day || "all days";
    metadata.push({ icon: "clock", label: `${year}/${month}/${day}` });
  }

  if (configuration?.sku) {
    metadata.push({ icon: "server", label: configuration.sku });
  }

  return metadata;
}

export const getWorkflowUsageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);

    return {
      ...base,
      metadata: getWorkflowUsageMetadata(context.node),
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const usage = outputs.default[0].data as WorkflowUsageOutput;
    details["Retrieved At"] = context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-";
    details["Minutes Used"] = `${usage.minutes_used || 0}`;
    details["Net Amount"] = `${usage.net_amount || 0}`;

    const breakdown = usage.minutes_used_breakdown || {};
    if (Object.keys(breakdown).length > 0) {
      details["OS Breakdown"] = Object.entries(breakdown)
        .map(([k, v]) => `${k}: ${v}`)
        .join(", ");
    }

    if (usage.repositories && usage.repositories.length > 0) {
      details["Repositories"] = usage.repositories.join(", ");
    }

    if (usage.usage_items && usage.usage_items.length > 0) {
      details["Rows"] = usage.usage_items.length.toString();
    }

    return details;
  },
};
