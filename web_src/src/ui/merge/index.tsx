import { ComponentBase, type EventSection } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";
import { neutral } from "@/pages/workflowv2/mappers/eventSectionUtils";

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
}

export const MergeComponent: React.FC<MergeComponentProps> = ({
  title = "Merge",
  lastEvent,
  nextInQueue,
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
  if (nextInQueue) {
    eventSections.push(
      neutral({
        title: "Next In Queue",
        eventTitle: nextInQueue.title,
        handleComponent: nextInQueue.subtitle ? (
          <div className="mt-2 text-right text-xs text-gray-500">{nextInQueue.subtitle}</div>
        ) : undefined,
      }),
    );
  }

  return (
    <ComponentBase
      title={title}
      iconSlug="git-merge"
      headerColor="bg-blue-100"
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
