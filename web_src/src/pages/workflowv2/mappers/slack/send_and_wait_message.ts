import {
  ComponentsComponent,
  ComponentsNode,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import { formatTimeAgo } from "@/utils/date";

interface SendAndWaitMessageConfiguration {
  channel?: string;
  message?: string;
  timeout?: number;
  buttons?: { name: string; value: string }[];
}

interface SendAndWaitMessageMetadata {
  channel?: {
    id?: string;
    name?: string;
  };
}

export const sendAndWaitMessageMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    _nodeQueueItems?: CanvasesCanvasNodeQueueItem[],
    _additionalData?: unknown,
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.component?.name || "unknown";

    return {
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      iconSrc: slackIcon,
      iconSlug: "slack",
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      eventSections: lastExecution ? sendAndWaitMessageEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: sendAndWaitMessageMetadataList(node),
      specs: sendAndWaitMessageSpecs(node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(
    execution: CanvasesCanvasNodeExecution,
    _node: ComponentsNode,
    _nodes?: ComponentsNode[],
  ): Record<string, any> {
    const outputs = execution.outputs as { received?: OutputPayload[]; timeout?: OutputPayload[] } | undefined;

    if (outputs?.received && outputs.received.length > 0) {
      const data = outputs.received[0].data as Record<string, unknown> | undefined;
      return {
        Status: "Received",
        Value: stringOrDash(data?.value),
        "Received At": execution.updatedAt ? new Date(execution.updatedAt).toLocaleString() : "-",
      };
    }

    if (outputs?.timeout && outputs.timeout.length > 0) {
      return {
        Status: "Timed Out",
        "Timed Out At": execution.updatedAt ? new Date(execution.updatedAt).toLocaleString() : "-",
      };
    }

    return {
      Status: "Waiting",
    };
  },
  subtitle(
    _node: ComponentsNode,
    execution: CanvasesCanvasNodeExecution,
    _additionalData?: unknown,
  ): string | React.ReactNode {
    if (!execution || !execution.createdAt) return "";
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function sendAndWaitMessageMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as SendAndWaitMessageMetadata | undefined;
  const configuration = node.configuration as SendAndWaitMessageConfiguration | undefined;

  const channelLabel = nodeMetadata?.channel?.name || configuration?.channel;
  if (channelLabel) {
    metadata.push({ icon: "hash", label: channelLabel });
  }

  if (configuration?.timeout) {
    metadata.push({ icon: "clock", label: `${configuration.timeout}s timeout` });
  }

  return metadata;
}

function sendAndWaitMessageSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as SendAndWaitMessageConfiguration | undefined;

  if (configuration?.message) {
    specs.push({
      title: "message",
      tooltipTitle: "message",
      iconSlug: "message-square",
      value: configuration.message,
      contentType: "text",
    });
  }

  if (configuration?.buttons && configuration.buttons.length > 0) {
    specs.push({
      title: "buttons",
      tooltipTitle: "buttons",
      iconSlug: "list",
      value: configuration.buttons.map((b) => b.name).join(", "),
      contentType: "text",
    });
  }

  return specs;
}

function sendAndWaitMessageEventSections(
  nodes: ComponentsNode[],
  execution: CanvasesCanvasNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const rootEvent = execution.rootEvent!;
  const triggerContext = (rootEvent as any).event ? rootEvent : { event: rootEvent };
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(triggerContext as any);

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution as any),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}
