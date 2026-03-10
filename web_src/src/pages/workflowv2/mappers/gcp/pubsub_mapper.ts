import { ComponentBaseMapper, ComponentBaseContext, EventStateRegistry, ExecutionDetailsContext, SubtitleContext } from "../types";
import { ComponentBaseProps } from "@/ui/componentBase";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { formatTimeAgo } from "@/utils/date";
import gcpPubSubIcon from "@/assets/icons/integrations/gcp.pubsub.svg";

function pubsubProps(context: ComponentBaseContext): ComponentBaseProps {
  return { ...baseMapper.props(context), iconSrc: gcpPubSubIcon };
}

function pubsubSubtitle(context: SubtitleContext): string {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
}

type PubSubOutputs<T> = { default?: Array<{ data?: T }> };

export const publishMessageMapper: ComponentBaseMapper = {
  props: pubsubProps,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = context.execution.outputs as PubSubOutputs<{ messageId?: string; topicId?: string }> | undefined;
    const item = payload?.default?.[0]?.data;
    const details: Record<string, string> = {};
    if (item?.topicId) details["Topic"] = item.topicId;
    if (item?.messageId) details["Message ID"] = item.messageId;
    if (context.execution.updatedAt) {
      details["Published At"] = new Date(context.execution.updatedAt).toLocaleString();
    }
    return details;
  },

  subtitle: pubsubSubtitle,
};

export const createTopicMapper: ComponentBaseMapper = {
  props: pubsubProps,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = context.execution.outputs as PubSubOutputs<{ topicId?: string; name?: string }> | undefined;
    const item = payload?.default?.[0]?.data;
    const details: Record<string, string> = {};
    if (item?.topicId) details["Topic ID"] = item.topicId;
    if (item?.name) details["Resource Name"] = item.name;
    return details;
  },

  subtitle: pubsubSubtitle,
};

export const deleteTopicMapper: ComponentBaseMapper = {
  props: pubsubProps,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = context.execution.outputs as PubSubOutputs<{ topicId?: string }> | undefined;
    const item = payload?.default?.[0]?.data;
    const details: Record<string, string> = {};
    if (item?.topicId) details["Topic ID"] = item.topicId;
    return details;
  },

  subtitle: pubsubSubtitle,
};

export const createSubscriptionMapper: ComponentBaseMapper = {
  props: pubsubProps,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = context.execution.outputs as PubSubOutputs<{
      subscriptionId?: string; topicId?: string; type?: string; name?: string
    }> | undefined;
    const item = payload?.default?.[0]?.data;
    const details: Record<string, string> = {};
    if (item?.subscriptionId) details["Subscription ID"] = item.subscriptionId;
    if (item?.topicId) details["Topic"] = item.topicId;
    if (item?.type) details["Type"] = item.type;
    if (item?.name) details["Resource Name"] = item.name;
    return details;
  },

  subtitle: pubsubSubtitle,
};

export const deleteSubscriptionMapper: ComponentBaseMapper = {
  props: pubsubProps,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = context.execution.outputs as PubSubOutputs<{ subscriptionId?: string }> | undefined;
    const item = payload?.default?.[0]?.data;
    const details: Record<string, string> = {};
    if (item?.subscriptionId) details["Subscription ID"] = item.subscriptionId;
    return details;
  },

  subtitle: pubsubSubtitle,
};

export const PUBSUB_ACTION_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("completed");
