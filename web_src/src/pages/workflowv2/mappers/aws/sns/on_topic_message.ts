import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsSnsIcon from "@/assets/icons/integrations/aws.sns.svg";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../../utils";

interface OnTopicMessageConfiguration {
  region?: string;
  topicArn?: string;
}

interface OnTopicMessageMetadata {
  region?: string;
  topicArn?: string;
}

interface TopicMessageEvent {
  Type?: string;
  Message?: string;
  MessageId?: string;
  TopicArn?: string;
  Subject?: string;
  Timestamp?: string;
  SignatureVersion?: string;
  Signature?: string;
  SigningCertURL?: string;
  UnsubscribeURL?: string;
  SubscribeURL?: string;
  Token?: string;
  MessageAttributes?: Record<string, { Type: string; Value: string }>;
}

export const onTopicMessageTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as TopicMessageEvent;
    const title = eventData?.MessageId ? eventData.MessageId : "SNS topic message";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as TopicMessageEvent;

    return {
      "Message ID": stringOrDash(eventData?.MessageId),
      Message: stringOrDash(eventData?.Message),
      "Topic ARN": stringOrDash(eventData?.TopicArn),
      Timestamp: stringOrDash(eventData?.Timestamp),
      Subject: stringOrDash(eventData?.Subject),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnTopicMessageMetadata | undefined;
    const configuration = node.configuration as OnTopicMessageConfiguration | undefined;
    const topicArn = metadata?.topicArn || configuration?.topicArn;
    const topicName = topicArn ? topicArn.split(":").at(-1) : undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsSnsIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: topicName ? [{ icon: "hash", label: topicName }] : [],
    };

    if (lastEvent) {
      const { title, subtitle } = onTopicMessageTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
