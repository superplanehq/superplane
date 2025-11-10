import { ComponentBase, type EventSection } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";

export interface NoopProps extends ComponentActionsProps {
  title?: string;
  lastEvent?: Omit<EventSection, "title">;
  collapsed?: boolean;
  selected?: boolean;
  collapsedBackground?: string;
}

export const Noop: React.FC<NoopProps> = ({
  title = "Don't do anything",
  lastEvent,
  collapsed = false,
  selected = false,
  collapsedBackground,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onEdit,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  const eventSections: EventSection[] = [];
  if (lastEvent) {
    eventSections.push({
      title: "Last Event",
      ...lastEvent,
    });
  }

  return (
    <ComponentBase
      title={title}
      iconSlug="circle-off"
      headerColor="bg-gray-50"
      eventSections={eventSections}
      collapsed={collapsed}
      collapsedBackground={collapsedBackground}
      selected={selected}
      onRun={onRun}
      runDisabled={runDisabled}
      runDisabledTooltip={runDisabledTooltip}
      onEdit={onEdit}
      onDuplicate={onDuplicate}
      onDeactivate={onDeactivate}
      onToggleView={onToggleView}
      onDelete={onDelete}
      isCompactView={isCompactView}
    />
  );
};
