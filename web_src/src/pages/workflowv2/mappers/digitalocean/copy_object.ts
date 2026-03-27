import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/utils/colors";
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
import type { MetadataItem } from "@/ui/metadataList";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { CopyObjectConfiguration, CopyObjectOutput } from "./types";
import { DEFAULT_STATE_REGISTRY } from "../stateRegistry";
import type { EventStateRegistry } from "../types";

type CopyObjectOutputs = {
  default?: OutputPayload[];
};

export const COPY_OBJECT_STATE_REGISTRY: EventStateRegistry = DEFAULT_STATE_REGISTRY;

export const copyObjectMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "digitalocean";

    return {
      iconSrc: doIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, unknown> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as CopyObjectOutputs | undefined;
    const result = outputs?.default?.[0]?.data as CopyObjectOutput | undefined;

    if (!result) return details;

    details["Source"] = result.sourceFilePath || "-";
    details["Destination"] = result.destinationFilePath || "-";
    if (result.moved !== undefined) {
      details["Action"] = result.moved ? "Moved" : "Copied";
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as CopyObjectConfiguration;

  if (configuration?.sourceBucket) {
    metadata.push({ icon: "database", label: configuration.sourceBucket });
  }

  if (configuration?.sourceFilePath && configuration?.destinationFilePath) {
    metadata.push({
      icon: configuration.deleteSource ? "move" : "copy",
      label: `${configuration.sourceFilePath} → ${configuration.destinationFilePath}`,
    });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id!,
    },
  ];
}
