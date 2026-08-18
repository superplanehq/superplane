import { Timestamp } from "@/components/Timestamp";
import { withEventSectionDarkBackground } from "@/lib/eventSectionBackground";
import { withEventStatusBadgeClasses } from "@/lib/eventStatusBadge";
import { eventSectionMetadataTextClassName } from "@/lib/nodeCanvasSections";
import { calcRelativeTimeFromDiff, cn } from "@/lib/utils";
import React from "react";
import type { EventSection, EventStateMap } from "./eventState";
import { DEFAULT_EVENT_STATE_MAP } from "./defaultEventStateMap";

interface EventSectionDisplayProps {
  section: EventSection;
  index: number;
  totalSections: number;
  className?: string;
  stateMap?: EventStateMap;
  lastSection?: boolean;
}

export const EventSectionDisplay: React.FC<EventSectionDisplayProps> = ({
  section,
  index,
  totalSections,
  className,
  stateMap = DEFAULT_EVENT_STATE_MAP,
  lastSection = false,
}) => {
  // Live timer for running executions
  const [liveDuration, setLiveDuration] = React.useState<number | null>(null);

  React.useEffect(() => {
    if (section.eventState === "running" && section.receivedAt) {
      const receivedAt = section.receivedAt;

      // Calculate initial duration
      setLiveDuration(Date.now() - receivedAt.getTime());

      // Update every second
      const interval = setInterval(() => {
        setLiveDuration(Date.now() - receivedAt.getTime());
      }, 1000);

      return () => clearInterval(interval);
    } else {
      setLiveDuration(null);
    }
  }, [section.eventState, section.receivedAt]);

  const currentState = section.eventState || "neutral";
  const stateStyle = stateMap[currentState] || stateMap["neutral"];

  const LastEventBackground = withEventSectionDarkBackground(stateStyle.backgroundColor);
  const LastEventStateColor = withEventStatusBadgeClasses(stateStyle.badgeColor);
  const durationText = liveDuration !== null ? calcRelativeTimeFromDiff(liveDuration) : "";

  return (
    <div
      key={index}
      className={
        `px-2 pt-2 relative ${lastSection ? "rounded-b-md" : ""} ${LastEventBackground}` +
        (index < totalSections - 1 ? " border-b border-slate-950/20 dark:border-gray-600/70" : "") +
        ` ${className}`
      }
    >
      <div className="flex items-center justify-between gap-2 min-w-0 flex-1">
        <div
          className={`uppercase text-[11px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white ${LastEventStateColor}`}
        >
          <span>{stateStyle.label || currentState}</span>
        </div>
        {section.eventSubtitle ? (
          <span
            title={typeof section.eventSubtitle === "string" ? section.eventSubtitle : undefined}
            className={cn(
              "text-[13px] font-medium truncate flex-shrink-0 max-w-[65%]",
              eventSectionMetadataTextClassName,
            )}
          >
            {section.showAutomaticTime && durationText ? durationText : section.eventSubtitle}
          </span>
        ) : (
          <span
            className={cn(
              "text-[13px] font-medium truncate flex-shrink-0 max-w-[65%]",
              eventSectionMetadataTextClassName,
            )}
          >
            <Timestamp date={section.receivedAt} display="relative" relativeStyle="abbreviated" />
          </span>
        )}
      </div>
      <div className="flex justify-left items-center mt-1 gap-2">
        {section.eventId && (
          <span className={cn("text-[13px] font-mono", eventSectionMetadataTextClassName)}>
            #{section.eventId?.slice(0, 4)}
          </span>
        )}
        <span className="text-sm text-gray-700 font-inter truncate text-md min-w-0 font-medium truncate dark:text-white/70">
          {section.eventTitle}
        </span>
      </div>
    </div>
  );
};
