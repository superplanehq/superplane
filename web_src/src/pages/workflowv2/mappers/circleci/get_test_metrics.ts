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
  most_failed_tests?: Array<{
    test_name?: string;
    classname?: string;
    failed_runs?: number;
    total_runs?: number;
    flaky?: boolean;
  }>;
  slowest_tests?: Array<{
    test_name?: string;
    p50_duration_secs?: number;
  }>;
  total_test_runs?: number;
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
      "Total Test Runs": stringOrDash(result?.total_test_runs),
    };

    if (result?.most_failed_tests && result.most_failed_tests.length > 0) {
      details["Most Failed Tests"] = result.most_failed_tests
        .map((t) => `${t.test_name || "-"} (${t.failed_runs || 0}/${t.total_runs || 0} failures)`)
        .join(", ");
    }

    if (result?.slowest_tests && result.slowest_tests.length > 0) {
      details["Slowest Tests"] = result.slowest_tests
        .map((t) => `${t.test_name || "-"} (${t.p50_duration_secs?.toFixed(1) || "-"}s)`)
        .join(", ");
    }

    return details;
  },
};
