import { resolveIcon } from "@/lib/utils";
import { TextAlignStart, X } from "lucide-react";
import React, { JSX, useCallback, useEffect, useRef, useState } from "react";
import { ChildEvents, ChildEventsInfo } from "../childEvents";
import { ChildEventsState } from "../composite";
import { MetadataItem, MetadataList } from "../metadataList";
import { SidebarActionsDropdown } from "./SidebarActionsDropdown";

interface SidebarEvent {
  id: string;
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
  hideQueueEvents?: boolean;

  onExpandChildEvents?: (childEventsInfo: ChildEventsInfo) => void;
  onReRunChildEvents?: (childEventsInfo: ChildEventsInfo) => void;
  onEventClick?: (event: SidebarEvent) => void;
  onClose?: () => void;
  onSeeFullHistory?: () => void;

  // Action handlers
  onRun?: () => void;
  runDisabled?: boolean;
  runDisabledTooltip?: string;
  onDuplicate?: () => void;
  onDocs?: () => void;
  onEdit?: () => void;
  onConfigure?: () => void;
  onDeactivate?: () => void;
  onToggleView?: () => void;
  onDelete?: () => void;
  isCompactView?: boolean;
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
  hideQueueEvents = false,
  onSeeFullHistory,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onDuplicate,
  onDocs,
  onEdit,
  onConfigure,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView = false,
}: ComponentSidebarProps) => {
  const [sidebarWidth, setSidebarWidth] = useState(420);
  const [isResizing, setIsResizing] = useState(false);
  const sidebarRef = useRef<HTMLDivElement>(null);

  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) return;

      const newWidth = window.innerWidth - e.clientX;
      // Set min width to 320px and max width to 800px
      const clampedWidth = Math.max(320, Math.min(800, newWidth));
      setSidebarWidth(clampedWidth);
    },
    [isResizing],
  );

  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (isResizing) {
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "ew-resize";
      document.body.style.userSelect = "none";

      return () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
      };
    }
  }, [isResizing, handleMouseMove, handleMouseUp]);

  const createEventItem = (event: SidebarEvent, index: number): JSX.Element => {
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
        EventIcon = resolveIcon("refresh-cw");
        EventColor = "text-blue-700";
        EventBackground = "bg-blue-100";
        iconBorderColor = "";
        iconSize = 17;
        iconContainerSize = 5;
        iconStrokeWidth = 2;
        animation = "animate-spin";
        break;
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
        onClick={(e) => {
          e.stopPropagation();
          onEventClick?.(event);
        }}
        className={`flex flex-col items-center justify-between gap-1 px-2 py-1.5 rounded-md cursor-pointer ${EventBackground} ${EventColor}`}
      >
        <div className="flex items-center gap-3 rounded-md w-full min-w-0">
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <div
              className={`w-${iconContainerSize} h-${iconContainerSize} flex-shrink-0 rounded-full flex items-center justify-center border-[1.5px] ${EventColor} ${iconBorderColor} ${animation}`}
            >
              <EventIcon size={iconSize} strokeWidth={iconStrokeWidth} className="thick   " />
            </div>
            <span className="truncate text-sm text-black font-medium">{event.title}</span>
          </div>
          {event.subtitle && (
            <span className="text-sm text-gray-500 truncate flex-shrink-0 max-w-[40%]">{event.subtitle}</span>
          )}
        </div>
        {event.isOpen &&
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

  if (!isOpen) return null;

  return (
    <div
      ref={sidebarRef}
      className="border-l-1 border-gray-200 border-border absolute right-0 top-0 h-full z-20 overflow-y-auto overflow-x-hidden bg-white shadow-2xl"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
    >
      {/* Resize handle */}
      <div
        onMouseDown={handleMouseDown}
        className={`absolute left-0 top-0 bottom-0 w-4 cursor-ew-resize hover:bg-blue-50 transition-colors flex items-center justify-center group ${
          isResizing ? "bg-blue-50" : ""
        }`}
        style={{ marginLeft: "-8px" }}
      >
        <div
          className={`w-1 h-12 rounded-full bg-gray-300 group-hover:bg-blue-500 transition-colors ${
            isResizing ? "bg-blue-500" : ""
          }`}
        />
      </div>
      <div className="flex items-center justify-between gap-3 p-3 relative border-b-1 border-gray-200 bg-gray-50">
        <div className="flex flex-col items-start gap-3 w-full mt-2">
          <div
            className={`w-8 h-8 rounded-full overflow-hidden flex items-center justify-center ${iconBackground || ""}`}
          >
            {iconSrc ? <img src={iconSrc} alt={title} className="w-7 h-7" /> : <Icon size={22} className={iconColor} />}
          </div>
          <div className="flex justify-between gap-3 w-full">
            <h2 className="text-xl font-semibold">{title}</h2>
            <SidebarActionsDropdown
              onRun={onRun}
              runDisabled={runDisabled}
              runDisabledTooltip={runDisabledTooltip}
              onDuplicate={onDuplicate}
              onDocs={onDocs}
              onEdit={onEdit}
              onConfigure={onConfigure}
              onDeactivate={onDeactivate}
              onToggleView={onToggleView}
              onDelete={onDelete}
              isCompactView={isCompactView}
            />
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
      {metadata.length > 0 && (
        <div className="px-3 py-1 border-b-1 border-gray-200">
          <MetadataList
            items={metadata}
            className="border-b-0 text-gray-500 font-medium gap-2 flex flex-col py-2 font-mono"
          />
        </div>
      )}
      <div className="px-3 py-1 border-b-1 border-gray-200 pb-3 text-left">
        <h2 className="text-sm font-semibold uppercase text-gray-500 my-2">Latest events</h2>
        <div className="flex flex-col gap-2">
          {latestEvents.length === 0 ? (
            <div className="text-center py-4 text-gray-500 text-sm">No events found</div>
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
      {!hideQueueEvents && (
        <div className="px-3 py-1 pb-3 text-left">
          <h2 className="text-sm font-semibold uppercase text-gray-500 my-2">Next in queue</h2>
          <div className="flex flex-col gap-2">
            {nextInQueueEvents.length === 0 ? (
              <div className="text-center py-4 text-gray-500 text-sm">Queue is empty</div>
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
      )}
    </div>
  );
};
