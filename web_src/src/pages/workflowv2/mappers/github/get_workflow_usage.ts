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

const MAX_REPOSITORIES_IN_SUMMARY = 3;
const MAX_BREAKDOWN_ENTRIES = 3;

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

function getMetadataRepositoryNames(node: NodeInfo): string[] {
  const nodeMetadata = node.metadata as GetWorkflowUsageMetadata | undefined;
  return (nodeMetadata?.repositories ?? []).map((repository) => repository.name).filter((name) => !!name);
}

function getWorkflowUsageMetadataList(node: NodeInfo): MetadataItem[] {
  const repositoryNames = getMetadataRepositoryNames(node);

  if (repositoryNames.length === 0) {
    return [{ icon: "book", label: "Organization-wide" }];
  }

  return [
    {
      icon: "book",
      label:
        repositoryNames.length > 1 ? `${repositoryNames[0]} +${repositoryNames.length - 1} more` : repositoryNames[0],
    },
  ];
}

function formatMinutes(value: number): string {
  return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

function formatRepositoryScope(repositories: string[]): string {
  if (repositories.length === 0) {
    return "Organization-wide";
  }

  const summary = repositories.slice(0, MAX_REPOSITORIES_IN_SUMMARY).join(", ");
  if (repositories.length > MAX_REPOSITORIES_IN_SUMMARY) {
    return `${summary} +${repositories.length - MAX_REPOSITORIES_IN_SUMMARY} more`;
  }

  return summary;
}

function formatBreakdownSummary(breakdown?: Record<string, number>): string | undefined {
  if (!breakdown) {
    return undefined;
  }

  const sortedEntries = Object.entries(breakdown)
    .filter(([, minutes]) => Number.isFinite(minutes) && minutes > 0)
    .sort((a, b) => b[1] - a[1]);

  if (sortedEntries.length === 0) {
    return undefined;
  }

  const summary = sortedEntries
    .slice(0, MAX_BREAKDOWN_ENTRIES)
    .map(([sku, minutes]) => `${sku}: ${formatMinutes(minutes)}`)
    .join(", ");

  if (sortedEntries.length > MAX_BREAKDOWN_ENTRIES) {
    return `${summary} +${sortedEntries.length - MAX_BREAKDOWN_ENTRIES} more`;
  }

  return summary;
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
    const usage = outputs?.default?.[0]?.data as WorkflowUsageOutput | undefined;
    const configuredRepositories = getMetadataRepositoryNames(context.node);
    const scopeRepositories = (usage?.repositories ?? configuredRepositories).filter((repository) => !!repository);
    const details: Record<string, string> = {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Usage Scope": formatRepositoryScope(scopeRepositories),
    };

    if (!usage) {
      return details;
    }

    if (usage.minutes_used !== undefined) {
      details["Minutes Used"] = formatMinutes(usage.minutes_used);
    }

    if (usage.total_paid_minutes_used !== undefined) {
      details["Paid Minutes Used"] = formatMinutes(usage.total_paid_minutes_used);
    }

    if (usage.included_minutes !== undefined && usage.included_minutes > 0) {
      details["Included Minutes"] = formatMinutes(usage.included_minutes);
    }

    const breakdownSummary = formatBreakdownSummary(usage.minutes_used_breakdown);
    if (breakdownSummary) {
      details["Breakdown"] = breakdownSummary;
    }

    return details;
  },
};
