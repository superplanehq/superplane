import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { buildSubtitle } from "../utils";
import posthogIcon from "@/assets/icons/integrations/posthog.svg";

interface OnEventConfiguration {
  projectId?: string;
  events?: string[];
  filterTestAccounts?: boolean;
}

interface OnEventEventData {
  event?: {
    event?: string;
    uuid?: string;
    distinct_id?: string;
    timestamp?: string;
    url?: string;
    properties?: Record<string, unknown>;
  };
  person?: {
    id?: string;
    url?: string;
    properties?: Record<string, unknown>;
  };
  project?: {
    id?: string;
    name?: string;
    url?: string;
  };
}

/**
 * The person's email or name is a far more useful subtitle than the distinct ID,
 * which is usually an opaque identifier, so prefer them when PostHog sent them.
 */
function getPersonLabel(eventData: OnEventEventData | undefined): string {
  const properties = eventData?.person?.properties;
  const email = typeof properties?.email === "string" ? properties.email : "";
  const name = typeof properties?.name === "string" ? properties.name : "";

  return email || name || eventData?.event?.distinct_id || "";
}

function getEventTitleAndSubtitle(
  eventData: OnEventEventData | undefined,
  createdAt?: string,
): { title: string; subtitle: string | React.ReactNode } {
  const title = eventData?.event?.event || "PostHog event";
  const subtitle = buildSubtitle(getPersonLabel(eventData), createdAt);

  return { title, subtitle };
}

export const onEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnEventEventData;
    return getEventTitleAndSubtitle(eventData, context.event?.createdAt);
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnEventEventData;
    const details: Record<string, string> = {};

    if (eventData?.event?.event) details["Event"] = eventData.event.event;
    if (eventData?.event?.distinct_id) details["Distinct ID"] = eventData.event.distinct_id;

    const person = getPersonLabel(eventData);
    if (person && person !== eventData?.event?.distinct_id) details["Person"] = person;

    if (eventData?.event?.timestamp) details["Timestamp"] = eventData.event.timestamp;
    if (eventData?.project?.name) details["Project"] = eventData.project.name;
    if (eventData?.event?.url) details["URL"] = eventData.event.url;

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnEventConfiguration;
    const metadataItems: { icon: string; label: string }[] = [];

    if (configuration?.projectId) {
      metadataItems.push({ icon: "folder", label: configuration.projectId });
    }

    metadataItems.push({
      icon: "funnel",
      label: configuration?.events?.length ? configuration.events.join(", ") : "All events",
    });

    if (configuration?.filterTestAccounts) {
      metadataItems.push({ icon: "user-minus", label: "Ignoring test accounts" });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: posthogIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnEventEventData;
      const { title, subtitle } = getEventTitleAndSubtitle(eventData, lastEvent.createdAt);

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
