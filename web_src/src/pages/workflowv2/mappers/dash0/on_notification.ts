import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { stringOrDash } from "../utils";

interface Dash0NotificationIssue {
  id?: string;
  issueIdentifier?: string;
  status?: string;
  summary?: string;
  start?: string;
  end?: string;
  url?: string;
  dataset?: string;
  description?: string;
  labels?: IssueLabel[];
}

interface IssueLabel {
  key?: string;
  value?: IssueLabelValue;
}

interface IssueLabelValue {
  stringValue?: string;
}

interface Dash0NotificationEventData {
  issue?: Dash0NotificationIssue;
}

interface OnNotificationConfiguration {
  statuses?: string[];
}

export const onNotificationTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as Dash0NotificationEventData | undefined;
    const issue = eventData?.issue;
    const title = issue?.summary || issue?.issueIdentifier || issue?.id || "Dash0 notification";
    const subtitleParts = [issue?.status].filter(Boolean).join(" · ");
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    const subtitle = subtitleParts && timeAgo ? `${subtitleParts} · ${timeAgo}` : subtitleParts || timeAgo;

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as Dash0NotificationEventData | undefined;

    return {
      "Issue ID": stringOrDash(eventData?.issue?.id),
      "Issue Identifier": stringOrDash(eventData?.issue?.issueIdentifier),
      URL: stringOrDash(eventData?.issue?.url),
      Status: stringOrDash(eventData?.issue?.status),
      Summary: stringOrDash(eventData?.issue?.summary),
      Dataset: stringOrDash(eventData?.issue?.dataset),
      Start: stringOrDash(eventData?.issue?.start),
      Labels: stringOrDash(
        eventData?.issue?.labels?.map((label) => `${label.key}: ${label.value?.stringValue}`).join(", "),
      ),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnNotificationConfiguration | undefined;
    const metadataItems = [];

    if (configuration?.statuses?.length) {
      metadataItems.push({
        icon: "funnel",
        label: `Statuses: ${configuration.statuses.join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: dash0Icon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onNotificationTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
