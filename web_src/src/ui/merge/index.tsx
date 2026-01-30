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
  error?: string;
  warning?: string;
  paused?: boolean;
}

export const MergeComponent: React.FC<MergeComponentProps> = ({
  title = "Merge",
  lastEvent,
  collapsed = false,
  selected = false,
  collapsedBackground,
  eventStateMap,
  error,
  warning,
  paused,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onEdit,
  onDuplicate,
  onDeactivate,
  onTogglePause,
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

  return (
    <ComponentBase
      title={title}
      iconSlug="git-merge"
      eventSections={eventSections}
      collapsed={collapsed}
      collapsedBackground={collapsedBackground}
      selected={selected}
      eventStateMap={eventStateMap}
      error={error}
      warning={warning}
      paused={paused}
      onRun={onRun}
      runDisabled={runDisabled}
      runDisabledTooltip={runDisabledTooltip}
      onEdit={onEdit}
      onDuplicate={onDuplicate}
      onDeactivate={onDeactivate}
      onTogglePause={onTogglePause}
      onToggleView={onToggleView}
      onDelete={onDelete}
      isCompactView={isCompactView}
      includeEmptyState={!lastEvent}
    />
  );
};

export default MergeComponent;
