import { getBackgroundColorClass } from "@/lib/colors";
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

interface OnIncidentConfiguration {
  events?: string[];
  urgencies?: string[];
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

    return {
      title: incidentTitle(incident),
      subtitle: buildSubtitle(incidentContent(incident), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data?.data as OnIncidentEventData;
    return getDetailsForIncident(eventData?.incident, eventData?.agent);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnIncidentMetadata;
    const configuration = node.configuration as OnIncidentConfiguration | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: incidentMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      props.lastEventData = lastIncidentEventData(lastEvent.data as OnIncidentEventData, lastEvent);
    }

    return props;
  },
};

function incidentTitle(incident?: Incident): string {
  return `${incident?.id || ""} - ${incident?.title || ""}`;
}

function incidentContent(incident?: Incident): string {
  return [incident?.urgency, incident?.status].filter(Boolean).join(" · ");
}

function incidentMetadataItems(metadata?: OnIncidentMetadata, configuration?: OnIncidentConfiguration) {
  const metadataItems = [];

  if (metadata?.service?.name) {
    metadataItems.push({
      icon: "bell",
      label: metadata.service.name,
    });
  }

  if (configuration?.events) {
    metadataItems.push({
      icon: "funnel",
      label: `Events: ${configuration.events.join(", ")}`,
    });
  }

  if (configuration?.urgencies) {
    metadataItems.push({
      icon: "funnel",
      label: `Urgencies: ${configuration.urgencies.join(", ")}`,
    });
  }

  return metadataItems;
}

function lastIncidentEventData(
  eventData: OnIncidentEventData | undefined,
  lastEvent: { createdAt: string; id: string },
) {
  const incident = eventData?.incident;

  return {
    title: incidentTitle(incident),
    subtitle: buildSubtitle(incidentContent(incident), lastEvent.createdAt),
    receivedAt: new Date(lastEvent.createdAt),
    state: "triggered",
    eventId: lastEvent.id,
  };
}

function buildSubtitle(content: string, createdAt?: string): string | React.ReactNode {
  if (content && createdAt) {
    return renderWithTimeAgo(content, new Date(createdAt));
  }

  if (createdAt) {
    return renderTimeAgo(new Date(createdAt));
  }

  return content;
}
