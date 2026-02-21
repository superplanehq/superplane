import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsSqsIcon from "@/assets/icons/integrations/aws.sqs.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { getQueueNameFromUrl } from "./utils";

interface SendMessageConfiguration {
  region?: string;
  queue?: string;
  format?: string;
  json?: any;
  xml?: string;
  text?: string;
}

interface SendMessageOutput {
  queueUrl?: string;
  messageId?: string;
}

export const sendMessageMapper: ComponentBaseMapper = {
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
      eventSections: lastExecution ? sendMessageEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: sendMessageMetadataList(context.node),
      specs: sendMessageSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as SendMessageOutput | undefined;

    if (!result) {
      return {};
    }

    return {
      "Queue URL": stringOrDash(result.queueUrl),
      "Message ID": stringOrDash(result.messageId),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function sendMessageMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as SendMessageConfiguration | undefined;

  const queueName = getQueueNameFromUrl(configuration?.queue);
  if (queueName) {
    metadata.push({ icon: "hash", label: queueName });
  }

  if (configuration?.format) {
    metadata.push({ icon: "code", label: `Message format: ${configuration.format}` });
  }

  return metadata;
}

function sendMessageEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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

function sendMessageSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as SendMessageConfiguration | undefined;

  if (configuration?.json) {
    specs.push({
      title: "JSON message",
      tooltipTitle: "JSON message",
      iconSlug: "file-json",
      value: configuration.json,
      contentType: "json",
    });
  }

  if (configuration?.xml) {
    specs.push({
      title: "XML message",
      tooltipTitle: "XML message",
      iconSlug: "file-code",
      value: configuration.xml,
      contentType: "xml",
    });
  }

  if (configuration?.text) {
    specs.push({
      title: "Text message",
      tooltipTitle: "Text message",
      iconSlug: "file-text",
      value: configuration.text,
      contentType: "text",
    });
  }

  return specs;
}
