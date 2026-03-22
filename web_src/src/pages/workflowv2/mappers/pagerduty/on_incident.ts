import { getBackgroundColorClass } from "@/utils/colors";
import type React from "react";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import type { Agent, Incident } from "./types";
import { getDetailsForIncident } from "./base";

interface OnIncidentMetadata {
  service?: {
    id: string;
    name: string;
    html_url: string;
  };
}

interface OnIncidentEventData {
  agent?: Agent;
  incident?: Incident;
}

/**
 * Renderer for the "pagerduty.onIncident" trigger type
 */
export const onIncidentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data?.data as OnIncidentEventData;
    const incident = eventData?.incident;
    const contentParts = [incident?.urgency, incident?.status].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);

    return {
      title: `${incident?.id || ""} - ${incident?.title || ""}`,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data?.data as OnIncidentEventData;
    return getDetailsForIncident(eventData?.incident, eventData?.agent);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnIncidentMetadata;
    const configuration = node.configuration as any;
    const metadataItems = [];

    if (metadata?.service?.name) {
      metadataItems.push({
        icon: "bell",
        label: metadata.service.name,
      });
    }

    if (configuration.events) {
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${configuration.events.join(", ")}`,
      });
    }

    if (configuration.urgencies) {
      metadataItems.push({
        icon: "funnel",
        label: `Urgencies: ${configuration.urgencies.join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIncidentEventData;
      const incident = eventData?.incident;
      const contentParts = [incident?.urgency, incident?.status].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: `${incident?.id || ""} - ${incident?.title || ""}`,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildSubtitle(content: string, createdAt?: string): string | React.ReactNode {
  if (content && createdAt) {
    return renderWithTimeAgo(content, new Date(createdAt));
  }

  if (createdAt) {
    return renderTimeAgo(new Date(createdAt));
  }

  return content;
}
