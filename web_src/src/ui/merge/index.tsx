import { ComponentBase, type EventSection, type EventStateMap } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";

export interface MergeComponentProps extends ComponentActionsProps {
  title?: string;
  lastEvent?: Omit<EventSection, "title">;
  // Show the next queued item for this merge node
  nextInQueue?: {
    title: string;
    subtitle?: string;
  };
  collapsed?: boolean;
  selected?: boolean;
  collapsedBackground?: string;
  eventStateMap?: EventStateMap;
}

export const MergeComponent: React.FC<MergeComponentProps> = ({
  title = "Merge",
  lastEvent,
  nextInQueue,
  collapsed = false,
  selected = false,
  collapsedBackground,
  eventStateMap,
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
      ...lastEvent,
    });
  }
  if (nextInQueue) {
    eventSections.push({
      eventTitle: nextInQueue.title,
      eventState: "queued",
      handleComponent: nextInQueue.subtitle ? (
        <div className="mt-2 text-right text-xs text-gray-500">{nextInQueue.subtitle}</div>
      ) : undefined,
    });
  }

  return (
    <ComponentBase
      title={title}
      iconSlug="git-merge"
      eventSections={eventSections}
      collapsed={collapsed}
      collapsedBackground={collapsedBackground}
      selected={selected}
      eventStateMap={eventStateMap}
      onRun={onRun}
      runDisabled={runDisabled}
      runDisabledTooltip={runDisabledTooltip}
      onEdit={onEdit}
      onDuplicate={onDuplicate}
      onDeactivate={onDeactivate}
      onToggleView={onToggleView}
      onDelete={onDelete}
      isCompactView={isCompactView}
      includeEmptyState={!lastEvent}
    />
  );
};

export default MergeComponent;
