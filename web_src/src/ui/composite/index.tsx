import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import React from "react";
import { ComponentHeader } from "../componentHeader";
import { CollapsedComponent } from "../collapsedComponent";

type LastRunState = "success" | "failed" | "running"
type ChildEventsState = "processed" | "discarded" | "waiting" | "running"

export interface WaitingInfo {
  icon: string;
  info: string;
  futureTimeDate: Date;
}

export interface ChildEventsInfo {
  count: number;
  state?: ChildEventsState;
  waitingInfos: WaitingInfo[];
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

interface MetadataItem {
  icon: string;
  label: string;
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
  lastRunTotalCount?: number;
  nextInQueue?: QueueItem;
  collapsedBackground?: string;
  collapsed?: boolean;

  startLastValuesOpen?: boolean;

  onExpandChildEvents?: () => void;
  onReRunChildEvents?: () => void;
  onToggleCollapse?: () => void;
  onShowMoreLastRuns?: () => void;
}

export const Composite: React.FC<CompositeProps> = ({ iconSrc, iconSlug, iconColor, iconBackground, headerColor, title, description, metadata, parameters = [], lastRunItem, lastRunItems, lastRunTotalCount, nextInQueue, collapsed = false, collapsedBackground, onExpandChildEvents, onReRunChildEvents, onToggleCollapse, onShowMoreLastRuns, startLastValuesOpen = false }) => {
  const NextInQueueIcon = React.useMemo(() => {
    return resolveIcon("circle-dashed")
  }, [])

  const ChildEventsArrowIcon = React.useMemo(() => {
    return resolveIcon("corner-down-right")
  }, [])

  const ExpandChildEventsIcon = React.useMemo(() => {
    return resolveIcon("expand")
  }, [])

  const ReRunChildEventsIcon = React.useMemo(() => {
    return resolveIcon("rotate-ccw")
  }, [])

  const events: LastRunItem[] = React.useMemo(() => {
    if (lastRunItems && lastRunItems.length > 0) {
      return lastRunItems
    }

    if (lastRunItem) {
      return [lastRunItem]
    }

    return []
  }, [lastRunItem, lastRunItems])

  const [eventExpansionState, setEventExpansionState] = React.useState<Record<string, { showValues: boolean; showWaiting: boolean }>>(() => {
    if (startLastValuesOpen && events[0]) {
      return {
        [createEventKey(events[0], 0)]: { showValues: true, showWaiting: false },
      }
    }

    return {}
  })

  React.useEffect(() => {
    if (!startLastValuesOpen || !events[0]) {
      return
    }

    const key = createEventKey(events[0], 0)

    setEventExpansionState((prev) => {
      if (prev[key]?.showValues) {
        return prev
      }

      return {
        ...prev,
        [key]: { showValues: true, showWaiting: prev[key]?.showWaiting ?? false },
      }
    })
  }, [events, startLastValuesOpen])

  const latestTimeAgo = React.useMemo(() => {
    if (!events[0]?.receivedAt) {
      return ""
    }

    const now = new Date()
    const diff = now.getTime() - new Date(events[0].receivedAt).getTime()
    return calcRelativeTimeFromDiff(diff)
  }, [events])

  const totalEventsCount = React.useMemo(() => {
    if (typeof lastRunTotalCount === "number") {
      return lastRunTotalCount
    }

    return events.length
  }, [events.length, lastRunTotalCount])

  const remainingEventsCount = Math.max(0, totalEventsCount - events.length)

  const toggleEventValues = (eventKey: string) => {
    setEventExpansionState((prev) => {
      const current = prev[eventKey]?.showValues ?? false

      return {
        ...prev,
        [eventKey]: {
          showValues: !current,
          showWaiting: prev[eventKey]?.showWaiting ?? false,
        },
      }
    })
  }

  const toggleWaitingInfos = (eventKey: string) => {
    setEventExpansionState((prev) => {
      const current = prev[eventKey]?.showWaiting ?? false

      return {
        ...prev,
        [eventKey]: {
          showValues: prev[eventKey]?.showValues ?? false,
          showWaiting: !current,
        },
      }
    })
  }

  if (collapsed) {
    return (
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
          <div className="flex flex-col gap-1 text-gray-500 mt-1">
            {parameters.map((group, index) => {
              const Icon = resolveIcon(group.icon)
              return (
                <div key={index} className="flex items-center gap-2">
                  <Icon size={16} />
                  <span className="text-sm font-mono">{group.items.join(", ")}</span>
                </div>
              )
            })}
          </div>
        )}
      </CollapsedComponent>
    )
  }

  return (
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

      {parameters.length > 0 &&
        <div className="px-2 py-3 border-b text-gray-500 flex flex-col gap-2">
          {parameters.map((group, index) => {
            const Icon = resolveIcon(group.icon)
            return (
              <div key={index} className="flex items-center gap-2">
                <Icon size={19} />
                <span className="text-sm font-mono">{group.items.join(", ")}</span>
              </div>
            )
          })}
        </div>
      }

      {metadata && metadata.length > 0 && (
        <div className="px-2 py-3 border-b text-gray-500 flex flex-col gap-2">
          {metadata.map((item, index) => {
            const Icon = resolveIcon(item.icon)
            return (
              <div key={index} className="flex items-center gap-2">
                <Icon size={16} />
                <span className="text-sm">{item.label}</span>
              </div>
            )
          })}
        </div>
      )}

      <div className="px-4 py-3 border-b">
        <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
          <span className="uppercase text-sm font-medium">Last Run</span>
          {events.length > 0 && <span className="text-sm">{latestTimeAgo}</span>}
        </div>

        {events.length > 0 ? (
          <>
            <div className="flex flex-col gap-3">
              {events.map((event, index) => {
                const key = createEventKey(event, index)
                const expansion = eventExpansionState[key] || { showValues: false, showWaiting: false }
                const { backgroundClass, textClass, iconBackgroundClass, iconColorClass, Icon: EventStateIcon, iconSize } = resolveLastRunState(event.state)
                const hasWaitingInfos = (event.childEventsInfo?.waitingInfos?.length || 0) > 0
                const relativeTime = event.receivedAt ? calcRelativeTimeFromDiff(new Date().getTime() - new Date(event.receivedAt).getTime()) : ""

                return (
                  <React.Fragment key={key}>
                    <div
                      onClick={() => toggleEventValues(key)}
                      className={`flex flex-col items-center justify-between gap-1 px-2 py-2 rounded-md cursor-pointer ${backgroundClass} ${textClass}`}
                    >
                      <div className="flex items-center gap-3 rounded-md w-full min-w-0">
                        <div className="flex items-center gap-2 min-w-0 flex-1">
                          <div className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center ${iconBackgroundClass}`}>
                            <EventStateIcon size={iconSize} className={iconColorClass} />
                          </div>
                          <div className="flex flex-col min-w-0">
                            <span className="truncate text-sm">{event.title}</span>
                            {relativeTime && <span className="text-xs text-gray-700">{relativeTime}</span>}
                          </div>
                        </div>
                        {event.subtitle && (
                          <span className="text-sm text-gray-600 truncate flex-shrink-0 max-w-[40%]">{event.subtitle}</span>
                        )}
                      </div>
                      {expansion.showValues && (
                        <div className="flex flex-col items-center justify-between mt-1 px-2 py-2 rounded-md bg-white text-gray-600 w-full">
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
                      <div className="mt-1 ml-3 text-gray-500">
                        <div className="flex items-center justify-between gap-2">
                          <div
                            onClick={(e) => {
                              e.stopPropagation()
                              if (hasWaitingInfos) {
                                toggleWaitingInfos(key)
                              }
                            }}
                            className={
                              "flex items-center gap-2 w-full " +
                              (hasWaitingInfos ? "cursor-pointer hover:text-gray-700 hover:scale-102 transition-all" : "")
                            }
                          >
                            <ChildEventsArrowIcon size={18} className="text-gray-500" />
                            <span className="text-sm">
                              {event.childEventsInfo.count} child event{event.childEventsInfo.count === 1 ? "" : "s"}{" "}
                              {event.childEventsInfo.state || ""}
                            </span>
                          </div>
                          <div className="flex items-center gap-2">
                            <ExpandChildEventsIcon
                              size={18}
                              className="text-gray-500 hover:text-gray-700 hover:scale-110 cursor-pointer"
                              onClick={onExpandChildEvents}
                            />
                            <ReRunChildEventsIcon
                              size={18}
                              className="text-gray-500 hover:text-gray-700 hover:scale-110 cursor-pointer"
                              onClick={onReRunChildEvents}
                            />
                          </div>
                        </div>
                        {hasWaitingInfos && expansion.showWaiting && (
                          <div className="flex flex-col items-center justify-between pl-2 py-1 rounded-md bg-white text-gray-500 w-full">
                            {event.childEventsInfo.waitingInfos.map((waitingInfo) => {
                              const Icon = resolveIcon(waitingInfo.icon)
                              return (
                                <div key={waitingInfo.info} className="flex justify-between items-center gap-3 pl-2 py-1 rounded-md w-full">
                                  <span className="text-sm text-right flex items-center gap-2">
                                    <Icon size={18} className="text-gray-500" />
                                    {waitingInfo.info}
                                  </span>
                                  <span className="text-sm">
                                    {calcRelativeTimeFromDiff(new Date(waitingInfo.futureTimeDate).getTime() - new Date().getTime())}
                                    &nbsp;left
                                  </span>
                                </div>
                              )
                            })}
                          </div>
                        )}
                      </div>
                    )}
                  </React.Fragment>
                )
              })}
            </div>
            {remainingEventsCount > 0 &&
              (onShowMoreLastRuns ? (
                <button
                  type="button"
                  className="mt-3 text-sm text-blue-700 hover:underline"
                  onClick={onShowMoreLastRuns}
                >
                  +{remainingEventsCount} more
                </button>
              ) : (
                <span className="mt-3 block text-sm text-blue-700">+{remainingEventsCount} more</span>
              ))}
          </>
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
  )
}

function resolveLastRunState(state: LastRunState | undefined) {
  if (state === "success") {
    return {
      backgroundClass: "bg-green-200",
      textClass: "text-green-700",
      iconBackgroundClass: "bg-green-600",
      iconColorClass: "text-white",
      Icon: resolveIcon("check"),
      iconSize: 12,
    }
  }

  if (state === "running") {
    return {
      backgroundClass: "bg-sky-100",
      textClass: "text-blue-800",
      iconBackgroundClass: "bg-none animate-spin",
      iconColorClass: "text-blue-800",
      Icon: resolveIcon("refresh-cw"),
      iconSize: 16,
    }
  }

  return {
    backgroundClass: "bg-red-200",
    textClass: "text-red-700",
    iconBackgroundClass: "bg-red-600",
    iconColorClass: "text-white",
    Icon: resolveIcon("x"),
    iconSize: 12,
  }
}

function createEventKey(event: LastRunItem, index: number) {
  const receivedAt = event.receivedAt ? new Date(event.receivedAt).getTime() : index
  return `${index}-${event.title}-${event.subtitle || ""}-${receivedAt}`
}
