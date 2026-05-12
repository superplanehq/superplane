import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
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
import restateIcon from "@/assets/icons/integrations/restate.svg";
import type { InvocationAction } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";

/**
 * Shared mapper for Cancel, Kill, and Purge Invocation components.
 * They all take an invocation ID and return a status.
 */
function createInvocationActionMapper(): ComponentBaseMapper {
  return {
    props(context: ComponentBaseContext): ComponentBaseProps {
      const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
      const componentName = context.componentDefinition.name || "unknown";

      return {
        iconSrc: restateIcon,
        collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
        collapsed: context.node.isCollapsed,
        title:
          context.node.name ||
          context.componentDefinition.label ||
          context.componentDefinition.name ||
          "Unnamed component",
        eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
        metadata: metadataList(context.node),
        includeEmptyState: !lastExecution,
        eventStateMap: getStateMap(componentName),
      };
    },

    getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
      const outputs = context.execution.outputs as { default: OutputPayload[] };
      if (!outputs?.default?.[0]?.data) {
        return {};
      }
      const data = outputs.default[0].data as InvocationAction;
      const details: Record<string, string> = {};

      if (data?.invocation_id) details["Invocation ID"] = data.invocation_id;
      if (data?.status) details["Status"] = data.status;

      return details;
    },

    subtitle(context: SubtitleContext): string | React.ReactNode {
      if (!context.execution.createdAt) return "";
      return renderTimeAgo(new Date(context.execution.createdAt));
    },
  };
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration?.invocationId) {
    metadata.push({ icon: "hash", label: configuration.invocationId });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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

export const cancelInvocationMapper = createInvocationActionMapper();
export const killInvocationMapper = createInvocationActionMapper();
export const purgeInvocationMapper = createInvocationActionMapper();
