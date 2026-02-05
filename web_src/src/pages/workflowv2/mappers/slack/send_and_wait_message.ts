import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
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
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: slackIcon,
      iconSlug: "slack",
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? sendAndWaitMessageEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: sendAndWaitMessageMetadataList(context.node),
      specs: sendAndWaitMessageSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const outputs = context.execution.outputs as { received?: OutputPayload[]; timeout?: OutputPayload[] } | undefined;

    if (outputs?.received && outputs.received.length > 0) {
      const data = outputs.received[0].data as Record<string, unknown> | undefined;
      return {
        Status: "Received",
        Value: stringOrDash(data?.value),
        "Received At": context.execution.updatedAt ? new Date(context.execution.updatedAt).toLocaleString() : "-",
      };
    }

    if (outputs?.timeout && outputs.timeout.length > 0) {
      return {
        Status: "Timed Out",
        "Timed Out At": context.execution.updatedAt ? new Date(context.execution.updatedAt).toLocaleString() : "-",
      };
    }

    return {
      Status: "Waiting",
    };
  },
  subtitle(context: SubtitleContext): string {
    if (!context.execution || !context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function sendAndWaitMessageMetadataList(node: NodeInfo): MetadataItem[] {
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

function sendAndWaitMessageSpecs(node: NodeInfo): ComponentBaseSpec[] {
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
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}
