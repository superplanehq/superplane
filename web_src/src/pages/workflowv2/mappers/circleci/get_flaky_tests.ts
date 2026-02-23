import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getStateMap } from "..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import CircleCILogo from "@/assets/icons/integrations/circleci.svg";
import { getEventSections } from "./common";

interface GetFlakyTestsOutput {
  flaky_tests?: Array<{
    test_name?: string;
    times_flaked?: number;
    workflow_name?: string;
    job_name?: string;
    file?: string;
  }>;
  total_count?: number;
}

export const getFlakyTestsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: CircleCILogo,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as GetFlakyTestsOutput | undefined;

    const details: Record<string, string> = {};

    if (result?.total_count !== undefined) {
      details["Flaky Tests"] = String(result.total_count);
    }

    if (result?.flaky_tests && result.flaky_tests.length > 0) {
      const topFlaky = result.flaky_tests
        .slice(0, 3)
        .filter((t) => t.test_name != null && t.times_flaked != null)
        .map((t) => `${t.test_name} (${t.times_flaked}x)`)
        .join(", ");
      if (topFlaky) {
        details["Top Flaky"] = topFlaky;
      }
    }

    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    if (timestamp) {
      details["Retrieved At"] = new Date(timestamp).toLocaleString();
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { projectSlug?: string } | undefined;
  const nodeMetadata = node.metadata as { projectName?: string } | undefined;

  const projectLabel = nodeMetadata?.projectName || configuration?.projectSlug;
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }

  return metadata;
}
