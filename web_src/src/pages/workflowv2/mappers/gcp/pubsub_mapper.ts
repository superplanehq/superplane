import {
  ComponentBaseMapper,
  ComponentBaseContext,
  EventStateRegistry,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps } from "@/ui/componentBase";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { formatTimeAgo } from "@/utils/date";
import gcpPubSubIcon from "@/assets/icons/integrations/gcp.pubsub.svg";
import { MetadataItem } from "@/ui/metadataList";

function pubsubProps(context: ComponentBaseContext): ComponentBaseProps {
  return {
    ...baseMapper.props(context),
    iconSrc: gcpPubSubIcon,
    metadata: pubsubMetadataList(context.node),
  };
}

function pubsubSubtitle(context: SubtitleContext): string {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
}

type PubSubOutputs<T> = { default?: Array<{ data?: T }> };

function formatLocalDateTime(value?: string): string | undefined {
  return value ? new Date(value).toLocaleString() : undefined;
}

function addCompletedAt(details: Record<string, string>, context: ExecutionDetailsContext): void {
  const completedAt = formatLocalDateTime(context.execution.updatedAt || context.execution.createdAt);
  if (completedAt) details["Completed At"] = completedAt;
}

export const publishMessageMapper: ComponentBaseMapper = {
  props: pubsubProps,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = context.execution.outputs as
      | PubSubOutputs<{ messageId?: string; topic?: string; topicId?: string; publishTime?: string }>
      | undefined;
    const item = payload?.default?.[0]?.data;
    const details: Record<string, string> = {};

    const publishedAt = formatLocalDateTime(
      item?.publishTime || context.execution.updatedAt || context.execution.createdAt,
    );
    if (publishedAt) details["Published At"] = publishedAt;
    const topic = item?.topic || item?.topicId;
    if (topic) details["Topic"] = topic;
    if (item?.messageId) details["Message ID"] = item.messageId;

    return details;
  },

  subtitle: pubsubSubtitle,
};

export const createTopicMapper: ComponentBaseMapper = {
  props: pubsubProps,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = context.execution.outputs as
      | PubSubOutputs<{ topic?: string; topicId?: string; name?: string }>
      | undefined;
    const item = payload?.default?.[0]?.data;
    const details: Record<string, string> = {};
    addCompletedAt(details, context);
    const topic = item?.topic || item?.topicId;
    if (topic) details["Topic"] = topic;
    return details;
  },

  subtitle: pubsubSubtitle,
};

export const deleteTopicMapper: ComponentBaseMapper = {
  props: pubsubProps,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = context.execution.outputs as PubSubOutputs<{ topic?: string; topicId?: string }> | undefined;
    const item = payload?.default?.[0]?.data;
    const details: Record<string, string> = {};
    addCompletedAt(details, context);
    const topic = item?.topic || item?.topicId;
    if (topic) details["Topic"] = topic;
    return details;
  },

  subtitle: pubsubSubtitle,
};

export const createSubscriptionMapper: ComponentBaseMapper = {
  props: pubsubProps,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = context.execution.outputs as
      | PubSubOutputs<{
          subscription?: string;
          subscriptionId?: string;
          topic?: string;
          topicId?: string;
          type?: string;
          name?: string;
        }>
      | undefined;
    const item = payload?.default?.[0]?.data;
    const details: Record<string, string> = {};
    addCompletedAt(details, context);
    const subscription = item?.subscription || item?.subscriptionId;
    const topic = item?.topic || item?.topicId;
    if (subscription) details["Subscription"] = subscription;
    if (topic) details["Topic"] = topic;
    if (item?.type) details["Type"] = formatSubscriptionType(item.type);
    return details;
  },

  subtitle: pubsubSubtitle,
};

export const deleteSubscriptionMapper: ComponentBaseMapper = {
  props: pubsubProps,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = context.execution.outputs as
      | PubSubOutputs<{ subscription?: string; subscriptionId?: string }>
      | undefined;
    const item = payload?.default?.[0]?.data;
    const details: Record<string, string> = {};
    addCompletedAt(details, context);
    const subscription = item?.subscription || item?.subscriptionId;
    if (subscription) details["Subscription"] = subscription;
    return details;
  },

  subtitle: pubsubSubtitle,
};

export const PUBSUB_ACTION_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("completed");

function formatSubscriptionType(value: string): string {
  switch (value.toUpperCase()) {
    case "PUSH":
      return "Push";
    case "PULL":
      return "Pull";
    default:
      return value;
  }
}

function pubsubMetadataList(node: NodeInfo): MetadataItem[] {
  const config = (node.configuration as Record<string, any> | undefined) ?? {};
  const metadata: MetadataItem[] = [];

  const topic = config.topic || config.topicId;
  if (topic) {
    metadata.push({ icon: "message-square", label: String(topic) });
  }

  const subscription = config.subscription || config.subscriptionId;
  if (subscription) {
    metadata.push({ icon: "radio", label: String(subscription) });
  }

  if (config.type) {
    metadata.push({ icon: "funnel", label: formatSubscriptionType(String(config.type)) });
  }

  if (config.pushEndpoint) {
    metadata.push({ icon: "link", label: compactURL(String(config.pushEndpoint)) });
  }

  return metadata;
}

function compactURL(value: string): string {
  if (value.length <= 64) {
    return value;
  }

  return `${value.slice(0, 64)}...`;
}
