import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { ExpireSilenceConfiguration, ExpireSilenceNodeMetadata, PrometheusSilencePayload } from "./types";

export const expireSilenceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildExpireSilenceProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, any> = {};

    if (context.execution.createdAt) {
      details["Expired At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const silence = outputs.default[0].data as PrometheusSilencePayload;

    if (silence?.silenceID) {
      details["Silence ID"] = silence.silenceID;
    }

    if (silence?.status) {
      details["Status"] = silence.status;
    }

    return details;
  },
};

function buildExpireSilenceProps(
  nodes: NodeInfo[],
  node: NodeInfo,
  componentDefinition: { name: string; label: string; color: string },
  lastExecutions: ExecutionInfo[],
): ComponentBaseProps {
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
}

function getMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as ExpireSilenceNodeMetadata | undefined;
  const configuration = node.configuration as ExpireSilenceConfiguration | undefined;

  const selectedSilence = configuration?.silence || configuration?.silenceID;
  if (selectedSilence) {
    metadata.push({ icon: "bell-off", label: selectedSilence });
  }

  if (nodeMetadata?.silenceID && nodeMetadata.silenceID !== selectedSilence) {
    metadata.push({ icon: "bell-off", label: nodeMetadata.silenceID });
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
      eventSubtitle: execution.createdAt ? renderTimeAgo(new Date(execution.createdAt)) : "",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
