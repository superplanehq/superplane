import React from "react";
import { ComponentBase, type EventSection, type EventState } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";
import { type MetadataItem } from "../metadataList";
import { type ChildEventsInfo } from "../childEvents";

export type LastRunState = "success" | "failed" | "running";
export type ChildEventsState = "processed" | "discarded" | "waiting" | "running" | string;

// Map LastRunState to EventState
const mapLastRunStateToEventState = (state: LastRunState): EventState => {
  switch (state) {
    case "success":
      return "success";
    case "failed":
      return "failed";
    case "running":
      return "running";
    default:
      return "neutral";
  }
};

export interface WaitingInfo {
  icon: string;
  info: string;
  futureTimeDate: Date;
}

export interface QueueItem {
  title: string;
  subtitle: string;
  receivedAt: Date;
}

export interface LastRunItem extends QueueItem {
  childEventsInfo?: ChildEventsInfo;
  state: LastRunState;
  values: Record<string, string>;
  id?: string;
}

export interface ParameterGroup {
  icon: string;
  items: Record<string, string>;
}

export interface CompositeProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  headerColor: string;
  title: string;
  metadata?: MetadataItem[];
  parameters?: ParameterGroup[];
  lastRunItem?: LastRunItem;
  lastRunItems?: LastRunItem[];
  maxVisibleEvents?: number;
  nextInQueue?: QueueItem;
  collapsedBackground?: string;
  collapsed?: boolean;
  selected?: boolean;
  isMissing?: boolean;

  onExpandChildEvents?: () => void;
  onReRunChildEvents?: () => void;
  onToggleCollapse?: () => void;
  onViewMoreEvents?: () => void;
}

export const Composite: React.FC<CompositeProps> = ({
  iconSrc,
  iconSlug,
  iconColor,
  iconBackground,
  headerColor,
  title,
  metadata,
  parameters = [],
  lastRunItem,
  lastRunItems,
  maxVisibleEvents = 5,
  nextInQueue,
  collapsed = false,
  collapsedBackground,
  onExpandChildEvents,
  onReRunChildEvents,
  onToggleCollapse,
  onViewMoreEvents,
  selected = false,
  isMissing = false,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onEdit,
  onConfigure,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  // Use lastRunItems if provided, otherwise fall back to single lastRunItem
  const eventsToDisplay = React.useMemo(() => {
    if (lastRunItems && lastRunItems.length > 0) {
      return lastRunItems;
    } else if (lastRunItem) {
      return [lastRunItem];
    }
    return [];
  }, [lastRunItem, lastRunItems]);

  const visibleEvents = React.useMemo(() => {
    return eventsToDisplay.slice(0, maxVisibleEvents);
  }, [eventsToDisplay, maxVisibleEvents]);

  const hiddenEventsCount = React.useMemo(() => {
    return Math.max(0, eventsToDisplay.length - maxVisibleEvents);
  }, [eventsToDisplay.length, maxVisibleEvents]);

  // Convert events to EventSection format for ComponentBase
  const eventSections: EventSection[] = React.useMemo(() => {
    const sections: EventSection[] = [];

    // Add visible events
    visibleEvents.forEach((event) => {
      sections.push({
        eventId: event.id?.slice(0, 4),
        eventState: mapLastRunStateToEventState(event.state),
        eventTitle: event.title,
        eventSubtitle: event.subtitle,
        receivedAt: event.receivedAt,
        showAutomaticTime: true,
        childEventsInfo: event.childEventsInfo,
        onExpandChildEvents,
        onReRunChildEvents,
      });
    });

    // Add "View More" section if there are hidden events
    if (hiddenEventsCount > 0) {
      sections.push({
        eventState: "neutral",
        eventTitle: `+${hiddenEventsCount} more`,
        handleComponent: (
          <div
            onClick={onViewMoreEvents}
            className="cursor-pointer hover:bg-gray-200 transition-colors px-2 py-1 rounded"
          >
            Click to view more events
          </div>
        ),
      });
    }

    // Add next in queue if provided
    if (nextInQueue) {
      sections.push({
        eventState: "queued",
        eventTitle: nextInQueue.title,
        eventSubtitle: nextInQueue.subtitle,
      });
    }

    return sections;
  }, [visibleEvents, hiddenEventsCount, nextInQueue, onExpandChildEvents, onReRunChildEvents, onViewMoreEvents]);

  // Convert parameters to specs format
  const specs = React.useMemo(() => {
    if (parameters.length === 0) return undefined;

    return parameters.map((group) => ({
      title: Object.keys(group.items).join(", "),
      iconSlug: group.icon,
      values: Object.entries(group.items).map(([key, value]) => ({
        badges: [
          {
            label: `${key}: ${value}`,
            bgColor: "bg-gray-100",
            textColor: "text-gray-700",
          },
        ],
      })),
    }));
  }, [parameters]);

  // Handle missing state with custom component
  const customField = isMissing ? (
    <div className="px-3 py-2 bg-amber-50 border-t border-amber-200 flex items-center gap-2">
      <div className="flex-shrink-0">
        <svg className="h-4 w-4 text-amber-700" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
            clipRule="evenodd"
          />
        </svg>
      </div>
      <div className="text-sm text-amber-700">
        <span className="font-medium">Component deleted:</span> This component no longer exists and needs to be removed
        or replaced.
      </div>
    </div>
  ) : undefined;

  return (
    <ComponentBase
      iconSrc={iconSrc}
      iconSlug={iconSlug}
      iconColor={iconColor}
      iconBackground={iconBackground}
      headerColor={headerColor}
      title={title}
      metadata={metadata}
      specs={specs}
      eventSections={eventSections}
      collapsed={collapsed}
      collapsedBackground={collapsedBackground}
      selected={selected}
      onToggleCollapse={onToggleCollapse}
      onRun={onRun}
      runDisabled={runDisabled}
      runDisabledTooltip={runDisabledTooltip}
      onEdit={onEdit}
      onConfigure={onConfigure}
      onDuplicate={onDuplicate}
      onDeactivate={onDeactivate}
      onToggleView={onToggleView}
      onDelete={onDelete}
      isCompactView={isCompactView}
      includeEmptyState={eventsToDisplay.length === 0}
      emptyStateTitle="No executions received yet"
      customField={customField}
    />
  );
};
