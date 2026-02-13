import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsSnsIcon from "@/assets/icons/integrations/aws.sns.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";

interface SnsNodeConfiguration {
  topicArn?: string;
  subscriptionArn?: string;
  name?: string;
  protocol?: string;
  endpoint?: string;
}

function buildSnsComponentMapper(): ComponentBaseMapper {
  return {
    props(context: ComponentBaseContext): ComponentBaseProps {
      const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
      const componentName = context.componentDefinition.name || "unknown";

      return {
        title: context.node.name || context.componentDefinition.label || "Unnamed component",
        iconSrc: awsSnsIcon,
        iconColor: getColorClass(context.componentDefinition.color),
        collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
        collapsed: context.node.isCollapsed,
        eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
        includeEmptyState: !lastExecution,
        metadata: buildMetadata(context.node),
        eventStateMap: getStateMap(componentName),
      };
    },

    getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
      const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
      const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
      if (!result || typeof result !== "object") {
        return {};
      }

      const details: Record<string, string> = {};
      for (const [key, value] of Object.entries(result)) {
        if (value === undefined || value === null) {
          continue;
        }

        if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
          details[toLabel(key)] = String(value);
          continue;
        }

        if (Array.isArray(value)) {
          details[toLabel(key)] = value.map((entry) => String(entry)).join(", ");
          continue;
        }

        details[toLabel(key)] = JSON.stringify(value);
      }

      return details;
    },

    subtitle(context: SubtitleContext): string {
      if (!context.execution.createdAt) {
        return "";
      }
      return formatTimeAgo(new Date(context.execution.createdAt));
    },
  };
}

function buildMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = (node.configuration || {}) as SnsNodeConfiguration;

  if (configuration.topicArn) {
    const topicName = configuration.topicArn.split(":").at(-1);
    metadata.push({ icon: "hash", label: stringOrDash(topicName) });
  }

  if (configuration.subscriptionArn) {
    const subscriptionID = configuration.subscriptionArn.split(":").at(-1);
    metadata.push({ icon: "link", label: stringOrDash(subscriptionID) });
  }

  if (configuration.name) {
    metadata.push({ icon: "tag", label: configuration.name });
  }

  if (configuration.protocol) {
    metadata.push({ icon: "network", label: configuration.protocol });
  }

  if (configuration.endpoint) {
    metadata.push({ icon: "globe", label: configuration.endpoint });
  }

  return metadata.slice(0, 2);
}

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.createdAt || !execution.rootEvent?.id) {
    return [];
  }

  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}

function toLabel(key: string): string {
  return key
    .replace(/([A-Z])/g, " $1")
    .replace(/^./, (char) => char.toUpperCase())
    .trim();
}

export const getTopicMapper = buildSnsComponentMapper();
export const getSubscriptionMapper = buildSnsComponentMapper();
export const createTopicMapper = buildSnsComponentMapper();
export const deleteTopicMapper = buildSnsComponentMapper();
export const publishMessageMapper = buildSnsComponentMapper();
