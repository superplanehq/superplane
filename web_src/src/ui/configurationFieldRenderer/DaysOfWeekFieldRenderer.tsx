import React from "react";
import type { FieldRendererProps } from "./types";
import { cn } from "@/lib/utils";

const DAYS_OF_WEEK = [
  { value: "monday", label: "Mo" },
  { value: "tuesday", label: "Tu" },
  { value: "wednesday", label: "We" },
  { value: "thursday", label: "Th" },
  { value: "friday", label: "Fr" },
  { value: "saturday", label: "Sa" },
  { value: "sunday", label: "Su" },
];

export const DaysOfWeekFieldRenderer: React.FC<FieldRendererProps> = ({ value, onChange, hasError }) => {
  const selectedDays = Array.isArray(value) ? value : value ? [value] : [];
  const selectedDaysSet = new Set(selectedDays);

  const toggleDay = (dayValue: string) => {
    const newSelectedDays = [...selectedDays];
    const dayIndex = newSelectedDays.indexOf(dayValue);

    if (dayIndex > -1) {
      newSelectedDays.splice(dayIndex, 1);
    } else {
      newSelectedDays.push(dayValue);
    }

    onChange(newSelectedDays.length > 0 ? newSelectedDays : undefined);
  };

  return (
    <div className={cn("flex gap-2", hasError && "border-red-500 border-2 rounded p-2")}>
      {DAYS_OF_WEEK.map((day) => {
        const isSelected = selectedDaysSet.has(day.value);

        return (
          <button
            key={day.value}
            type="button"
            onClick={() => toggleDay(day.value)}
            className={cn(
              "w-9 h-9 rounded-full text-sm font-medium focus:outline-none",
              isSelected
                ? "bg-action-primary text-action-primary-content"
                : "bg-surface-raised border border-edge-strong text-content-secondary hover:border-focus-ring",
            )}
            aria-label={day.value}
            aria-pressed={isSelected}
          >
            {day.label}
          </button>
        );
      })}
    </div>
  );
};
