import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import linearIcon from "@/assets/icons/integrations/linear.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import { addTeamMetadata, getIssueLabel, getUserLabel } from "./utils";
import type { LinearCommentWebhookEvent, LinearNodeMetadata, OnIssueCommentConfiguration } from "./types";

/** Webhook action values Linear sends for comment events. */
const ACTION_LABELS: Record<string, string> = {
  create: "Created",
  update: "Updated",
  remove: "Deleted",
};

function actionLabel(action: string | undefined): string | undefined {
  if (!action) return undefined;
  return ACTION_LABELS[action] ?? action;
}

export const onIssueCommentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const event = context.event?.data as LinearCommentWebhookEvent | undefined;
    const comment = event?.data;

    return {
      title: getIssueLabel(comment?.issue) || "Comment",
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const event = (context.event?.data ?? {}) as LinearCommentWebhookEvent;
    const comment = event.data;

    return {
      "Received At": context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-",
      Issue: stringOrDash(getIssueLabel(comment?.issue)),
      Author: stringOrDash(getUserLabel(comment?.user)),
      Comment: stringOrDash(comment?.body),
      Action: stringOrDash(actionLabel(event.action)),
      "Comment URL": stringOrDash(event.url || comment?.url),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as LinearNodeMetadata | undefined;
    const configuration = node.configuration as OnIssueCommentConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    addTeamMetadata(metadataItems, metadata?.team, configuration?.team);

    if (configuration?.actions && configuration.actions.length > 0) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.actions.map((action) => actionLabel(action)).join(", "),
      });
    }

    if (configuration?.contentFilter) {
      metadataItems.push({
        icon: "funnel",
        label: `Filter: ${configuration.contentFilter}`,
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
      const { title, subtitle } = onIssueCommentTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
