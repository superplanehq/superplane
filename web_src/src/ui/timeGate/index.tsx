import React from "react";
import { ComponentBase, EventSection } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";
import { calcRelativeTimeFromDiff } from "@/lib/utils";
import { MetadataItem } from "../metadataList";

export type TimeGateState = "success" | "failed" | "running";

export interface TimeGateExecutionItem {
  title: string;
  receivedAt?: Date;
  state?: TimeGateState;
  values?: Record<string, string>;
  nextRunTime?: Date;
}

export interface NextInQueueItem {
  title: string;
}

export interface TimeGateProps extends ComponentActionsProps {
  title?: string;
  mode?: "include_range" | "exclude_range" | "include_specific" | "exclude_specific";
  timeWindow?: string;
  days?: string;
  timezone?: string;
  lastExecution?: TimeGateExecutionItem;
  nextInQueue?: NextInQueueItem;
  collapsed?: boolean;
  selected?: boolean;
  collapsedBackground?: string;
  iconBackground?: string;
  iconColor?: string;
  headerColor?: string;
  hideLastRun?: boolean;
}

const daysOfWeekOrder = { "monday": 1, "tuesday": 2, "wednesday": 3, "thursday": 4, "friday": 5, "saturday": 6, "sunday": 7 };

export const TimeGate: React.FC<TimeGateProps> = ({
  title = "Time Gate",
  mode = "include_range",
  timeWindow,
  days,
  timezone,
  lastExecution,
  nextInQueue,
  collapsed = false,
  selected = false,
  collapsedBackground,
  iconBackground,
  iconColor,
  headerColor,
  hideLastRun = false,
  onRun,
  onEdit,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {

  const spec = days ? {
    title: "day",
    tooltipTitle: "Days of the week",
    values: [
      ...days.split(",").sort((a, b) => daysOfWeekOrder[a.trim() as keyof typeof daysOfWeekOrder] - daysOfWeekOrder[b.trim() as keyof typeof daysOfWeekOrder]).map(day => ({
        badges: [
          {
            label: day.trim(),
            bgColor: "bg-gray-100",
            textColor: "text-gray-700"
          }
        ]
      }))
    ]
  } : undefined;

  const getModeLabel = (mode: string) => {
    switch (mode) {
      case "include_range":
        return "Include Range";
      case "exclude_range":
        return "Exclude Range";
      case "include_specific":
        return "Include Specific";
      case "exclude_specific":
        return "Exclude Specific";
      default:
        return mode.charAt(0).toUpperCase() + mode.slice(1).replace(/_/g, ' ');
    }
  };

  const getMetadataItems = () => {
    const items: MetadataItem[] = [
      {
        icon: "settings",
        label: getModeLabel(mode)
      }
    ];

    if (timeWindow) {
      items.push({
        icon: "clock",
        label: timeWindow
      });
    }

    // Add timezone if provided
    if (timezone) {
      items.push({
        icon: "globe",
        label: `Timezone: ${timezone}`
      });
    }

    return items;
  };

  const metadata: MetadataItem[] = getMetadataItems();

  const eventSections: EventSection[] = [];

  if (!hideLastRun) {
    if (lastExecution && lastExecution.state && lastExecution.receivedAt) {
      let eventTitle: string;
      let eventSubtitle: string | undefined;

      // Use trigger-based title
      eventTitle = lastExecution.title;

      if (lastExecution.state === "running" && lastExecution.nextRunTime) {
        // Show time remaining for running state
        const now = new Date();
        const timeDiff = lastExecution.nextRunTime.getTime() - now.getTime();
        const timeLeftText = timeDiff > 0 ? calcRelativeTimeFromDiff(timeDiff) : "Ready to run";
        eventSubtitle = `Runs in ${timeLeftText}`;
      }

      eventSections.push({
        title: "LAST EVENT",
        receivedAt: lastExecution.receivedAt,
        eventState: lastExecution.state,
        eventTitle,
        eventSubtitle,
      });
    } else {
      // Show placeholder if no events
      eventSections.push({
        title: "LAST EVENT",
        eventState: "neutral" as const,
        eventTitle: "No events received yet",
      });
    }
  }

  if (nextInQueue) {
    eventSections.push({
      title: "NEXT IN QUEUE",
      eventState: "next-in-queue",
      eventTitle: nextInQueue.title,
    });
  }

  return (
    <ComponentBase
      iconSlug="clock"
      iconBackground={iconBackground || "bg-blue-100"}
      iconColor={iconColor || "text-blue-600"}
      headerColor={headerColor || "bg-blue-50"}
      title={title}
      spec={spec}
      eventSections={eventSections}
      collapsed={collapsed}
      selected={selected}
      collapsedBackground={collapsedBackground}
      onRun={onRun}
      onEdit={onEdit}
      onDuplicate={onDuplicate}
      onDeactivate={onDeactivate}
      onToggleView={onToggleView}
      onDelete={onDelete}
      isCompactView={isCompactView}
      metadata={metadata}
    />
  );
};