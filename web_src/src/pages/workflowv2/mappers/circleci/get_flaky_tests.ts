import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../utils";
import { baseProps } from "./base";

interface GetFlakyTestsConfiguration {
  projectSlug?: string;
}

interface FlakyTestsOutput {
  flakyTests?: Array<{
    testName?: string;
    classname?: string;
    workflowName?: string;
    jobName?: string;
    timesFlaky?: number;
    file?: string;
  }>;
  totalFlakyTests?: number;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetFlakyTestsConfiguration | undefined;

  if (configuration?.projectSlug) {
    metadata.push({ icon: "workflow", label: `Project: ${configuration.projectSlug}` });
  }

  return metadata;
}

export const getFlakyTestsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as FlakyTestsOutput | undefined;

    const details: Record<string, string> = {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Total Flaky Tests": stringOrDash(result?.totalFlakyTests),
    };

    const flakyTests = result?.flakyTests;
    if (flakyTests && flakyTests.length > 0) {
      details["Flaky Tests"] = flakyTests.map((t) => `${t.testName || "-"} (flaky ${t.timesFlaky || 0}x)`).join(", ");
    }

    return details;
  },
};
