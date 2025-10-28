import { ComponentBase, type EventSection } from "../componentBase";

export interface NoopProps {
  title?: string;
  lastEvent?: Omit<EventSection, "title">;
  collapsed?: boolean;
  selected?: boolean;
}

export const Noop: React.FC<NoopProps> = ({
  title = "Don't do anything",
  lastEvent,
  collapsed = false,
  selected = false,
}) => {
  const eventSections: EventSection[] = [];
  if (lastEvent) {
    eventSections.push({
      title: "Last Event",
      ...lastEvent
    });
  }

  return (
    <ComponentBase
      title={title}
      iconSlug="circle-off"
      headerColor="bg-gray-50"
      eventSections={eventSections}
      collapsed={collapsed}
      selected={selected}
    />
  );
};