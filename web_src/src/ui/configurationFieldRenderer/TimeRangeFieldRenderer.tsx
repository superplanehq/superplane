import React, { useMemo } from "react";
import { FieldRendererProps } from "./types";
import { TimeFieldRenderer } from "./TimeFieldRenderer";

const DEFAULT_RANGE = "00:00-23:59";

const parseTimeRange = (value: string | undefined): { start: string; end: string } => {
  if (!value) return { start: "00:00", end: "23:59" };

  const parts = value.split("-");
  if (parts.length !== 2) return { start: "00:00", end: "23:59" };

  const start = parts[0].trim();
  const end = parts[1].trim();
  return {
    start: start || "00:00",
    end: end || "23:59",
  };
};

const timeToMinutes = (timeStr: string): number | null => {
  const match = timeStr.match(/^(\d{1,2}):(\d{2})$/);
  if (!match) return null;
  const hours = Number.parseInt(match[1], 10);
  const minutes = Number.parseInt(match[2], 10);
  if (Number.isNaN(hours) || Number.isNaN(minutes)) return null;
  if (hours < 0 || hours > 23 || minutes < 0 || minutes > 59) return null;
  return hours * 60 + minutes;
};

export const TimeRangeFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const currentValue = (value as string) ?? (field.defaultValue as string) ?? DEFAULT_RANGE;
  const { start, end } = useMemo(() => parseTimeRange(currentValue), [currentValue]);

  const updateRange = (nextStart: string, nextEnd: string) => {
    const startMinutes = timeToMinutes(nextStart);
    const endMinutes = timeToMinutes(nextEnd);
    const safeEnd = startMinutes !== null && endMinutes !== null && endMinutes < startMinutes ? nextStart : nextEnd;
    onChange(`${nextStart} - ${safeEnd}`);
  };

  return (
    <div className="flex items-center gap-2">
      <div className="flex-1">
        <TimeFieldRenderer
          field={{ name: `${field.name || "timeRange"}-start`, label: "Start", type: "time" }}
          value={start}
          onChange={(nextStart) => updateRange((nextStart as string) || "00:00", end)}
        />
      </div>
      <span className="text-sm text-gray-500">-</span>
      <div className="flex-1">
        <TimeFieldRenderer
          field={{ name: `${field.name || "timeRange"}-end`, label: "End", type: "time" }}
          value={end}
          onChange={(nextEnd) => updateRange(start, (nextEnd as string) || "23:59")}
        />
      </div>
    </div>
  );
};
