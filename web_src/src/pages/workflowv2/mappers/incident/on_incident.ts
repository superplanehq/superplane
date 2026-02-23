import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import incidentIcon from "@/assets/icons/integrations/incident.svg";
import { Incident } from "./types";
import { getDetailsForIncident } from "./base";

const eventLabels: Record<string, string> = {
  "public_incident.incident_created_v2": "Incident created",
  "public_incident.incident_updated_v2": "Incident updated",
  "incident.created": "Incident created",
  "incident.updated": "Incident updated",
};

function formatEventLabel(event: string): string {
  return eventLabels[event] ?? event;
}

interface OnIncidentEventData {
  event_type?: string;
  incident?: Incident;
}

export const onIncidentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnIncidentEventData;
    const incident = eventData?.incident;
    const severityName = incident?.severity?.name;
    const statusName = incident?.incident_status?.name;
    const contentParts = [severityName, statusName].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);
    return { title: incident?.name || "Incident", subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnIncidentEventData;
    return getDetailsForIncident(eventData?.incident);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as { events?: string[] };
    const metadataItems: { icon: string; label: string }[] = [];
    if (configuration?.events?.length) {
      const formattedEvents = configuration.events.map(formatEventLabel).join(", ");
      metadataItems.push({ icon: "funnel", label: "Events: " + formattedEvents });
    }
    const props: TriggerProps = {
      title: node.name!,
      iconSrc: incidentIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };
    if (lastEvent) {
      const eventData = lastEvent.data as OnIncidentEventData;
      const incident = eventData?.incident;
      const contentParts = [incident?.severity?.name, incident?.incident_status?.name].filter(Boolean).join(" · ");
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
  return content && timeAgo ? content + " · " + timeAgo : content || timeAgo;
}
