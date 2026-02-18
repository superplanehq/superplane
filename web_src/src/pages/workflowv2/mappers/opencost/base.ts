import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import opencostIcon from "@/assets/icons/integrations/opencost.svg";
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
import { CostAllocationPayload, GetCostAllocationConfiguration } from "./types";

const windowLabels: Record<string, string> = {
  "1h": "1 Hour",
  "1d": "1 Day",
  "2d": "2 Days",
  "7d": "7 Days",
};

const aggregateLabels: Record<string, string> = {
  namespace: "Namespace",
  cluster: "Cluster",
  controller: "Controller",
  service: "Service",
  deployment: "Deployment",
};

export const baseCostAllocationMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildBaseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }

    return formatTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, any> = {};

    if (context.execution.createdAt) {
      details["Retrieved At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const allocation = outputs.default[0].data as CostAllocationPayload;
    return {
      ...details,
      ...getDetailsForAllocation(allocation),
    };
  },
};

export function buildBaseProps(
  nodes: NodeInfo[],
  node: NodeInfo,
  componentDefinition: { name: string; label: string; color: string },
  lastExecutions: ExecutionInfo[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
  const componentName = componentDefinition.name || node.componentName || "unknown";

  return {
    iconSrc: opencostIcon,
    iconColor: getColorClass(componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(componentDefinition.color),
    collapsed: node.isCollapsed,
    title: node.name || componentDefinition.label || "Unnamed component",
    eventSections: lastExecution ? buildEventSections(nodes, lastExecution, componentName) : undefined,
    metadata: getMetadata(node),
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

export function getDetailsForAllocation(allocation: CostAllocationPayload): Record<string, string> {
  const details: Record<string, string> = {};

  if (allocation?.name) {
    details["Name"] = allocation.name;
  }

  if (allocation?.totalCost !== undefined) {
    details["Total Cost"] = `$${allocation.totalCost.toFixed(2)}`;
  }

  if (allocation?.cpuCost !== undefined && allocation.cpuCost > 0) {
    details["CPU Cost"] = `$${allocation.cpuCost.toFixed(2)}`;
  }

  if (allocation?.ramCost !== undefined && allocation.ramCost > 0) {
    details["RAM Cost"] = `$${allocation.ramCost.toFixed(2)}`;
  }

  if (allocation?.pvCost !== undefined && allocation.pvCost > 0) {
    details["PV Cost"] = `$${allocation.pvCost.toFixed(2)}`;
  }

  if (allocation?.networkCost !== undefined && allocation.networkCost > 0) {
    details["Network Cost"] = `$${allocation.networkCost.toFixed(2)}`;
  }

  if (allocation?.window) {
    details["Window"] = windowLabels[allocation.window] || allocation.window;
  }

  if (allocation?.aggregate) {
    details["Aggregate"] = aggregateLabels[allocation.aggregate] || allocation.aggregate;
  }

  if (allocation?.start) {
    details["Start"] = new Date(allocation.start).toLocaleString();
  }

  if (allocation?.end) {
    details["End"] = new Date(allocation.end).toLocaleString();
  }

  return details;
}

function getMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetCostAllocationConfiguration | undefined;

  if (configuration?.window) {
    metadata.push({ icon: "clock", label: windowLabels[configuration.window] || configuration.window });
  }

  if (configuration?.aggregate) {
    metadata.push({
      icon: "layers",
      label: aggregateLabels[configuration.aggregate] || configuration.aggregate,
    });
  }

  if (configuration?.filter) {
    metadata.push({ icon: "funnel", label: configuration.filter });
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
      eventSubtitle: execution.createdAt ? formatTimeAgo(new Date(execution.createdAt)) : "",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
