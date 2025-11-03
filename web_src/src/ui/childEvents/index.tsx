import { resolveIcon } from "@/lib/utils";
import React from "react";
import type { ChildEventsState } from "../composite";

export interface WaitingInfo {
  icon: string;
  info: string;
  futureTimeDate?: Date;
}

export interface ChildEventsInfo {
  count: number;
  state?: ChildEventsState;
  waitingInfos: WaitingInfo[];
  items?: {
    label: string;
    state: ChildEventsState;
    startedAt?: Date;
  }[];
}

export interface ChildEventsProps {
  childEventsInfo: ChildEventsInfo;
  className?: string;
  onExpandChildEvents?: (childEventsInfo: ChildEventsInfo) => void;
  onReRunChildEvents?: (childEventsInfo: ChildEventsInfo) => void;
  showItems?: boolean;
}

export const ChildEvents: React.FC<ChildEventsProps> = ({
  childEventsInfo,
  className = "mt-1 ml-3 text-gray-500",
  onReRunChildEvents,
  onExpandChildEvents,
  showItems = true,
}) => {
  const ChildEventsArrowIcon = React.useMemo(() => {
    return resolveIcon("corner-down-right");
  }, []);

  const ReRunChildEventsIcon = React.useMemo(() => {
    return resolveIcon("rotate-ccw");
  }, []);
  const ExpandIcon = React.useMemo(() => {
    return resolveIcon("expand");
  }, []);

  return (
    <div className={className}>
      <div className="flex items-center justify-between gap-2">
        <div className={"flex items-center gap-2 w-full"}>
          <ChildEventsArrowIcon size={18} className="text-gray-500" />
          <span className="text-sm">
            {childEventsInfo.count} child event{childEventsInfo.count === 1 ? "" : "s"} {childEventsInfo.state || ""}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {onExpandChildEvents && (
            <ExpandIcon
              size={16}
              className="text-gray-500 hover:text-gray-700 hover:scale-110 cursor-pointer mt-1"
              onClick={(e) => {
                e.stopPropagation();
                onExpandChildEvents(childEventsInfo);
              }}
            />
          )}
          {onReRunChildEvents && (
            <ReRunChildEventsIcon
              size={18}
              className="text-gray-500 hover:text-gray-700 hover:scale-110 cursor-pointer"
              onClick={(e) => {
                e.stopPropagation();
                onReRunChildEvents(childEventsInfo);
              }}
            />
          )}
        </div>
      </div>
      {showItems && childEventsInfo.items && childEventsInfo.items.length > 0 && (
        <div className="flex flex-col items-start justify-between pl-7 py-1 text-gray-600 w-full">
          {childEventsInfo.items.map((item, idx) => {
            const Icon =
              item.state === "processed"
                ? resolveIcon("check")
                : item.state === "discarded"
                ? resolveIcon("x")
                : resolveIcon("clock");
            const colorClass =
              item.state === "processed"
                ? "text-green-700"
                : item.state === "discarded"
                ? "text-red-700"
                : "text-blue-800";

            return (
              <div key={`${item.label}-${idx}`} className="flex justify-between items-center gap-3 py-1 w-full">
                <span className={`text-sm flex items-center gap-2 ${colorClass}`}>
                  <Icon size={16} />
                  {item.label}
                </span>
              </div>
            );
          })}
        </div>
      )}
      {showItems && childEventsInfo.waitingInfos && childEventsInfo.waitingInfos.length > 0 && (
        <div className="flex flex-col items-center justify-between pl-2 py-1 text-gray-500 w-full">
          {childEventsInfo.waitingInfos.map((waitingInfo) => {
            const Icon = resolveIcon(waitingInfo.icon);
            return (
              <div
                key={waitingInfo.info}
                className="flex justify-between items-center gap-3 pl-2 py-1 rounded-md w-full"
              >
                <span className="text-sm text-right flex items-center gap-2">
                  <Icon size={18} className="text-gray-500" />
                  {waitingInfo.info}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
