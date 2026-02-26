import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../utils";
import { baseProps } from "./base";

interface GetTestMetricsConfiguration {
  projectSlug?: string;
  workflowName?: string;
}

interface TestMetricsOutput {
  mostFailedTests?: Array<{
    testName?: string;
    classname?: string;
    failedRuns?: number;
    totalRuns?: number;
    flaky?: boolean;
  }>;
  slowestTests?: Array<{
    testName?: string;
    classname?: string;
    failedRuns?: number;
    totalRuns?: number;
    flaky?: boolean;
    p50DurationSecs?: number;
  }>;
  totalTestRuns?: number;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetTestMetricsConfiguration | undefined;

  if (configuration?.projectSlug) {
    metadata.push({ icon: "workflow", label: `Project: ${configuration.projectSlug}` });
  }

  if (configuration?.workflowName) {
    metadata.push({ icon: "play", label: `Workflow: ${configuration.workflowName}` });
  }

  return metadata;
}

export const getTestMetricsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as TestMetricsOutput | undefined;

    const details: Record<string, string> = {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Total Test Runs": stringOrDash(result?.totalTestRuns),
    };

    if (result?.mostFailedTests && result.mostFailedTests.length > 0) {
      details["Most Failed Tests"] = result.mostFailedTests
        .map((t) => `${t.testName || "-"} (${t.failedRuns || 0}/${t.totalRuns || 0} failures)`)
        .join(", ");
    }

    if (result?.slowestTests && result.slowestTests.length > 0) {
      details["Slowest Tests"] = result.slowestTests
        .map((t) => {
          const duration = t.p50DurationSecs != null ? ` (${t.p50DurationSecs.toFixed(1)}s)` : "";
          return `${t.testName || "-"}${duration}`;
        })
        .join(", ");
    }

    return details;
  },
};
