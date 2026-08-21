import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import linearIcon from "@/assets/icons/integrations/linear.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import { addTeamMetadata, getIssueLabel } from "./utils";
import type { LinearAttachmentWebhookEvent, LinearNodeMetadata, OnIssueAttachmentConfiguration } from "./types";

/** Webhook action values Linear sends for attachment events. */
const ACTION_LABELS: Record<string, string> = {
  create: "Added",
  update: "Updated",
  remove: "Removed",
};

function actionLabel(action: string | undefined): string | undefined {
  if (!action) return undefined;
  return ACTION_LABELS[action] ?? action;
}

export const onIssueAttachmentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const event = context.event?.data as LinearAttachmentWebhookEvent | undefined;
    const attachment = event?.data;

    return {
      title: attachment?.title || "Attachment",
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const event = (context.event?.data ?? {}) as LinearAttachmentWebhookEvent;
    const attachment = event.data;

    return {
      "Received At": context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-",
      Issue: stringOrDash(getIssueLabel(attachment?.issue)),
      Title: stringOrDash(attachment?.title),
      Subtitle: stringOrDash(attachment?.subtitle),
      Action: stringOrDash(actionLabel(event.action)),
      "Attachment URL": stringOrDash(attachment?.url),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as LinearNodeMetadata | undefined;
    const configuration = node.configuration as OnIssueAttachmentConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    addTeamMetadata(metadataItems, metadata?.team, configuration?.team);

    if (configuration?.actions && configuration.actions.length > 0) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.actions.map((action) => actionLabel(action)).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: linearIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onIssueAttachmentTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
