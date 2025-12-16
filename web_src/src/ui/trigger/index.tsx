import React from "react";
import { ComponentBase, type ComponentBaseProps, type EventSection } from "../componentBase";
import { type MetadataItem } from "../metadataList";

type LastEventState = string;

interface TriggerLastEventData {
  title: string;
  subtitle?: string;
  receivedAt: Date;
  state: LastEventState;
  eventId?: string;
}

export interface TriggerProps extends Omit<ComponentBaseProps, "eventSections"> {
  metadata: MetadataItem[];
  lastEventData?: TriggerLastEventData;
}

export const Trigger: React.FC<TriggerProps> = ({ lastEventData, ...componentBaseProps }) => {
  const eventSections: EventSection[] = React.useMemo(() => {
    if (!lastEventData) return [];

    return [
      {
        receivedAt: lastEventData.receivedAt,
        eventState: lastEventData.state,
        eventTitle: lastEventData.title,
        eventSubtitle: lastEventData.subtitle,
        eventId: lastEventData.eventId,
      },
    ];
  }, [lastEventData]);

  if (!lastEventData) {
    return <ComponentBase {...componentBaseProps} includeEmptyState emptyStateProps={{ title: "Waiting for the first event" }} />;
  }

  return <ComponentBase {...componentBaseProps} eventSections={eventSections} />;
};
