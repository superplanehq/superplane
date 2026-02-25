import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import firehydrantIcon from "@/assets/icons/integrations/firehydrant.svg";
import { Incident } from "./types";
import { getDetailsForIncident } from "./base";

interface OnIncidentEventData {
  event?: string;
  incident?: Incident;
}

export const onIncidentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnIncidentEventData;
    const incident = eventData?.incident;
    const contentParts = [incident?.severity, incident?.current_milestone].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);

    return {
      title: incident?.name || "Incident",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnIncidentEventData;
    return getDetailsForIncident(eventData?.incident);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as { severities?: string[]; subscriptions?: string[] };
    const metadataItems = [];

    if (configuration?.subscriptions?.length) {
      const labels = configuration.subscriptions.map((s) => subscriptionLabel(s));
      metadataItems.push({
        icon: "rss",
        label: "Events: " + labels.join(", "),
      });
    }

    if (configuration?.severities?.length) {
      const formattedSeverities = configuration.severities.join(", ");
      metadataItems.push({
        icon: "funnel",
        label: "Severities: " + formattedSeverities,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: firehydrantIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIncidentEventData;
      const incident = eventData?.incident;
      const contentParts = [incident?.severity, incident?.current_milestone].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: incident?.name || "Incident",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildSubtitle(content: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  if (content && timeAgo) {
    return content + " · " + timeAgo;
  }

  return content || timeAgo;
}

const subscriptionLabels: Record<string, string> = {
  incidents: "Incidents",
  "incidents.private": "Private Incidents",
};

function subscriptionLabel(value: string): string {
  return subscriptionLabels[value] || value;
}
