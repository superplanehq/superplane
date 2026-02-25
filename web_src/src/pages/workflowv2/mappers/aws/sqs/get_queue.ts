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
import awsSqsIcon from "@/assets/icons/integrations/aws.sqs.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { getQueueNameFromUrl } from "./utils";

interface GetQueueConfiguration {
  region?: string;
  queue?: string;
}

interface GetQueueOutput {
  queueUrl?: string;
  attributes?: Record<string, string>;
}

export const getQueueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: awsSqsIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getQueueEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getQueueMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as GetQueueOutput | undefined;

    if (!result) {
      return {};
    }

    const details: Record<string, string> = {
      "Queue URL": stringOrDash(result.queueUrl),
    };

    if (result.attributes) {
      const attrs = result.attributes;
      if (attrs.QueueArn) {
        details["Queue ARN"] = stringOrDash(attrs.QueueArn);
      }
      if (attrs.VisibilityTimeout) {
        details["Visibility Timeout"] = `${attrs.VisibilityTimeout} seconds`;
      }
      if (attrs.MessageRetentionPeriod) {
        details["Message Retention Period"] = `${attrs.MessageRetentionPeriod} seconds`;
      }
      if (attrs.MaximumMessageSize) {
        details["Maximum Message Size"] = `${attrs.MaximumMessageSize} bytes`;
      }
      if (attrs.DelaySeconds) {
        details["Delay Seconds"] = `${attrs.DelaySeconds} seconds`;
      }
      if (attrs.ReceiveMessageWaitTimeSeconds) {
        details["Receive Message Wait Time"] = `${attrs.ReceiveMessageWaitTimeSeconds} seconds`;
      }
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

function getQueueMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetQueueConfiguration | undefined;

  const queueName = getQueueNameFromUrl(configuration?.queue);
  if (queueName) {
    metadata.push({ icon: "hash", label: queueName });
  }

  return metadata;
}

function getQueueEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
