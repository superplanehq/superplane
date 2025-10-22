import { BookMarked, type LucideIcon } from "lucide-react";
import * as LucideIcons from "lucide-react"
import React from "react";

type LastRunState = "success" | "failed" | "running"
type ChildEventsState = "processed" | "discarded" | "waiting" | "running"

interface WaitingInfo {
  icon: string;
  info: string;
  futureTimeDate: Date;
}

interface ChildEventsInfo {
  count: number;
  state?: ChildEventsState;
  waitingInfos: WaitingInfo[];
}

interface QueueItem {
  title: string;
  subtitle: string;
  receivedAt: Date;
}

interface LastRunItem extends QueueItem {
  childEventsInfo?: ChildEventsInfo;
  state: LastRunState;
  values: Record<string, string>;
}

export interface CompositeProps {
  iconSrc: string;
  iconBackground?: string;
  headerColor: string;
  title: string;
  description?: string;
  parameters: string[];
  parametersIcon: string;
  lastRunItem: LastRunItem;
  nextInQueue?: QueueItem;
  collapsedBackground?: string;
  collapsed?: boolean;

  startLastValuesOpen?: boolean;

  onExpandChildEvents?: () => void;
  onReRunChildEvents?: () => void;
}

export const Composite: React.FC<CompositeProps> = ({ iconSrc, iconBackground, headerColor, title, description, parameters, parametersIcon, lastRunItem, nextInQueue, collapsed = false, collapsedBackground, onExpandChildEvents, onReRunChildEvents, startLastValuesOpen = false }) => {
  const [showLastRunValues, setShowLastRunValues] = React.useState(startLastValuesOpen)

  const resolveIcon = React.useCallback((slug?: string): LucideIcon => {
    if (!slug) {
      return BookMarked
    }

    const pascalCase = slug
      .split("-")
      .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
      .join("")

    const candidate = (LucideIcons as Record<string, unknown>)[pascalCase]

    if (
      candidate &&
      (typeof candidate === "function" ||
        (typeof candidate === "object" && "render" in candidate))
    ) {
      return candidate as LucideIcon
    }

    return BookMarked
  }, [])

  const calcRelativeTimeFromDiff = (diff: number) => {
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)
    if (days > 0) {
      return `${days}d`
    } else if (hours > 0) {
      return `${hours}h`
    } else if (minutes > 0) {
      return `${minutes}m`
    } else {
      return `${seconds}s`
    }
  }

  const timeAgo = React.useMemo(() => {
    const now = new Date()
    const diff = now.getTime() - lastRunItem.receivedAt.getTime()
    return calcRelativeTimeFromDiff(diff)
  }, [lastRunItem])

  const LastRunIcon = React.useMemo(() => {
    if (lastRunItem.state === "success") {
      return resolveIcon("check")
    } else if (lastRunItem.state === "running") {
      return resolveIcon("refresh-cw")
    } else {
      return resolveIcon("x")
    }
  }, [lastRunItem])

  const LastRunColor = React.useMemo(() => {
    if (lastRunItem.state === "success") {
      return "text-green-700"
    } else if (lastRunItem.state === "running") {
      return "text-blue-800"
    } else {
      return "text-red-700"
    }
  }, [lastRunItem])

  const LastRunBackground = React.useMemo(() => {
    if (lastRunItem.state === "success") {
      return "bg-green-200"
    } else if (lastRunItem.state === "running") {
      return "bg-sky-100"
    } else {
      return "bg-red-200"
    }
  }, [lastRunItem])

  const lastRunIconBackground = React.useMemo(() => {
    if (lastRunItem.state === "success") {
      return "bg-green-600"
    } else if (lastRunItem.state === "running") {
      return "bg-none animate-spin"
    } else {
      return "bg-red-600"
    }
  }, [lastRunItem])

  const lastRunIconColor = React.useMemo(() => {
    if (lastRunItem.state === "success") {
      return "text-white"
    } else if (lastRunItem.state === "running") {
      return "text-blue-800"
    } else {
      return "text-white"
    }
  }, [lastRunItem])

  const NextInQueueIcon = React.useMemo(() => {
    return resolveIcon("circle-dashed")
  }, [])

  const ParametersIcon = React.useMemo(() => {
    return resolveIcon(parametersIcon)
  }, [parametersIcon])

  const ChildEventsArrowIcon = React.useMemo(() => {
    return resolveIcon("corner-down-right")
  }, [])

  if (collapsed) {
    return (
      <div className="flex w-fit flex-col items-center">
        <div className={`flex h-20 w-20 items-center justify-center rounded-md border border-border ${collapsedBackground || ''}`}>
          <img src={iconSrc} alt={title} className="h-12 w-12 object-contain" />
        </div>
        <h2 className="text-base font-semibold text-neutral-900 pt-1">{title}</h2>
      </div>
    )
  }

  const ExpandChildEventsIcon = React.useMemo(() => {
    return resolveIcon("expand")
  }, [])

  const ReRunChildEventsIcon = React.useMemo(() => {
    return resolveIcon("rotate-ccw")
  }, [])

  return (
    <div className="flex flex-col border border-border rounded-md w-[26rem] bg-white" >
      <div className={"w-full px-2 flex flex-col border-b p-2 gap-2 rounded-t-md items-center " + headerColor}>
        <div className="w-full flex items-center gap-2">
          <div className={`w-6 h-6 rounded-full overflow-hidden flex items-center justify-center ${iconBackground || ''}`}>
            <img src={iconSrc} alt={title} className="w-5 h-5 " />
          </div>
          <h2 className="text-md font-semibold">{title}</h2>
        </div>
        {description && <p className="w-full text-sm text-gray-500 pl-8">{description}</p>}
      </div>

      {parameters.length > 0 &&
        <div className="px-2 py-3 border-b text-gray-500 flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <ParametersIcon size={19} />
            <span className="text-sm">{parameters.join(", ")}</span>
          </div>
        </div>
      }

      <div className="px-4 py-3 border-b">
        <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
          <span className="uppercase text-sm font-medium">Last Run</span>
          <span className="text-sm">{timeAgo}</span>
        </div>

        <div onClick={() => setShowLastRunValues(!showLastRunValues)} className={`flex flex-col items-center justify-between gap-1 px-2 py-2 rounded-md cursor-pointer ${LastRunBackground} ${LastRunColor}`}>
          <div className="flex items-center gap-3 rounded-md w-full">
            <div className="w-full flex items-center gap-2 w-full">
              <div className={`w-5 h-5 rounded-full flex items-center justify-center ${lastRunIconBackground}`}>
                <LastRunIcon size={lastRunItem.state === "running" ? 16 : 12} className={`${lastRunIconColor}`} />
              </div>
              <span className="truncate text-sm">{lastRunItem.title}</span>
            </div>
            {lastRunItem.subtitle && (
              <span className="text-sm text-gray-500 no-wrap whitespace-nowrap w-[20%]">{lastRunItem.subtitle}</span>
            )}
          </div>
          {showLastRunValues && (
            <div className="flex flex-col items-center justify-between mt-1 px-2 py-2 rounded-md bg-white text-gray-500 w-full">
              {Object.entries(lastRunItem.values).map(([key, value]) => (
                <div key={key} className="flex justify-between gap-3 px-2 py-1 rounded-md w-full">
                  <span className="text-sm w-[20%] text-right">{key}</span>
                  <span className="text-sm w-[80%]">{value}</span>
                </div>
              ))}
            </div>
          )}
        </div>
        {lastRunItem.childEventsInfo && (
          <div className="mt-3 ml-3 text-gray-500">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 w-full">
                <ChildEventsArrowIcon size={18} className="text-gray-500" />
                <span className="text-sm">{lastRunItem.childEventsInfo.count} child event{lastRunItem.childEventsInfo.count === 1 ? "" : "s"} {lastRunItem.childEventsInfo.state || ""}</span>
              </div>
              <div className="flex items-center gap-2">
                <ExpandChildEventsIcon size={18} className="text-gray-500 hover:text-gray-700 hover:scale-110 cursor-pointer" onClick={onExpandChildEvents} />
                <ReRunChildEventsIcon size={18} className="text-gray-500 hover:text-gray-700 hover:scale-110 cursor-pointer" onClick={onReRunChildEvents} />
              </div>
            </div>
            {lastRunItem.childEventsInfo.waitingInfos && (
              <div className="flex flex-col items-center justify-between pl-2 py-1 rounded-md bg-white text-gray-500 w-full">
                {lastRunItem.childEventsInfo.waitingInfos.map((waitingInfo) => {
                  const Icon = resolveIcon(waitingInfo.icon)
                  return (
                    <div key={waitingInfo.info} className="flex justify-between items-center gap-3 pl-2 py-1 rounded-md w-full">
                      <span className="text-sm text-right flex items-center gap-2">
                        <Icon size={18} className="text-gray-500" />
                        {waitingInfo.info}
                      </span>
                      <span className="text-sm">
                        {calcRelativeTimeFromDiff(waitingInfo.futureTimeDate.getTime() - new Date().getTime())}
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

      <div className="px-4 pt-3 pb-6">
        <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
          <span className="uppercase text-sm font-medium">Next In Queue</span>
        </div>
        {nextInQueue ? <div className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md bg-gray-100`}>
          <div className="flex items-center gap-2 w-[80%] text-gray-500">
            <div className={`w-5 h-5 rounded-full flex items-center justify-center`}>
              <NextInQueueIcon size={20} className="text-gray-500" />
            </div>
            <span className="truncate text-sm">{nextInQueue?.title}</span>
          </div>
          {nextInQueue?.subtitle && (
            <span className="text-sm no-wrap whitespace-nowrap w-[20%] text-gray-500">{nextInQueue?.subtitle}</span>
          )}
        </div> :
          <div className="text-sm text-gray-500 bg-gray-100 px-2 py-2 rounded-md w-full">
            <div className="flex items-center gap-2">
              <NextInQueueIcon size={20} className="text-gray-500" />
              No item in queue...
            </div>
          </div>
        }
      </div>
    </div>
  )
}