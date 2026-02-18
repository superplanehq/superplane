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
  included_minutes?: number;
  total_paid_minutes_used?: number;
  repositories?: string[];
}

interface GetWorkflowUsageMetadata {
  repositories?: Array<{
    id: number;
    name: string;
    url: string;
  }>;
}

function getWorkflowUsageMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as GetWorkflowUsageMetadata | undefined;

  // Show selected repositories if any (up to 3 shown inline, rest in tooltip)
  if (nodeMetadata?.repositories && nodeMetadata.repositories.length > 0) {
    const repoNames = nodeMetadata.repositories.map((repo) => repo.name);
    const displayNames = repoNames.slice(0, 3).join(", ");
    const label = repoNames.length > 3 ? `${displayNames} +${repoNames.length - 3} more` : displayNames;

    metadata.push({
      icon: "book",
      label: label,
    });
  }

  return metadata;
}

export const getWorkflowUsageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);

    // Override metadata to show repositories
    const metadata = getWorkflowUsageMetadataList(context.node);

    return {
      ...base,
      metadata: metadata.length > 0 ? metadata : undefined,
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (outputs && outputs.default && outputs.default.length > 0) {
      const usage = outputs.default[0].data as WorkflowUsageOutput;
      Object.assign(details, {
        "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      });

      if (usage.minutes_used !== undefined) {
        details["Minutes Used"] = usage.minutes_used.toFixed(2);
      }

      if (usage.total_paid_minutes_used !== undefined) {
        details["Paid Minutes Used"] = usage.total_paid_minutes_used.toFixed(2);
      }

      if (usage.included_minutes !== undefined) {
        details["Included Minutes"] = usage.included_minutes.toFixed(2);
      }

      // Add breakdown by runner type
      if (usage.minutes_used_breakdown) {
        const breakdown = Object.entries(usage.minutes_used_breakdown)
          .map(([sku, minutes]) => `${sku}: ${minutes.toFixed(2)}`)
          .join(", ");
        if (breakdown) {
          details["Breakdown"] = breakdown;
        }
      }

      // Show tracked repositories if any
      if (usage.repositories && usage.repositories.length > 0) {
        details["Tracked Repositories"] = usage.repositories.join(", ");
      }
    }

    return details;
  },
};
