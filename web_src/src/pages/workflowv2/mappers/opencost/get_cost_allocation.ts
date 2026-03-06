import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import opencostIcon from "@/assets/icons/integrations/opencost.svg";
import { CostAllocationPayload, GetCostAllocationConfiguration } from "./types";
import { formatTimeAgo } from "@/utils/date";

export const getCostAllocationMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: opencostIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: getMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default: OutputPayload[] };
    if (!outputs?.default?.[0]?.data) {
      return {};
    }
    const payload = outputs.default[0].data as CostAllocationPayload;
    return getDetailsForAllocation(payload);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetCostAllocationConfiguration | undefined;

  if (configuration?.window) {
    metadata.push({ icon: "clock", label: `Window: ${configuration.window}` });
  }

  if (configuration?.aggregate) {
    metadata.push({ icon: "layers", label: `By: ${configuration.aggregate}` });
  }

  return metadata.slice(0, 3);
}

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function getDetailsForAllocation(payload: CostAllocationPayload): Record<string, string> {
  const details: Record<string, string> = {};

  if (payload?.window) {
    details["Window"] = payload.window;
  }

  if (payload?.aggregate) {
    details["Aggregate By"] = payload.aggregate;
  }

  if (payload?.totalCost !== undefined) {
    details["Total Cost"] = `$${payload.totalCost.toFixed(2)}`;
  }

  if (payload?.allocations && payload.allocations.length > 0) {
    details["Allocations"] = String(payload.allocations.length);
  }

  return details;
}
