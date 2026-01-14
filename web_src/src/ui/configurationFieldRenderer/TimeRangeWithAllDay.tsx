import React, { useEffect, useMemo } from "react";
import { Input } from "@/components/ui/input";
import { Switch } from "@/ui/switch";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

interface TimeRangeWithAllDayProps {
  startTime: string | undefined;
  endTime: string | undefined;
  onStartTimeChange: (value: string | undefined) => void;
  onEndTimeChange: (value: string | undefined) => void;
  onBothTimesChange?: (startTime: string | undefined, endTime: string | undefined) => void;
  hasError?: boolean;
}

export const TimeRangeWithAllDay: React.FC<TimeRangeWithAllDayProps> = ({
  startTime,
  endTime,
  onStartTimeChange,
  onEndTimeChange,
  onBothTimesChange,
  hasError,
}) => {
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

  return (
    <div className="space-y-3">
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
        <div className="flex gap-3 items-center">
          <div className="flex-1">
            <Label className="text-sm text-gray-600 dark:text-gray-400 mb-1 block">Start Time</Label>
            <Input
              type="time"
              value={startTime || ""}
              onChange={(e) => onStartTimeChange(e.target.value || undefined)}
              className={cn(hasError && "border-red-500 border-2")}
              min="00:00"
              max={endTime || "23:59"}
            />
          </div>
          <div className="flex-1">
            <Label className="text-sm text-gray-600 dark:text-gray-400 mb-1 block">End Time</Label>
            <Input
              type="time"
              value={endTime || ""}
              onChange={(e) => onEndTimeChange(e.target.value || undefined)}
              className={cn(hasError && "border-red-500 border-2")}
              min={startTime || "00:00"}
              max="23:59"
            />
          </div>
        </div>
      )}
    </div>
  );
};
