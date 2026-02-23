import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import CircleCILogo from "@/assets/icons/integrations/circleci.svg";

interface GetTestMetricsOutput {
  average_test_count?: number;
  total_test_runs?: number;
  most_failed_tests?: Array<{ test_name?: string; failed_runs?: number }>;
  slowest_tests?: Array<{ test_name?: string; p95_duration?: number }>;
}

export const getTestMetricsMapper: ComponentBaseMapper = {
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
    const result = outputs?.default?.[0]?.data as GetTestMetricsOutput | undefined;

    const details: Record<string, string> = {};

    if (result?.total_test_runs !== undefined) {
      details["Total Test Runs"] = String(result.total_test_runs);
    }

    if (result?.average_test_count !== undefined) {
      details["Avg Test Count"] = String(result.average_test_count);
    }

    if (result?.most_failed_tests && result.most_failed_tests.length > 0) {
      details["Most Failed"] = result.most_failed_tests
        .slice(0, 3)
        .map((t) => t.test_name)
        .filter(Boolean)
        .join(", ");
    }

    if (result?.slowest_tests && result.slowest_tests.length > 0) {
      details["Slowest Tests"] = result.slowest_tests
        .slice(0, 3)
        .map((t) => t.test_name)
        .filter(Boolean)
        .join(", ");
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
  const configuration = node.configuration as { projectSlug?: string; workflowName?: string } | undefined;
  const nodeMetadata = node.metadata as { projectName?: string } | undefined;

  const projectLabel = nodeMetadata?.projectName || configuration?.projectSlug;
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }

  if (configuration?.workflowName) {
    metadata.push({ icon: "workflow", label: configuration.workflowName });
  }

  return metadata;
}

function getEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}
