import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";

interface CompareCommitsConfiguration {
  base?: string;
  head?: string;
  statuses?: string[];
  paths?: string[];
}

interface ComparisonPayload {
  comparison?: {
    changedFileCount?: number;
    baseSha?: string;
    headSha?: string;
    url?: string;
  };
  appliedFilters?: {
    statuses?: string[];
    paths?: string[];
  };
  files?: unknown[];
}

const SUMMARY_LIMIT = 36;

export const compareCommitsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions),
      metadata: compareCommitsMetadata(context.node),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const { payload, filtered } = comparisonOutput(context.execution.outputs);
    if (!payload) {
      return {};
    }

    return {
      "Changed files": displayCount(payload.comparison?.changedFileCount),
      ...(filtered ? { "Matched files": displayCount(payload.files?.length) } : {}),
      "Base SHA": payload.comparison?.baseSha || "-",
      "Head SHA": payload.comparison?.headSha || "-",
      "Comparison URL": payload.comparison?.url || "-",
    };
  },
};

function displayCount(count: number | undefined): string {
  return count === undefined ? "-" : String(count);
}

function compareCommitsMetadata(node: NodeInfo): MetadataItem[] {
  const metadata = node.metadata as { repository?: { name?: string } } | undefined;
  const configuration = node.configuration as CompareCommitsConfiguration | undefined;

  return [repositoryMetadata(metadata), refsMetadata(configuration), filterMetadata(configuration)]
    .filter((item): item is MetadataItem => Boolean(item))
    .slice(0, 3);
}

function repositoryMetadata(metadata: { repository?: { name?: string } } | undefined): MetadataItem | undefined {
  return metadata?.repository?.name ? { icon: "book", label: metadata.repository.name } : undefined;
}

function refsMetadata(configuration: CompareCommitsConfiguration | undefined): MetadataItem | undefined {
  if (!configuration?.base && !configuration?.head) {
    return undefined;
  }

  return { icon: "git-compare", label: `${configuration.base || "-"} → ${configuration.head || "-"}` };
}

function filterMetadata(configuration: CompareCommitsConfiguration | undefined): MetadataItem | undefined {
  if (!configuration) {
    return undefined;
  }

  const hasStatuses = Object.prototype.hasOwnProperty.call(configuration, "statuses");
  const hasPaths = Object.prototype.hasOwnProperty.call(configuration, "paths");
  if (hasStatuses && hasPaths) {
    return { icon: "funnel", label: "2 filters" };
  }
  if (hasStatuses) {
    return { icon: "funnel", label: compactSummary(configuration.statuses || [], "All statuses") };
  }
  if (hasPaths) {
    return { icon: "funnel", label: compactSummary(configuration.paths || [], "All paths") };
  }

  return undefined;
}

function compactSummary(values: string[], emptySummary: string): string {
  const summary =
    values
      .map((value) => value.trim())
      .filter(Boolean)
      .join(", ") || emptySummary;
  return summary.length <= SUMMARY_LIMIT ? summary : `${summary.slice(0, SUMMARY_LIMIT - 1)}…`;
}

function comparisonOutput(outputs: unknown): { payload?: ComparisonPayload; filtered: boolean } {
  const outputPayloads = outputs as Record<string, OutputPayload[]> | undefined;
  for (const channel of ["matched", "unmatched"]) {
    const payload = outputPayloads?.[channel]?.[0]?.data as ComparisonPayload | undefined;
    if (payload) {
      return { payload, filtered: Boolean(payload.appliedFilters) };
    }
  }
  return { filtered: false };
}
