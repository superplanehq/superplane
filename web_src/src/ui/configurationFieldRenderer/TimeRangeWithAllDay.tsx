import React, { useEffect, useMemo } from "react";
import { Switch } from "@/ui/switch";
import { Label } from "@/components/ui/label";
import { TimePickerField } from "./TimePickerField";

interface TimeRangeWithAllDayProps {
  startTime: string | undefined;
  endTime: string | undefined;
  onStartTimeChange: (value: string | undefined) => void;
  onEndTimeChange: (value: string | undefined) => void;
  onBothTimesChange?: (startTime: string | undefined, endTime: string | undefined) => void;
  hasError?: boolean;
  itemType?: string; // "weekly" or "specific_dates"
}

export const TimeRangeWithAllDay: React.FC<TimeRangeWithAllDayProps> = ({
  startTime,
  endTime,
  onStartTimeChange,
  onEndTimeChange,
  onBothTimesChange,
  hasError,
  itemType,
}) => {
  const isWeekly = itemType === "weekly";
  // Check if "All day" is enabled (both times are set to full day range)
  const isAllDay = useMemo(() => {
    return startTime === "00:00" && endTime === "23:59";
  }, [startTime, endTime]);

  // Initialize with "All day" ON by default if no times are set
  useEffect(() => {
    if ((startTime === undefined || startTime === null || startTime === "") &&
        (endTime === undefined || endTime === null || endTime === "")) {
      onStartTimeChange("00:00");
      onEndTimeChange("23:59");
    }
  }, [startTime, endTime, onStartTimeChange, onEndTimeChange]);

  const handleAllDayToggle = (checked: boolean) => {
    if (checked) {
      // Turn ON "All day" - set to full day range
      // Update both times together atomically to ensure state consistency
      if (onBothTimesChange) {
        onBothTimesChange("00:00", "23:59");
      } else {
        onStartTimeChange("00:00");
        onEndTimeChange("23:59");
      }
    } else {
      // Turn OFF "All day" - set to default working hours if currently all day
      if (startTime === "00:00" && endTime === "23:59") {
        if (onBothTimesChange) {
          onBothTimesChange("09:00", "17:00");
        } else {
          onStartTimeChange("09:00");
          onEndTimeChange("17:00");
        }
      }
      // If times are already set to something else, keep them as is
    }
  };

  // For weekly type, show "All day" at the bottom; for specific_dates, show at top
  if (isWeekly) {
    return (
      <div className="flex items-center gap-3">
        {/* All day toggle - on the left for weekly */}
        <div className="flex items-center gap-3">
          <Switch
            checked={isAllDay}
            onCheckedChange={handleAllDayToggle}
            id="all-day-toggle"
          />
          <Label htmlFor="all-day-toggle" className="cursor-pointer">
            All day
          </Label>
        </div>

        {/* Time inputs - only show when "All day" is OFF, on the right */}
        {!isAllDay && (
          <div className="flex items-center gap-0 flex-1">
            <div className="flex-[0.5]">
              <TimePickerField
                field={{ name: "startTime", label: "Start Time", type: "time" } as any}
                value={startTime}
                onChange={(val) => onStartTimeChange(val as string | undefined)}
                hasError={hasError}
                allValues={{ startTime, endTime }}
                className="rounded-r-none border-r-0"
              />
            </div>
            <div className="px-3 py-2 text-sm bg-gray-50 dark:bg-gray-800 text-gray-500 dark:text-gray-400 border border-gray-300 dark:border-gray-600 border-l-0 border-r-0 flex items-center justify-center font-medium">
              -
            </div>
            <div className="flex-[0.5]">
              <TimePickerField
                field={{ name: "endTime", label: "End Time", type: "time" } as any}
                value={endTime}
                onChange={(val) => onEndTimeChange(val as string | undefined)}
                hasError={hasError}
                allValues={{ startTime, endTime }}
                className="rounded-l-none"
              />
            </div>
          </div>
        )}
      </div>
    );
  }

  // For specific_dates, keep original layout (All day at top)
  return (
    <div className="flex items-center gap-3">
      {/* All day toggle */}
      <div className="flex items-center gap-3">
        <Switch
          checked={isAllDay}
          onCheckedChange={handleAllDayToggle}
          id="all-day-toggle"
        />
        <Label htmlFor="all-day-toggle" className="cursor-pointer">
          All day
        </Label>
      </div>

      {/* Time inputs - only show when "All day" is OFF */}
      {!isAllDay && (
        <div className="flex items-center gap-0 flex-1">
          <div className="flex-[0.5]">
            <TimePickerField
              field={{ name: "startTime", label: "Start Time", type: "time" } as any}
              value={startTime}
              onChange={(val) => onStartTimeChange(val as string | undefined)}
              hasError={hasError}
              allValues={{ startTime, endTime }}
              className="rounded-r-none border-r-0"
            />
          </div>
          <div className="px-3 py-2 text-sm bg-gray-50 dark:bg-gray-800 text-gray-500 dark:text-gray-400 border border-gray-300 dark:border-gray-600 border-l-0 border-r-0 flex items-center justify-center font-medium">
            -
          </div>
          <div className="flex-[0.5]">
            <TimePickerField
              field={{ name: "endTime", label: "End Time", type: "time" } as any}
              value={endTime}
              onChange={(val) => onEndTimeChange(val as string | undefined)}
              hasError={hasError}
              allValues={{ startTime, endTime }}
              className="rounded-l-none"
            />
          </div>
        </div>
      )}
    </div>
  );
};
