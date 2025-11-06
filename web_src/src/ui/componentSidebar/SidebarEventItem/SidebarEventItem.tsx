import { resolveIcon } from "@/lib/utils";
import React from "react";
import { ChildEvents, ChildEventsInfo } from "../../childEvents";
import { SidebarEvent } from "../types";

interface SidebarEventItemProps {
  event: SidebarEvent;
  index: number;
  variant?: 'latest' | 'queue';
  isOpen: boolean;
  onToggleOpen: (eventId: string) => void;
  onExpandChildEvents?: (childEventsInfo: ChildEventsInfo) => void;
  onReRunChildEvents?: (childEventsInfo: ChildEventsInfo) => void;
  onEventClick?: (event: SidebarEvent) => void;
}

export const SidebarEventItem: React.FC<SidebarEventItemProps> = ({
  event,
  index,
  variant = 'latest',
  isOpen,
  onToggleOpen,
  onExpandChildEvents,
  onReRunChildEvents,
  onEventClick,
}) => {
  let EventIcon = resolveIcon("check");
  let EventColor = "text-green-700";
  let EventBackground = "bg-green-200";
  let iconBorderColor = "border-gray-700";
  let iconSize = 8;
  let iconContainerSize = 4;
  let iconStrokeWidth = 3;
  let animation = "";

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
      if (variant === 'queue') {
        // Match node card styling (neutral grey + dashed icon)
        EventIcon = resolveIcon("circle-dashed");
        EventColor = "text-gray-500";
        EventBackground = "bg-gray-100";
        iconBorderColor = "";
        iconSize = 20;
        iconContainerSize = 5;
        iconStrokeWidth = 2;
        animation = "";
      } else {
        EventIcon = resolveIcon("refresh-cw");
        EventColor = "text-blue-700";
        EventBackground = "bg-blue-100";
        iconBorderColor = "";
        iconSize = 17;
        iconContainerSize = 5;
        iconStrokeWidth = 2;
        animation = "animate-spin";
      }
      break;
    case "running":
      EventIcon = resolveIcon("refresh-cw");
      EventColor = "text-blue-700";
      EventBackground = "bg-blue-100";
      iconBorderColor = "";
      iconSize = 17;
      iconContainerSize = 5;
      iconStrokeWidth = 2;
      animation = "animate-spin";
      break;
  }

  return (
    <div
      key={event.title + index}
      className={`flex flex-col items-center justify-between gap-1 px-2 py-1.5 rounded-md ${EventBackground} ${EventColor}`}
    >
      <div
        className="flex items-center gap-3 rounded-md w-full min-w-0 cursor-pointer"
        onClick={(e) => {
          e.stopPropagation();
          onToggleOpen(event.id);
          onEventClick?.(event);
        }}
      >
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <div
            className={`w-${iconContainerSize} h-${iconContainerSize} flex-shrink-0 rounded-full flex items-center justify-center border-[1.5px] ${EventColor} ${iconBorderColor} ${animation}`}
          >
            <EventIcon size={iconSize} strokeWidth={iconStrokeWidth} className="thick" />
          </div>
          <span className="truncate text-sm text-black font-medium">{event.title}</span>
        </div>
        {event.subtitle && (
          <span className="text-sm text-gray-500 truncate flex-shrink-0 max-w-[40%]">{event.subtitle}</span>
        )}
      </div>
      {isOpen &&
        ((event.values && Object.entries(event.values).length > 0) ||
          (event.childEventsInfo && event.childEventsInfo.count > 0)) && (
          <div className="rounded-sm bg-white border-1 border-gray-200 text-gray-500 w-full">
            {event.values && Object.entries(event.values).length > 0 && (
              <div className="w-full flex flex-col gap-1 items-center justify-between mt-1 px-2 py-2">
                {Object.entries(event.values || {}).map(([key, value]) => (
                  <div key={key} className="flex items-center gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                    <span className="text-sm flex-shrink-0 text-right w-[25%]">{key}:</span>
                    <span className="text-sm flex-1 truncate text-left w-[75%] hover:underline">{value}</span>
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