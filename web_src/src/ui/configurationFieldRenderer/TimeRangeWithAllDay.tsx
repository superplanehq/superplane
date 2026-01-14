import React, { useEffect } from "react";
import { Label } from "@/components/ui/label";
import { TimePickerField } from "./TimePickerField";

interface TimeRangeWithAllDayProps {
  startTime: string | undefined;
  endTime: string | undefined;
  onStartTimeChange: (value: string | undefined) => void;
  onEndTimeChange: (value: string | undefined) => void;
  onBothTimesChange?: (startTime: string | undefined, endTime: string | undefined) => void;
  hasError?: boolean;
  hasStartTimeError?: boolean;
  hasEndTimeError?: boolean;
  itemType?: string;
}

export const TimeRangeWithAllDay: React.FC<TimeRangeWithAllDayProps> = ({
  startTime,
  endTime,
  onStartTimeChange,
  onEndTimeChange,
  onBothTimesChange,
  hasError,
  hasStartTimeError,
  hasEndTimeError,
  itemType: _itemType,
}) => {
  // Initialize with default times (00:00 - 23:59) if no times are set
  useEffect(() => {
    if (
      (startTime === undefined || startTime === null || startTime === "") &&
      (endTime === undefined || endTime === null || endTime === "")
    ) {
      if (onBothTimesChange) {
        onBothTimesChange("00:00", "23:59");
      } else {
        onStartTimeChange("00:00");
        onEndTimeChange("23:59");
      }
    }
  }, [startTime, endTime, onStartTimeChange, onEndTimeChange, onBothTimesChange]);

  return (
    <div className="space-y-2">
      <Label className="block text-left">
        Active Time
        <span className="text-gray-800 dark:text-gray-300 ml-1">*</span>
      </Label>
      <div className="flex items-center gap-2">
        <TimePickerField
          field={{ name: "startTime", label: "Start Time", type: "time" } as any}
          value={startTime}
          onChange={(val) => onStartTimeChange(val as string | undefined)}
          hasError={hasError || hasStartTimeError}
          allValues={{ startTime, endTime }}
        />
        <div className="text-sm text-gray-500 dark:text-gray-400 font-medium">–</div>
        <TimePickerField
          field={{ name: "endTime", label: "End Time", type: "time" } as any}
          value={endTime}
          onChange={(val) => onEndTimeChange(val as string | undefined)}
          hasError={hasError || hasEndTimeError}
          allValues={{ startTime, endTime }}
        />
      </div>
    </div>
  );
};
