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
import { UpsertCheckRuleConfiguration } from "./types";

export const updateCheckRuleMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Update Check Rule",
      eventSections: lastExecution ? buildDash0EventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    return buildDash0ExecutionDetails(context.execution, "Update Response");
  },

  subtitle(context: SubtitleContext): string {
    return formatTimeAgo(new Date(context.execution.createdAt!));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpsertCheckRuleConfiguration;

  if (configuration?.originOrId) {
    metadata.push({
      icon: "hash",
      label: configuration.originOrId,
    });
  }

  return metadata;
}
