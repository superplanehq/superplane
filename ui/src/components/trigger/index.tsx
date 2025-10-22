import { BookMarked, type LucideIcon } from "lucide-react";
import * as LucideIcons from "lucide-react"
import React from "react";

interface TriggerMetadataItem {
  icon: string;
  label: string;
}

type LastEventState = "processed" | "discarded"

interface TriggerLastEventData {
  title: string;
  sizeInMB?: number;
  receivedAt: Date;
  state: LastEventState;
}

export interface TriggerProps {
  iconSrc: string;
  iconBackground?: string;
  headerColor: string;
  title: string;
  metadata: TriggerMetadataItem[];
  lastEventData: TriggerLastEventData;
}

export const Trigger: React.FC<TriggerProps> = ({ iconSrc, iconBackground, headerColor, title, metadata, lastEventData }) => {
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

  const timeAgo = React.useMemo(() => {
    const now = new Date()
    const diff = now.getTime() - lastEventData.receivedAt.getTime()
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
  }, [lastEventData])

  const LastEventIcon = React.useMemo(() => {
    if (lastEventData.state === "processed") {
      return resolveIcon("check")
    } else {
      return resolveIcon("x")
    }
  }, [lastEventData])

  const LastEventColor = React.useMemo(() => {
    if (lastEventData.state === "processed") {
      return "text-green-700"
    } else {
      return "text-red-700"
    }
  }, [lastEventData])

  const LastEventBackground = React.useMemo(() => {
    if (lastEventData.state === "processed") {
      return "bg-green-200"
    } else {
      return "bg-red-200"
    }
  }, [lastEventData])

  return (
    <div className="flex flex-col border border-border rounded-md w-[23rem]" >
      <div className={"px-2 flex border-b p-2 gap-2 rounded-t-md items-center " + headerColor}>
        <div className={`w-6 h-6 rounded-full overflow-hidden flex items-center justify-center ${iconBackground || ''}`}>
          <img src={iconSrc} alt={title} className="w-5 h-5 " />
        </div>
        <h2 className="text-md font-semibold">{title}</h2>
      </div>
      <div className="px-2 py-3 border-b text-gray-500 flex flex-col gap-2">
        {metadata.map((item, index) => {
          const Icon = resolveIcon(item.icon)
          return (
            <div key={index} className="flex items-center gap-2">
              <Icon size={19} />
              <span className="text-sm">{item.label}</span>
            </div>
          )
        })}
      </div>
      <div className="px-4 pt-3 pb-6">
        <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
          <span className="uppercase text-sm font-medium">Last Event</span>
          <span className="text-sm">{timeAgo}</span>
        </div>
        <div className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md ${LastEventBackground} ${LastEventColor}`}>
          <div className="flex items-center gap-2 w-[80%]">
            <div className={`w-5 h-5 rounded-full flex items-center justify-center ${lastEventData.state === "processed" ? "bg-green-600" : "bg-red-600"}`}>
              <LastEventIcon size={12} className="text-white" />
            </div>
            <span className="truncate text-sm">{lastEventData.title}</span>
          </div>
          {lastEventData.sizeInMB && (
            <span className="text-sm no-wrap whitespace-nowrap w-[20%]">{lastEventData.sizeInMB.toFixed(1)} MB</span>
          )}
        </div>
      </div>
    </div>
  )
}