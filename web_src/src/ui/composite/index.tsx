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
  nextInQueue?: QueueItem;
  collapsedBackground?: string;
  collapsed?: boolean;

  startLastValuesOpen?: boolean;

  onExpandChildEvents?: () => void;
  onReRunChildEvents?: () => void;
  onToggleCollapse?: () => void;
}

export const Composite: React.FC<CompositeProps> = ({ iconSrc, iconSlug, iconColor, iconBackground, headerColor, title, description, metadata, parameters = [], lastRunItem, nextInQueue, collapsed = false, collapsedBackground, onExpandChildEvents, onReRunChildEvents, onToggleCollapse, startLastValuesOpen = false }) => {
  const [showLastRunValues, setShowLastRunValues] = React.useState(startLastValuesOpen)
  const [showWaitingInfo, setShowWaitingInfo] = React.useState(false)

  const timeAgo = React.useMemo(() => {
    if (!lastRunItem?.receivedAt) return ""

    const now = new Date()
    const diff = now.getTime() - new Date(lastRunItem?.receivedAt).getTime()
    return calcRelativeTimeFromDiff(diff)
  }, [lastRunItem])

  const LastRunIcon = React.useMemo(() => {
    if (lastRunItem?.state === "success") {
      return resolveIcon("check")
    } else if (lastRunItem?.state === "running") {
      return resolveIcon("refresh-cw")
    } else {
      return resolveIcon("x")
    }
  }, [lastRunItem])

  const LastRunColor = React.useMemo(() => {
    if (lastRunItem?.state === "success") {
      return "text-green-700"
    } else if (lastRunItem?.state === "running") {
      return "text-blue-800"
    } else {
      return "text-red-700"
    }
  }, [lastRunItem])

  const LastRunBackground = React.useMemo(() => {
    if (lastRunItem?.state === "success") {
      return "bg-green-200"
    } else if (lastRunItem?.state === "running") {
      return "bg-sky-100"
    } else {
      return "bg-red-200"
    }
  }, [lastRunItem])

  const lastRunIconBackground = React.useMemo(() => {
    if (lastRunItem?.state === "success") {
      return "bg-green-600"
    } else if (lastRunItem?.state === "running") {
      return "bg-none animate-spin"
    } else {
      return "bg-red-600"
    }
  }, [lastRunItem])

  const lastRunIconColor = React.useMemo(() => {
    if (lastRunItem?.state === "success") {
      return "text-white"
    } else if (lastRunItem?.state === "running") {
      return "text-blue-800"
    } else {
      return "text-white"
    }
  }, [lastRunItem])

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

  const hasWaitingInfos = React.useMemo(() => {
    return (lastRunItem?.childEventsInfo?.waitingInfos?.length || 0) > 0
  }, [lastRunItem])

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
          <span className="text-sm">{timeAgo}</span>
        </div>

        <div onClick={() => setShowLastRunValues(!showLastRunValues)} className={`flex flex-col items-center justify-between gap-1 px-2 py-2 rounded-md cursor-pointer ${LastRunBackground} ${LastRunColor}`}>
          <div className="flex items-center gap-3 rounded-md w-full">
            <div className="w-full flex items-center gap-2 w-full">
              <div className={`w-5 h-5 rounded-full flex items-center justify-center ${lastRunIconBackground}`}>
                <LastRunIcon size={lastRunItem?.state === "running" ? 16 : 12} className={`${lastRunIconColor}`} />
              </div>
              <span className="truncate text-sm">{lastRunItem?.title}</span>
            </div>
            {lastRunItem?.subtitle && (
              <span className="text-sm text-gray-500 no-wrap whitespace-nowrap w-[20%]">{lastRunItem?.subtitle}</span>
            )}
          </div>
          {showLastRunValues && (
            <div className="flex flex-col items-center justify-between mt-1 px-2 py-2 rounded-md bg-white text-gray-500 w-full">
              {Object.entries(lastRunItem?.values || {}).map(([key, value]) => (
                <div key={key} className="flex items-center gap-1 px-2 py-1 rounded-md w-full min-w-0">
                  <span className="text-sm font-bold flex-shrink-0 text-right">{key}:</span>
                  <span className="text-sm flex-1 truncate text-left">{value}</span>
                </div>
              ))}
            </div>
          )}
        </div>
        {lastRunItem?.childEventsInfo && (
          <div className="mt-3 ml-3 text-gray-500">
            <div className="flex items-center justify-between gap-2">
              <div onClick={(e) => {
                e.stopPropagation()
                if (hasWaitingInfos) {
                  setShowWaitingInfo(!showWaitingInfo)
                }
              }} className={"flex items-center gap-2 w-full " + (hasWaitingInfos ? "cursor-pointer hover:text-gray-700 hover:scale-102 transition-all" : "")}>
                <ChildEventsArrowIcon size={18} className="text-gray-500" />
                <span className="text-sm">{lastRunItem?.childEventsInfo.count} child event{lastRunItem?.childEventsInfo.count === 1 ? "" : "s"} {lastRunItem?.childEventsInfo.state || ""}</span>
              </div>
              <div className="flex items-center gap-2">
                <ExpandChildEventsIcon size={18} className="text-gray-500 hover:text-gray-700 hover:scale-110 cursor-pointer" onClick={onExpandChildEvents} />
                <ReRunChildEventsIcon size={18} className="text-gray-500 hover:text-gray-700 hover:scale-110 cursor-pointer" onClick={onReRunChildEvents} />
              </div>
            </div>
            {hasWaitingInfos && showWaitingInfo && (
              <div className="flex flex-col items-center justify-between pl-2 py-1 rounded-md bg-white text-gray-500 w-full">
                {lastRunItem?.childEventsInfo.waitingInfos.map((waitingInfo) => {
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
      </div>

      {nextInQueue && (
        <div className="px-4 pt-3 pb-6">
          <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
            <span className="uppercase text-sm font-medium">Next In Queue</span>
          </div>
          <div className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md bg-gray-100`}>
            <div className="flex items-center gap-2 w-[80%] text-gray-500">
              <div className={`w-5 h-5 rounded-full flex items-center justify-center`}>
                <NextInQueueIcon size={20} className="text-gray-500" />
              </div>
              <span className="truncate text-sm">{nextInQueue.title}</span>
            </div>
            {nextInQueue.subtitle && (
              <span className="text-sm no-wrap whitespace-nowrap w-[20%] text-gray-500">{nextInQueue.subtitle}</span>
            )}
          </div>
        </div>
      )}
    </div>
  )
}