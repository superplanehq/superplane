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

interface GetRecentWorkflowRunsOutput {
  workflows?: Array<{
    name?: string;
    metrics?: {
      success_rate?: number;
      total_runs?: number;
    };
  }>;
  total?: number;
}

export const getRecentWorkflowRunsMapper: ComponentBaseMapper = {
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
    const result = outputs?.default?.[0]?.data as GetRecentWorkflowRunsOutput | undefined;

    const details: Record<string, string> = {};

    if (result?.total !== undefined) {
      details["Workflows"] = String(result.total);
    }

    if (result?.workflows && result.workflows.length > 0) {
      const names = result.workflows.map((w) => w.name).filter(Boolean);
      if (names.length > 0) {
        details["Workflow Names"] = names.join(", ");
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
