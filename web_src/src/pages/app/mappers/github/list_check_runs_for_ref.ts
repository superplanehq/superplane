import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
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
import type { MetadataItem } from "@/ui/metadataList";

interface Configuration {
  ref?: string;
  checkName?: string;
  status?: string;
  filter?: string;
}

interface CheckRun {
  name?: string;
  status?: string;
  conclusion?: string;
  html_url?: string;
  details_url?: string;
}

interface ListCheckRunsOutput {
  total_count?: number;
  check_runs?: CheckRun[];
}

export const listCheckRunsForRefMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);

    return {
      ...base,
      metadata: listCheckRunsMetadataItems(context.node),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const output = firstOutput(context.execution.outputs) as ListCheckRunsOutput | undefined;
    if (!output) {
      return {};
    }

    const checkRuns = output.check_runs || [];
    const unsuccessfulCheck = checkRuns.find((checkRun) => isUnsuccessfulConclusion(checkRun.conclusion));

    return {
      "Total check runs": String(output.total_count ?? checkRuns.length),
      Completed: String(checkRuns.filter((checkRun) => checkRun.status === "completed").length),
      Successful: String(checkRuns.filter((checkRun) => checkRun.conclusion === "success").length),
      "Not successful": String(checkRuns.filter((checkRun) => isUnsuccessfulConclusion(checkRun.conclusion)).length),
      Pending: String(checkRuns.filter((checkRun) => checkRun.status && checkRun.status !== "completed").length),
      "First non-green check": unsuccessfulCheck?.name || "-",
    };
  },
};

function listCheckRunsMetadataItems(node: NodeInfo): MetadataItem[] {
  const metadataItems: MetadataItem[] = [];
  const metadata = node.metadata as { repository?: { name?: string } } | undefined;
  const configuration = node.configuration as Configuration | undefined;

  if (metadata?.repository?.name) {
    metadataItems.push({ icon: "book", label: metadata.repository.name });
  }

  if (configuration?.ref) {
    metadataItems.push({ icon: "git-commit", label: configuration.ref });
  }

  if (configuration?.checkName) {
    metadataItems.push({ icon: "circle-check", label: configuration.checkName });
  }

  if (configuration?.status) {
    metadataItems.push({ icon: "funnel", label: configuration.status });
  }

  if (configuration?.filter) {
    metadataItems.push({ icon: "list-filter", label: configuration.filter });
  }

  return metadataItems;
}

function firstOutput(outputs: unknown): unknown {
  const outputPayloads = outputs as { default?: OutputPayload[] } | undefined;
  return outputPayloads?.default?.[0]?.data;
}

function isUnsuccessfulConclusion(conclusion: string | undefined): boolean {
  return Boolean(conclusion && !["success", "neutral", "skipped"].includes(conclusion));
}
