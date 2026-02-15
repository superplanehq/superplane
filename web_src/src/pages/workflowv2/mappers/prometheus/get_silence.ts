import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
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
import { GetSilenceConfiguration, PrometheusSilencePayload } from "./types";
import { getDetailsForSilence } from "./create_silence";

export const getSilenceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { nodes, node, componentDefinition, lastExecutions } = context;
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.componentName || "unknown";

    return {
      iconSrc: prometheusIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? buildEventSections(nodes, lastExecution, componentName) : undefined,
      metadata: getMetadata(node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
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

    const silence = outputs.default[0].data as PrometheusSilencePayload;
    return {
      ...details,
      ...getDetailsForSilence(silence),
    };
  },
};

function getMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetSilenceConfiguration | undefined;

  if (configuration?.silenceID) {
    metadata.push({ icon: "search", label: `ID: ${truncateID(configuration.silenceID)}` });
  }

  return metadata.slice(0, 3);
}

function truncateID(id: string): string {
  if (id.length <= 12) {
    return id;
  }
  return id.substring(0, 12) + "…";
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
