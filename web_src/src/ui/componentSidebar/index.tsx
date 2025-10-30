import { resolveIcon } from "@/lib/utils";
import { EllipsisVertical, TextAlignStart, X } from "lucide-react";
import React, { JSX } from "react";
import { ChildEvents, ChildEventsInfo } from "../childEvents";
import { ChildEventsState } from "../composite";
import { MetadataItem, MetadataList } from "../metadataList";

interface SidebarEvent {
  title: string;
  subtitle?: string;
  state: ChildEventsState;
  isOpen: boolean;
  receivedAt?: Date;
  values?: Record<string, string>;
  childEventsInfo?: ChildEventsInfo;
}

interface ComponentSidebarProps {
  isOpen?: boolean;
  setIsOpen?: (isOpen: boolean) => void;

  latestEvents: SidebarEvent[];
  nextInQueueEvents: SidebarEvent[];
  metadata: MetadataItem[];
  title: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  moreInQueueCount: number;

  onExpandChildEvents?: (childEventsInfo: ChildEventsInfo) => void;
  onReRunChildEvents?: (childEventsInfo: ChildEventsInfo) => void;
  onEventClick?: (event: SidebarEvent) => void;
  onClose?: () => void;
  onSeeFullHistory?: () => void;
}

export const ComponentSidebar = ({
  isOpen,
  metadata,
  title,
  iconSrc,
  iconSlug,
  iconColor,
  iconBackground,
  onExpandChildEvents,
  onReRunChildEvents,
  onEventClick,
  onClose,
  latestEvents,
  nextInQueueEvents,
  moreInQueueCount = 0,
  onSeeFullHistory,
}: ComponentSidebarProps) => {
  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  const createEventItem = (event: SidebarEvent, index: number): JSX.Element => {
    let EventIcon = resolveIcon("check");
    let EventColor = "text-green-700";
    let EventBackground = "bg-green-200";
    let iconBorderColor = "border-gray-700";
    let iconSize = 8;
    let iconContainerSize = 4;
    let iconStrokeWidth = 3;

    switch (event.state) {
      case "processed":
        EventIcon = resolveIcon("check");
        EventColor = "text-green-700";
        EventBackground = "bg-green-200";
        iconBorderColor = "border-green-700";
        iconSize = 8;
        break;
      case "discarded":
        EventIcon = resolveIcon("x");
        EventColor = "text-red-700";
        EventBackground = "bg-red-200";
        iconBorderColor = "border-red-700";
        iconSize = 8;
        break;
      case "waiting":
        EventIcon = resolveIcon("circle-dashed");
        EventColor = "text-orange-700";
        EventBackground = "bg-orange-200";
        iconBorderColor = "";
        iconSize = 17;
        iconContainerSize = 5;
        iconStrokeWidth = 2;
        break;
    }

    return (
      <div
        key={event.title + index}
        onClick={(e) => {
          e.stopPropagation();
          onEventClick?.(event);
        }}
        className={`flex flex-col items-center justify-between gap-1 px-2 py-1.5 rounded-md cursor-pointer ${EventBackground} ${EventColor}`}
      >
        <div className="flex items-center gap-3 rounded-md w-full min-w-0">
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <div
              className={`w-${iconContainerSize} h-${iconContainerSize} flex-shrink-0 rounded-full flex items-center justify-center border-[1.5px] ${EventColor} ${iconBorderColor}`}
            >
              <EventIcon
                size={iconSize}
                strokeWidth={iconStrokeWidth}
                className="thick   "
              />
            </div>
            <span className="truncate text-sm text-black font-medium">
              {event.title}
            </span>
          </div>
          {event.subtitle && (
            <span className="text-sm text-gray-500 truncate flex-shrink-0 max-w-[40%]">
              {event.subtitle}
            </span>
          )}
        </div>
        {event.isOpen &&
          ((event.values && Object.entries(event.values).length > 0) ||
            (event.childEventsInfo && event.childEventsInfo.count > 0)) && (
            <div className="rounded-sm bg-white border-1 border-gray-200 text-gray-500 w-full">
              {event.values && Object.entries(event.values).length > 0 && (
                <div className="w-full flex flex-col gap-1 items-center justify-between mt-1 px-2 py-2">
                  {Object.entries(event.values || {}).map(([key, value]) => (
                    <div
                      key={key}
                      className="flex items-center gap-1 px-2 rounded-md w-full min-w-0 font-medium"
                    >
                      <span className="text-sm flex-shrink-0 text-right w-[25%]">
                        {key}:
                      </span>
                      <span className="text-sm flex-1 truncate text-left w-[75%] hover:underline">
                        {value}
                      </span>
                    </div>
                  ))}
                </div>
              )}

              {event.childEventsInfo && event.childEventsInfo.count > 0 && (
                <div
                  className={`w-full bg-gray-100 rounded-b-sm px-4 py-3 ${
                    event.values && Object.entries(event.values).length > 0
                      ? "border-t-1 border-gray-200"
                      : " rounded-t-sm"
                  }`}
                >
                  <ChildEvents
                    childEventsInfo={event.childEventsInfo}
                    onExpandChildEvents={onExpandChildEvents}
                    onReRunChildEvents={onReRunChildEvents}
                    className="font-medium"
                  />
                </div>
              )}
            </div>
          )}
      </div>
    );
  };

  if (!isOpen) return null;

  return (
    <div className="min-w-[27rem] border-l-1 border-t-1 border-gray-200 border-border flex-1 absolute right-0 top-[48px] h-full z-20 overflow-y-auto bg-white shadow-xl">
      <div className="flex items-center justify-between gap-3 p-3 relative border-b-1 border-gray-200 bg-gray-50">
        <div className="flex flex-col items-start gap-3 w-full mt-2">
          <div
            className={`w-8 h-8 rounded-full overflow-hidden flex items-center justify-center ${
              iconBackground || ""
            }`}
          >
            {iconSrc ? (
              <img src={iconSrc} alt={title} className="w-7 h-7" />
            ) : (
              <Icon size={22} className={iconColor} />
            )}
          </div>
          <div className="flex justify-between gap-3 w-full">
            <h2 className="text-xl font-semibold">{title}</h2>
            <button className="ml-auto">
              <EllipsisVertical size={16} />
            </button>
          </div>
          <div
            onClick={() => onClose?.()}
            className="flex items-center justify-center gap-1 absolute top-6 right-2 text-xs font-medium cursor-pointer"
          >
            <span>Close</span>
            <X size={14} />
          </div>
        </div>
      </div>
      <div className="px-3 py-1 border-b-1 border-gray-200">
        <MetadataList
          items={metadata}
          className="border-b-0 text-gray-500 font-medium gap-2 flex flex-col py-2 font-mono"
        />
      </div>
      <div className="px-3 py-1 border-b-1 border-gray-200 pb-3">
        <h2 className="text-sm font-semibold uppercase text-gray-500 my-2">
          Latest events
        </h2>
        <div className="flex flex-col gap-2">
          {latestEvents.length === 0 ? (
            <div className="text-center py-4 text-gray-500 text-sm">
              No events found
            </div>
          ) : (
            <>
              {latestEvents.slice(0, 5).map((event, index) => {
                return createEventItem(event, index);
              })}
              {moreInQueueCount > 0 && (
                <button
                  onClick={() => onSeeFullHistory?.()}
                  className="text-xs font-medium text-gray-500 hover:underline flex items-center gap-1 px-2 py-1"
                >
                  <TextAlignStart size={16} />
                  See full history
                </button>
              )}
            </>
          )}
        </div>
      </div>
      <div className="px-3 py-1 pb-3">
        <h2 className="text-sm font-semibold uppercase text-gray-500 my-2">
          Next in queue
        </h2>
        <div className="flex flex-col gap-2">
          {nextInQueueEvents.length === 0 ? (
            <div className="text-center py-4 text-gray-500 text-sm">
              Queue is empty
            </div>
          ) : (
            <>
              {nextInQueueEvents.slice(0, 5).map((event, index) => {
                return createEventItem(event, index);
              })}
              {moreInQueueCount > 0 && (
                <button
                  onClick={() => onSeeFullHistory?.()}
                  className="text-xs font-medium text-gray-500 hover:underline flex items-center gap-1 px-2 py-1"
                >
                  <TextAlignStart size={16} />
                  {moreInQueueCount} more in the queue
                </button>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};
