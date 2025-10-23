import React from "react";
import { ApprovalItem, type ApprovalItemProps } from "../approvalItem";
import { ItemGroup, ItemSeparator } from "../item";
import { CircleDashedIcon } from "lucide-react"
import { resolveIcon } from "@/lib/utils";

export interface AwaitingEvent {
  title: string;
  subtitle?: string;
}

export interface ApprovalProps {
  iconSrc?: string;
  iconSlug?: string;
  iconBackground?: string;
  iconColor?: string;
  headerColor: string;
  title: string;
  description?: string;
  approvals: ApprovalItemProps[];
  awaitingEvent: AwaitingEvent;
  collapsedBackground?: string;
  receivedAt: Date;
  collapsed?: boolean;
}

export const Approval: React.FC<ApprovalProps> = ({ iconSrc, iconSlug, iconBackground, iconColor, headerColor, title, description, collapsed = false, collapsedBackground, receivedAt, approvals, awaitingEvent }) => {
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
    const diff = now.getTime() - receivedAt.getTime()
    return calcRelativeTimeFromDiff(diff)
  }, [receivedAt])

  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug)
  }, [iconSlug])


  if (collapsed) {
    return (
      <div className="flex w-fit flex-col items-center">
        <div className={`flex h-20 w-20 items-center justify-center rounded-md border border-border ${collapsedBackground || ''}`}>
          {iconSrc ? <img src={iconSrc} alt={title} className="h-12 w-12 object-contain" /> : <Icon size={30} className={iconColor} />}
        </div>
        <h2 className="text-base font-semibold text-neutral-900 pt-1">{title}</h2>
      </div>
    )
  }

  return (
    <div className="flex flex-col border border-border rounded-md w-[30rem] bg-white" >
      <div className={"w-full px-2 flex flex-col border-b p-2 gap-2 rounded-t-md items-center " + headerColor}>
        <div className="w-full flex items-center gap-2">
          <div className={`w-6 h-6 rounded-full overflow-hidden flex items-center justify-center ${iconBackground || ''}`}>
            {iconSrc ? <img src={iconSrc} alt={title} className="w-5 h-5 " /> : <Icon size={20} className={iconColor} />}
          </div>
          <h2 className="text-md font-semibold">{title}</h2>
        </div>
        {description && <p className="w-full text-md text-gray-500 pl-8">{description}</p>}
      </div>

      <div className="px-4 py-3">
        <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
          <span className="uppercase text-sm font-medium">Awaiting Approval</span>
          <span className="text-sm">{timeAgo}</span>
        </div>

        <div className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md bg-orange-200 mb-4`}>
          <div className="flex items-center gap-2 w-[80%] text-amber-800">
            <div className={`w-5 h-5 rounded-full flex items-center justify-center`}>
              <CircleDashedIcon size={20} className="text-amber-800" />
            </div>
            <span className="truncate text-sm">{awaitingEvent?.title}</span>
          </div>
          {awaitingEvent?.subtitle && (
            <span className="text-sm no-wrap whitespace-nowrap w-[20%] text-amber-800">{awaitingEvent?.subtitle}</span>
          )}
        </div>


        <ItemGroup className="w-full">
          {approvals.map((approval, index) => (
            <React.Fragment key={`${approval.title}-${index}`}>
              <ApprovalItem {...approval} />
            </React.Fragment>
          ))}
        </ItemGroup>
      </div>
    </div>
  )
}