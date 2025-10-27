import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import React from "react";

type ChildEventsState = "processed" | "discarded" | "waiting" | "running";

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

export interface ChildEventsProps {
  childEventsInfo: ChildEventsInfo;
  className?: string;
  onExpandChildEvents?: () => void;
  onReRunChildEvents?: () => void;
}

export const ChildEvents: React.FC<ChildEventsProps> = ({
  childEventsInfo,
  className = "mt-1 ml-3 text-gray-500",
  onExpandChildEvents,
  onReRunChildEvents,
}) => {
  const [showWaiting, setShowWaiting] = React.useState(false);

  const ChildEventsArrowIcon = React.useMemo(() => {
    return resolveIcon("corner-down-right");
  }, []);

  const ExpandChildEventsIcon = React.useMemo(() => {
    return resolveIcon("expand");
  }, []);

  const ReRunChildEventsIcon = React.useMemo(() => {
    return resolveIcon("rotate-ccw");
  }, []);

  const hasWaitingInfos = (childEventsInfo?.waitingInfos?.length || 0) > 0;

  const toggleWaitingInfos = () => {
    setShowWaiting(!showWaiting);
  };

  return (
    <div className={className}>
      <div className="flex items-center justify-between gap-2">
        <div
          onClick={hasWaitingInfos ? toggleWaitingInfos : undefined}
          className={
            "flex items-center gap-2 w-full " +
            (hasWaitingInfos ? "cursor-pointer hover:text-gray-700 hover:scale-102 transition-all" : "")
          }
        >
          <ChildEventsArrowIcon size={18} className="text-gray-500" />
          <span className="text-sm">
            {childEventsInfo.count} child event{childEventsInfo.count === 1 ? "" : "s"}{" "}
            {childEventsInfo.state || ""}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {onExpandChildEvents && (
            <ExpandChildEventsIcon
              size={18}
              className="text-gray-500 hover:text-gray-700 hover:scale-110 cursor-pointer"
              onClick={onExpandChildEvents}
            />
          )}
          {onReRunChildEvents && (
            <ReRunChildEventsIcon
              size={18}
              className="text-gray-500 hover:text-gray-700 hover:scale-110 cursor-pointer"
              onClick={onReRunChildEvents}
            />
          )}
        </div>
      </div>
      {hasWaitingInfos && showWaiting && (
        <div className="flex flex-col items-center justify-between pl-2 py-1 rounded-md bg-white text-gray-500 w-full">
          {childEventsInfo.waitingInfos.map((waitingInfo) => {
            const Icon = resolveIcon(waitingInfo.icon);
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
            );
          })}
        </div>
      )}
    </div>
  );
};