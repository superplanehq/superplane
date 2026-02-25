import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import newrelicIcon from "@/assets/icons/integrations/newrelic.svg";

interface OnIssueEventData {
  issueId?: string;
  title?: string;
  priority?: string;
  state?: string;
  issueUrl?: string;
  owner?: string;
}

/**
 * Renderer for the "newrelic.onIssue" trigger type
 */
export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data?.data as OnIssueEventData;
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    const contentParts = [eventData?.priority, eventData?.state].filter(Boolean).join(" · ");
    const subtitle = [contentParts, timeAgo].filter(Boolean).join(" · ");

    return {
      title: eventData?.title || eventData?.issueId || "New Relic Issue",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data?.data as OnIssueEventData;
    const values: Record<string, string> = {};
    if (eventData?.issueId) values["Issue ID"] = eventData.issueId;
    if (eventData?.title) values["Title"] = eventData.title;
    if (eventData?.priority) values["Priority"] = eventData.priority;
    if (eventData?.state) values["State"] = eventData.state;
    if (eventData?.owner) values["Owner"] = eventData.owner;
    if (eventData?.issueUrl) values["Issue URL"] = eventData.issueUrl;
    return values;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as any;
    const metadataItems = [];

    if (configuration?.priorities?.length) {
      metadataItems.push({
        icon: "funnel",
        label: `Priorities: ${configuration.priorities.join(", ")}`,
      });
    }

    if (configuration?.states?.length) {
      metadataItems.push({
        icon: "funnel",
        label: `States: ${configuration.states.join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: newrelicIcon,
      iconSlug: "newrelic",
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIssueEventData;
      const timeAgo = formatTimeAgo(new Date(lastEvent.createdAt));
      const contentParts = [eventData?.priority, eventData?.state].filter(Boolean).join(" · ");
      const subtitle = [contentParts, timeAgo].filter(Boolean).join(" · ");

      props.lastEventData = {
        title: eventData?.title || eventData?.issueId || "New Relic Issue",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
