import { ComponentBaseProps } from "@/ui/componentBase";
import { getStateMap } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { buildDash0EventSections, buildDash0ExecutionDetails } from "./base";
import { SendLogEventConfiguration } from "./types";

export const sendLogEventMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Send Log Event",
      eventSections: lastExecution
        ? buildDash0EventSections(context.nodes, lastExecution, componentName, getSubtitlePrefix(lastExecution))
        : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    return buildDash0ExecutionDetails(context.execution, "Response");
  },

  subtitle(context: SubtitleContext): string {
    return formatTimeAgo(new Date(context.execution.createdAt!));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as SendLogEventConfiguration;

  if (configuration?.serviceName) {
    metadata.push({
      icon: "database",
      label: configuration.serviceName,
    });
  }

  if (configuration?.records && configuration.records.length > 0) {
    metadata.push({
      icon: "list",
      label: `${configuration.records.length} records`,
    });
  }

  return metadata;
}

function getSubtitlePrefix(execution: { outputs?: unknown }): string | undefined {
  const outputs = execution.outputs as { default?: Array<{ data?: { sentCount?: number } }> } | undefined;
  const sentCount = outputs?.default?.[0]?.data?.sentCount;
  if (typeof sentCount === "number") {
    return `${sentCount} sent`;
  }

  return undefined;
}
