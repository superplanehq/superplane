import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import React from "react";
import { ComponentHeader } from "../componentHeader";
import { CollapsedComponent } from "../collapsedComponent";
import { MetadataList, type MetadataItem } from "../metadataList";
import { ChildEvents, type ChildEventsInfo } from "../childEvents";
import { SelectionWrapper } from "../selectionWrapper";

export type LastRunState = "success" | "failed" | "running"
export type ChildEventsState = "processed" | "discarded" | "waiting" | "running"

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
}

export interface ParameterGroup {
  icon: string;
  items: string[];
}

export interface CompositeProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  headerColor: string;
  title: string;
  description?: string;
  metadata?: MetadataItem[];
  parameters?: ParameterGroup[];
  lastRunItem?: LastRunItem;
  lastRunItems?: LastRunItem[];
  maxVisibleEvents?: number;
  nextInQueue?: QueueItem;
  collapsedBackground?: string;
  collapsed?: boolean;
  selected?: boolean;

  startLastValuesOpen?: boolean;

  onExpandChildEvents?: () => void;
  onReRunChildEvents?: () => void;
  onToggleCollapse?: () => void;
  onViewMoreEvents?: () => void;
}

export const Composite: React.FC<CompositeProps> = ({ iconSrc, iconSlug, iconColor, iconBackground, headerColor, title, description, metadata, parameters = [], lastRunItem, lastRunItems, maxVisibleEvents = 5, nextInQueue, collapsed = false, collapsedBackground, onExpandChildEvents, onReRunChildEvents, onToggleCollapse, onViewMoreEvents, startLastValuesOpen = false, selected = false }) => {
  // All hooks must be called before any early returns
  const [showLastRunValues, setShowLastRunValues] = React.useState<Record<number, boolean>>(
    startLastValuesOpen ? { 0: true } : {}
  )

  // Use lastRunItems if provided, otherwise fall back to single lastRunItem
  const eventsToDisplay = React.useMemo(() => {
    if (lastRunItems && lastRunItems.length > 0) {
      return lastRunItems
    } else if (lastRunItem) {
      return [lastRunItem]
    }
    return []
  }, [lastRunItem, lastRunItems])

  const visibleEvents = React.useMemo(() => {
    return eventsToDisplay.slice(0, maxVisibleEvents)
  }, [eventsToDisplay, maxVisibleEvents])

  const hiddenEventsCount = React.useMemo(() => {
    return Math.max(0, eventsToDisplay.length - maxVisibleEvents)
  }, [eventsToDisplay.length, maxVisibleEvents])

  const toggleEventValues = React.useCallback((index: number) => {
    setShowLastRunValues(prev => ({
      ...prev,
      [index]: !prev[index]
    }))
  }, [])

  const getEventIcon = React.useCallback((state: LastRunState) => {
    if (state === "success") {
      return resolveIcon("check")
    } else if (state === "running") {
      return resolveIcon("refresh-cw")
    } else {
      return resolveIcon("x")
    }
  }, [])

  const getEventColor = React.useCallback((state: LastRunState) => {
    if (state === "success") {
      return "text-green-700"
    } else if (state === "running") {
      return "text-blue-800"
    } else {
      return "text-red-700"
    }
  }, [])

  const getEventBackground = React.useCallback((state: LastRunState) => {
    if (state === "success") {
      return "bg-green-200"
    } else if (state === "running") {
      return "bg-sky-100"
    } else {
      return "bg-red-200"
    }
  }, [])

  const getEventIconBackground = React.useCallback((state: LastRunState) => {
    if (state === "success") {
      return "bg-green-600"
    } else if (state === "running") {
      return "bg-none animate-spin"
    } else {
      return "bg-red-600"
    }
  }, [])

  const getEventIconColor = React.useCallback((state: LastRunState) => {
    if (state === "success") {
      return "text-white"
    } else if (state === "running") {
      return "text-blue-800"
    } else {
      return "text-white"
    }
  }, [])

  const NextInQueueIcon = React.useMemo(() => {
    return resolveIcon("circle-dashed")
  }, [])

  // Now safe to do early return after all hooks are called
  if (collapsed) {
    return (
      <SelectionWrapper selected={selected}>
        <CollapsedComponent
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconColor={iconColor}
          iconBackground={iconBackground}
          title={title}
          collapsedBackground={collapsedBackground}
          shape="rounded"
          onDoubleClick={onToggleCollapse}
        >
          {parameters.length > 0 && (
            <MetadataList
              items={parameters.map(group => ({
                icon: group.icon,
                label: group.items.join(", ")
              }))}
              className="flex flex-col gap-1 text-gray-500 mt-1"
              iconSize={16}
            />
          )}
        </CollapsedComponent>
      </SelectionWrapper>
    )
  }

  return (
    <SelectionWrapper selected={selected}>
      <div className="flex flex-col border-2 border-border rounded-md w-[26rem] bg-white" >
        <ComponentHeader
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        iconBackground={iconBackground}
        iconColor={iconColor}
        headerColor={headerColor}
        title={title}
        description={description}
        onDoubleClick={onToggleCollapse}
      />

      {parameters.length > 0 && (
        <MetadataList
          items={parameters.map(group => ({
            icon: group.icon,
            label: group.items.join(", ")
          }))}
          className="px-2 py-3 border-b text-gray-500 flex flex-col gap-2"
        />
      )}

      {metadata && metadata.length > 0 && (
        <MetadataList
          items={metadata}
          className="px-2 py-3 border-b text-gray-500 flex flex-col gap-2"
          iconSize={16}
        />
      )}

      <div className="px-4 py-3 border-b">
        <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
          <span className="uppercase text-sm font-medium">Last Run</span>
        </div>

        {eventsToDisplay.length > 0 ? (
          <div className="flex flex-col gap-2">
            {visibleEvents.map((event, index) => {
              const EventIcon = getEventIcon(event.state)
              const eventColor = getEventColor(event.state)
              const eventBackground = getEventBackground(event.state)
              const eventIconBackground = getEventIconBackground(event.state)
              const eventIconColor = getEventIconColor(event.state)
              const now = new Date()
              const diff = now.getTime() - new Date(event.receivedAt).getTime()
              const timeAgo = calcRelativeTimeFromDiff(diff)

              return (
                <div key={index}>
                  <div onClick={() => toggleEventValues(index)} className={`flex flex-col items-center justify-between gap-1 px-2 py-2 rounded-md cursor-pointer ${eventBackground} ${eventColor}`}>
                    <div className="flex items-center gap-3 rounded-md w-full min-w-0">
                      <div className="flex items-center gap-2 min-w-0 flex-1">
                        <div className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center ${eventIconBackground}`}>
                          <EventIcon size={event.state === "running" ? 16 : 12} className={`${eventIconColor}`} />
                        </div>
                        <span className="truncate text-sm">{event.title}</span>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        {event.subtitle && (
                          <span className="text-sm text-gray-500 truncate max-w-[100px]">{event.subtitle}</span>
                        )}
                        <span className="text-xs text-gray-500">{timeAgo}</span>
                      </div>
                    </div>
                    {showLastRunValues[index] && (
                      <div className="flex flex-col items-center justify-between mt-1 px-2 py-2 rounded-md bg-white text-gray-500 w-full">
                        {Object.entries(event.values || {}).map(([key, value]) => (
                          <div key={key} className="flex items-center gap-1 px-2 py-1 rounded-md w-full min-w-0">
                            <span className="text-sm font-bold flex-shrink-0 text-right">{key}:</span>
                            <span className="text-sm flex-1 truncate text-left">{value}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                  {event.childEventsInfo && (
                    <ChildEvents
                      childEventsInfo={event.childEventsInfo}
                      onExpandChildEvents={onExpandChildEvents}
                      onReRunChildEvents={onReRunChildEvents}
                    />
                  )}
                </div>
              )
            })}
            {hiddenEventsCount > 0 && (
              <div
                onClick={onViewMoreEvents}
                className="flex items-center justify-center px-2 py-2 rounded-md bg-gray-100 text-gray-600 cursor-pointer hover:bg-gray-200 transition-colors"
              >
                <span className="text-sm font-medium">+{hiddenEventsCount} more</span>
              </div>
            )}
          </div>
        ) : (
          <div className="flex items-center gap-3 px-2 py-2 rounded-md bg-gray-100 text-gray-500">
            <div className="w-5 h-5 rounded-full flex items-center justify-center bg-gray-400">
              <div className="w-2 h-2 rounded-full bg-white"></div>
            </div>
            <span className="text-sm">No executions received yet</span>
          </div>
        )}
      </div>

      {nextInQueue && (
        <div className="px-4 pt-3 pb-6">
          <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
            <span className="uppercase text-sm font-medium">Next In Queue</span>
          </div>
          <div className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md bg-gray-100 min-w-0`}>
            <div className="flex items-center gap-2 text-gray-500 min-w-0 flex-1">
              <div className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center`}>
                <NextInQueueIcon size={20} className="text-gray-500" />
              </div>
              <span className="truncate text-sm">{nextInQueue.title}</span>
            </div>
            {nextInQueue.subtitle && (
              <span className="text-sm truncate text-gray-500 flex-shrink-0 max-w-[40%]">{nextInQueue.subtitle}</span>
            )}
          </div>
        </div>
      )}
      </div>
    </SelectionWrapper>
  )
}