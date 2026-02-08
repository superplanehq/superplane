import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsIcon from "@/assets/icons/integrations/aws.svg";
import { SnsTopicMessageEvent, SnsTriggerConfiguration, SnsTriggerMetadata } from "./types";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../../utils";

export const onTopicMessageTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as SnsTopicMessageEvent;
    const topicArn = extractTopicArn(eventData);
    const subject = eventData?.detail?.subject || eventData?.detail?.Subject;
    const topicName = topicArn ? topicArn.split(":").at(-1) : undefined;

    const title = topicName ? `${topicName}${subject ? ` • ${subject}` : ""}` : "SNS topic message";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as SnsTopicMessageEvent;
    const topicArn = extractTopicArn(eventData);
    const subject = eventData?.detail?.subject || eventData?.detail?.Subject;
    const message = eventData?.detail?.message || eventData?.detail?.Message;

    return {
      "Topic ARN": stringOrDash(topicArn),
      Subject: stringOrDash(subject),
      Message: stringOrDash(message),
      Region: stringOrDash(eventData?.region),
      Account: stringOrDash(eventData?.account),
      Source: stringOrDash(eventData?.source),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as SnsTriggerMetadata | undefined;
    const configuration = node.configuration as SnsTriggerConfiguration | undefined;
    const topicArn = metadata?.topicArn || configuration?.topicArn;
    const topicName = topicArn ? topicArn.split(":").at(-1) : undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsIcon,
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

function extractTopicArn(eventData: SnsTopicMessageEvent | undefined): string | undefined {
  const fromDetail = eventData?.detail?.topicArn || eventData?.detail?.TopicArn;
  if (fromDetail) {
    return fromDetail;
  }

  const firstResource = eventData?.resources?.[0];
  if (firstResource) {
    return firstResource;
  }

  return undefined;
}
