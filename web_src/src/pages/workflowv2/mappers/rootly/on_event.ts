import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { Incident, IncidentEvent } from "./types";
import { getDetailsForIncident, getDetailsForIncidentEvent } from "./base";

interface OnEventConfiguration {
  incidentStatuses?: string[];
  severities?: string[];
  services?: string[];
  teams?: string[];
  eventSources?: string[];
  visibilities?: string[];
  eventKinds?: string[];
}

interface OnEventEventData extends IncidentEvent {
  incident?: Incident;
}

/**
 * Renderer for the "rootly.onEvent" trigger type
 */
export const onEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnEventEventData;
    const incident = eventData?.incident;
    const contentParts = [];

    if (eventData?.kind) {
      contentParts.push(eventData.kind);
    }

    if (eventData?.user_display_name) {
      contentParts.push(`by ${eventData.user_display_name}`);
    }

    if (eventData?.event) {
      contentParts.push(eventData.event);
    }

    const subtitle = buildSubtitle(contentParts.filter(Boolean).join(" · "), context.event?.createdAt);

    return {
      title: incident?.title || "Incident Event",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnEventEventData;
    const details = getDetailsForIncidentEvent(eventData);
    if (eventData?.incident) {
      Object.assign(details, getDetailsForIncident(eventData.incident));
    }
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnEventConfiguration;
    const metadataItems: Array<{ icon: string; label: string }> = [];

    addMetadata(metadataItems, "Incident Status", configuration?.incidentStatuses);
    addMetadata(metadataItems, "Severity", configuration?.severities);
    addMetadata(metadataItems, "Service", configuration?.services);
    addMetadata(metadataItems, "Team", configuration?.teams);
    addMetadata(metadataItems, "Event Source", configuration?.eventSources);
    addMetadata(metadataItems, "Visibility", configuration?.visibilities);
    addMetadata(metadataItems, "Event Kind", configuration?.eventKinds);

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: rootlyIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnEventEventData;
      const incident = eventData?.incident;
      const contentParts = [];

      if (eventData?.kind) {
        contentParts.push(eventData.kind);
      }

      if (eventData?.user_display_name) {
        contentParts.push(`by ${eventData.user_display_name}`);
      }

      if (eventData?.event) {
        contentParts.push(eventData.event);
      }

      const subtitle = buildSubtitle(contentParts.filter(Boolean).join(" · "), lastEvent.createdAt);

      props.lastEventData = {
        title: incident?.title || "Incident Event",
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
    return `${content} · ${timeAgo}`;
  }

  return content || timeAgo;
}

function addMetadata(metadataItems: Array<{ icon: string; label: string }>, label: string, values?: string[]): void {
  if (!values || values.length === 0) {
    return;
  }

  metadataItems.push({
    icon: "funnel",
    label: `${label}: ${values.join(", ")}`,
  });
}
