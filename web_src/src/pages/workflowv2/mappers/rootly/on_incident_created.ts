import { ComponentsNode, TriggersTrigger, CanvasesCanvasEvent } from "@/api-client";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { Incident } from "./types";
import { getDetailsForIncident } from "./base";

interface OnIncidentCreatedEventData {
  event?: string;
  incident?: Incident;
}

/**
 * Renderer for the "rootly.onIncidentCreated" trigger type
 */
export const onIncidentCreatedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: CanvasesCanvasEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnIncidentCreatedEventData;
    const incident = eventData?.incident;
    const contentParts = [incident?.severity?.name, incident?.status].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, event.createdAt);

    return {
      title: incident?.title || "Incident",
      subtitle,
    };
  },

  getRootEventValues: (lastEvent: CanvasesCanvasEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnIncidentCreatedEventData;
    return getDetailsForIncident(eventData?.incident!);
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: CanvasesCanvasEvent) => {
    const configuration = node.configuration as {
      severityFilter?: string[];
      serviceFilter?: string[];
      teamFilter?: string[];
    } | undefined;

    const metadataItems = [];
    if (configuration?.severityFilter?.length) {
      metadataItems.push({ icon: "funnel", label: `Severities: ${configuration.severityFilter.length}` });
    }
    if (configuration?.serviceFilter?.length) {
      metadataItems.push({ icon: "funnel", label: `Services: ${configuration.serviceFilter.length}` });
    }
    if (configuration?.teamFilter?.length) {
      metadataItems.push({ icon: "funnel", label: `Teams: ${configuration.teamFilter.length}` });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: rootlyIcon,
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnIncidentCreatedEventData;
      const incident = eventData?.incident;
      const contentParts = [incident?.severity?.name, incident?.status].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: incident?.title || "Incident",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
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
