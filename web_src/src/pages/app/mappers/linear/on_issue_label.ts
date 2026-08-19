import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import linearIcon from "@/assets/icons/integrations/linear.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import { addTeamMetadata, getIssueLabel } from "./utils";
import type { LinearLabel, LinearNodeMetadata, LinearWebhookEvent, OnIssueLabelConfiguration } from "./types";

function labelNames(labels: LinearLabel[] | undefined): string {
  if (!labels || labels.length === 0) return "";
  return labels
    .map((label) => label.name)
    .filter(Boolean)
    .join(", ");
}

export const onIssueLabelTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const event = context.event?.data as LinearWebhookEvent | undefined;
    const issue = event?.data;

    return {
      title: getIssueLabel(issue) || "Issue",
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const event = (context.event?.data ?? {}) as LinearWebhookEvent;
    const issue = event.data;

    return {
      "Received At": context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-",
      Issue: stringOrDash(issue?.identifier),
      Title: stringOrDash(issue?.title),
      Labels: stringOrDash(labelNames(issue?.labels)),
      Status: stringOrDash(issue?.state?.name),
      "Issue URL": stringOrDash(event.url),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as LinearNodeMetadata | undefined;
    const configuration = node.configuration as OnIssueLabelConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    addTeamMetadata(metadataItems, metadata?.team, configuration?.team);

    if (configuration?.labels && configuration.labels.length > 0) {
      const count = configuration.labels.length;
      metadataItems.push({
        icon: "tag",
        label: `${count} ${count === 1 ? "label" : "labels"}`,
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
      const { title, subtitle } = onIssueLabelTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
