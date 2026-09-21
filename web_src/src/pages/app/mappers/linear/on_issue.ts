import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import linearIcon from "@/assets/icons/integrations/linear.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../eventDisplay";
import { formatPredicate } from "../eventDisplay";
import { addTeamMetadata, getIssueLabel } from "./utils";
import type { LinearNodeMetadata, LinearWebhookEvent, OnIssueConfiguration } from "./types";

/** Webhook action values Linear sends for issue events. */
const ACTION_LABELS: Record<string, string> = {
  create: "Created",
  update: "Updated",
  remove: "Deleted",
};

function actionLabel(action: string | undefined): string | undefined {
  if (!action) return undefined;
  return ACTION_LABELS[action] ?? action;
}

export const onIssueTriggerRenderer: TriggerRenderer = {
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
      Action: stringOrDash(actionLabel(event.action)),
      Status: stringOrDash(issue?.state?.name),
      "Issue URL": stringOrDash(event.url),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as LinearNodeMetadata | undefined;
    const configuration = node.configuration as OnIssueConfiguration | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: linearIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: issueMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const { title, subtitle } = onIssueTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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

function issueMetadataItems(metadata?: LinearNodeMetadata, configuration?: OnIssueConfiguration): MetadataItem[] {
  const metadataItems: MetadataItem[] = [];

  addTeamMetadata(metadataItems, metadata?.team, configuration?.team);
  appendActionMetadata(metadataItems, configuration?.actions);
  appendLabelMetadata(metadataItems, configuration?.labels);

  return metadataItems;
}

function appendActionMetadata(metadataItems: MetadataItem[], actions?: string[]): void {
  if (!actions || actions.length === 0) {
    return;
  }

  metadataItems.push({
    icon: "funnel",
    label: actions.map((action) => actionLabel(action)).join(", "),
  });
}

function appendLabelMetadata(metadataItems: MetadataItem[], labels?: OnIssueConfiguration["labels"]): void {
  if (!labels || labels.length === 0) {
    return;
  }

  metadataItems.push({
    icon: "tag",
    label: labels.map((label) => formatPredicate(label)).join(", "),
  });
}
