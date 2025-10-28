import React from "react";
import { MetadataItem, MetadataList } from "../metadataList";
import { resolveIcon } from "@/lib/utils";
import { EllipsisVertical, X } from "lucide-react"
import { ChildEvents, ChildEventsInfo } from "../childEvents";
import { ChildEventsState } from "../composite";
import { JSX } from "react";

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
  latestEvents: SidebarEvent[];
  nextInQueueEvents: SidebarEvent[];
  metadata: MetadataItem[];
  title: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;

  onExpandChildEvents?: () => void;
  onReRunChildEvents?: () => void;
}

export const ComponentSidebar = ({
  metadata,
  title,
  iconSrc,
  iconSlug,
  iconColor,
  iconBackground,
  onExpandChildEvents,
  onReRunChildEvents,
}: ComponentSidebarProps) => {
  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);
  const latestEvents: SidebarEvent[] = [
    {
      title: "New commit",
      subtitle: "main",
      state: "processed",
      isOpen: true,
      receivedAt: new Date(),
      childEventsInfo: {
        count: 1,
        state: "processed",
        waitingInfos: [],
      },
    },
    {
      title: "New commit",
      subtitle: "main",
      state: "processed",
      isOpen: false,
      receivedAt: new Date(),
      childEventsInfo: {
        count: 1,
        state: "processed",
        waitingInfos: [],
      },
    },
    {
      title: "New commit",
      subtitle: "main",
      state: "processed",
      isOpen: true,
      receivedAt: new Date(),
      values: {
        "Author": "Pedro Forestileao",
        "Commit": "New commit",
        "Branch": "main",
        "Type": "push",
        "Event ID": "123123123-123123123-123123123",
      },
      childEventsInfo: {
        count: 1,
        state: "processed",
        waitingInfos: [
          {
            icon: "check",
            info: "Processed",
          },
          {
            icon: "check",
            info: "Deploy",
          },
          {
            icon: "check",
            info: "Post-deploy verification",
          },
        ],
      },
    },
  ];

  const nextInQueueEvents: SidebarEvent[] = [
    {
      title: "New commit",
      subtitle: "main",
      state: "processed",
      isOpen: false,
      receivedAt: new Date(),
      childEventsInfo: {
        count: 1,
        state: "processed",
        waitingInfos: [],
      },
    }
  ];

  const createEventItem = (event: SidebarEvent, index: number): JSX.Element => {
    let EventIcon = resolveIcon("check")
    switch (event.state) {
      case "processed":
        EventIcon = resolveIcon("check")
        break;
      case "discarded":
        EventIcon = resolveIcon("x")
        break;
    }

    let EventColor = "text-green-700"
    switch (event.state) {
      case "processed":
        EventColor = "text-green-700"
        break;
      case "discarded":
        EventColor = "text-red-700"
        break;
    }

    let EventBackground = "bg-green-200"
    switch (event.state) {
      case "processed":
        EventBackground = "bg-green-200"
        break;
      case "discarded":
        EventBackground = "bg-red-200"
        break;
      case "waiting":
        EventBackground = "bg-orange-200"
        break;
    }

    let iconBorderColor = "border-gray-700"
    switch (event.state) {
      case "processed":
        iconBorderColor = "border-green-700"
        break;
      case "discarded":
        iconBorderColor = "border-red-700"
        break;
      case "waiting":
        iconBorderColor = "border-orange-700"
        break;
    }

    return (
      <div key={event.title + index} onClick={() => { }} className={`flex flex-col items-center justify-between gap-1 px-2 py-1.5 rounded-md cursor-pointer ${EventBackground} ${EventColor}`}>
        <div className="flex items-center gap-3 rounded-md w-full min-w-0">
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <div className={`w-4 h-4 flex-shrink-0 rounded-full flex items-center justify-center border-[1.5px] ${EventColor} ${iconBorderColor}`}>
              <EventIcon size={8} strokeWidth={3} className="thick   " />
            </div>
            <span className="truncate text-sm text-black font-medium">{event.title}</span>
          </div>
          {event.subtitle && (
            <span className="text-sm text-gray-500 truncate flex-shrink-0 max-w-[40%]">{event.subtitle}</span>
          )}
        </div>
        {event.isOpen && (
          (event.values && Object.entries(event.values).length > 0) ||
          (event.childEventsInfo && event.childEventsInfo.count > 0)
        ) && <div className="rounded-sm bg-white border-2 border-gray-300 text-gray-500 w-full">

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
              <div className={`w-full bg-gray-100 rounded-b-sm px-4 py-3 ${event.values && Object.entries(event.values).length > 0 ? "border-t-2 border-gray-300" : " rounded-t-sm"}`}>
                <ChildEvents
                  childEventsInfo={event.childEventsInfo}
                  onExpandChildEvents={onExpandChildEvents}
                  onReRunChildEvents={onReRunChildEvents}
                  className="font-medium"
                />
              </div>
            )}
          </div>}
      </div>
    )
  }

  return (
    <div className="min-w-[27rem] border-l-2 border-gray-400 border-border flex-1">
      <div className="flex items-center justify-between gap-3 p-3 relative border-b-2 border-gray-400 bg-gray-50">
        <div className="flex flex-col items-start gap-3 w-full mt-2">
          <div className={`w-8 h-8 rounded-full overflow-hidden flex items-center justify-center ${iconBackground || ''}`}>
            {iconSrc ? <img src={iconSrc} alt={title} className="w-7 h-7" /> : <Icon size={22} className={iconColor} />}
          </div>
          <div className="flex justify-between gap-3 w-full">
            <h2 className="text-xl font-semibold">{title}</h2>
            <button className="ml-auto">
              <EllipsisVertical size={16} />
            </button>
          </div>
          <div className="flex items-center justify-center gap-1 absolute top-6 right-2 text-xs font-medium cursor-pointer">
            <span>Close</span>
            <X size={14} />
          </div>
        </div>
      </div>
      <div className="px-3 py-1 border-b-2 border-gray-400">
        <MetadataList items={metadata} className="border-b-0 text-gray-500 font-medium gap-2 flex flex-col py-2 font-mono" />
      </div>
      <div className="px-3 py-1 border-b-2 border-gray-400">
        <h2 className="text-sm font-semibold uppercase text-gray-500 my-2">Latest events</h2>
        <div className="flex flex-col gap-2">
          {latestEvents.slice(0, 5).map((event, index) => {
            return createEventItem(event, index)
          })}
        </div>
      </div>
      <div className="px-3 py-1">
        <h2 className="text-sm font-semibold uppercase text-gray-500 my-2">Next in queue</h2>
        <div className="flex flex-col gap-2">
          {nextInQueueEvents.slice(0, 5).map((event, index) => {
            return createEventItem(event, index)
          })}
        </div>
      </div>
    </div>
  );
};