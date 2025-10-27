import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import React from "react";
import { ComponentHeader } from "../componentHeader";
import { CollapsedComponent } from "../collapsedComponent";

export interface TriggerMetadataItem {
  icon: string;
  label: string;
}

type LastEventState = "processed" | "discarded"

interface TriggerLastEventData {
  title: string;
  subtitle?: string;
  receivedAt: Date;
  state: LastEventState;
}

export interface TriggerProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  headerColor: string;
  title: string;
  description?: string;
  metadata: TriggerMetadataItem[];
  lastEventData?: TriggerLastEventData;
  zeroStateText?: string;
  collapsedBackground?: string;
  collapsed?: boolean;
}

export const Trigger: React.FC<TriggerProps> = ({ iconSrc, iconSlug, iconColor, iconBackground, headerColor, title, description, metadata, lastEventData, zeroStateText = "No events yet", collapsed = false, collapsedBackground }) => {
  const timeAgo = React.useMemo(() => {
    if (!lastEventData) return null;
    const now = new Date()
    const diff = now.getTime() - lastEventData.receivedAt.getTime()
    return calcRelativeTimeFromDiff(diff)
  }, [lastEventData])

  const LastEventIcon = React.useMemo(() => {
    if (!lastEventData) return null;
    if (lastEventData.state === "processed") {
      return resolveIcon("check")
    } else {
      return resolveIcon("x")
    }
  }, [lastEventData])

  const LastEventColor = React.useMemo(() => {
    if (!lastEventData) return "text-gray-700";
    if (lastEventData.state === "processed") {
      return "text-green-700"
    } else {
      return "text-red-700"
    }
  }, [lastEventData])

  const LastEventBackground = React.useMemo(() => {
    if (!lastEventData) return "bg-gray-200";
    if (lastEventData.state === "processed") {
      return "bg-green-200"
    } else {
      return "bg-red-200"
    }
  }, [lastEventData])

  if (collapsed) {
    return (
      <CollapsedComponent
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        iconColor={iconColor}
        iconBackground={iconBackground}
        title={title}
        collapsedBackground={collapsedBackground}
        shape="circle"
      >
        <div className="flex flex-col items-center gap-1">
          {metadata.map((item, index) => {
            const Icon = resolveIcon(item.icon)
            return (
              <div key={index} className="flex items-center gap-1 text-xs text-gray-500">
                <Icon size={12} />
                <span>{item.label}</span>
              </div>
            )
          })}
        </div>
      </CollapsedComponent>
    )
  }

  return (
    <div className="flex flex-col border-2 border-border rounded-md w-[23rem] bg-white" >
      <ComponentHeader
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        iconBackground={iconBackground}
        iconColor={iconColor}
        headerColor={headerColor}
        title={title}
        description={description}
      />
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
        {lastEventData ? (
          <>
            <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
              <span className="uppercase text-sm font-medium">Last Event</span>
              <span className="text-sm">{timeAgo}</span>
            </div>
            <div className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md ${LastEventBackground} ${LastEventColor}`}>
              <div className="flex items-center gap-2 w-[80%]">
                <div className={`w-5 h-5 rounded-full flex items-center justify-center ${lastEventData.state === "processed" ? "bg-green-600" : "bg-red-600"}`}>
                  {LastEventIcon && <LastEventIcon size={12} className="text-white" />}
                </div>
                <span className="truncate text-sm">{lastEventData.title}</span>
              </div>
              {lastEventData.subtitle && (
                <span className="text-sm no-wrap whitespace-nowrap w-[20%]">{lastEventData.subtitle}</span>
              )}
            </div>
          </>
        ) : (
          <div className="flex items-center justify-center px-2 py-4 rounded-md bg-gray-50 border border-dashed border-gray-300">
            <span className="text-sm text-gray-400">{zeroStateText}</span>
          </div>
        )}
      </div>
    </div>
  )
}