import { ComponentBase, type EventSection } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";

export interface MergeComponentProps extends ComponentActionsProps {
  title?: string;
  lastEvent?: Omit<EventSection, "title">;
  collapsed?: boolean;
  selected?: boolean;
  collapsedBackground?: string;
}

export const MergeComponent: React.FC<MergeComponentProps> = ({
  title = "Merge branches",
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
      iconSlug="git-merge"
      headerColor="bg-indigo-50"
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

export default MergeComponent;

