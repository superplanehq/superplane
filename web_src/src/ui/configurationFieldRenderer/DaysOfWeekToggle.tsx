import React from "react";
import { FieldRendererProps } from "./types";
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

export const DaysOfWeekToggle: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  // Get current selected days as an array
  const selectedDays = Array.isArray(value) ? value : value ? [value] : [];
  
  // Convert to Set for easier checking
  const selectedDaysSet = new Set(selectedDays);

  const toggleDay = (dayValue: string) => {
    const newSelectedDays = [...selectedDays];
    const dayIndex = newSelectedDays.indexOf(dayValue);
    
    if (dayIndex > -1) {
      // Remove day if already selected
      newSelectedDays.splice(dayIndex, 1);
    } else {
      // Add day if not selected
      newSelectedDays.push(dayValue);
    }
    
    // Update value - use undefined if no days selected, otherwise use the array
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
              "w-10 h-10 rounded-full text-sm font-medium transition-all",
              "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2",
              isSelected
                ? "bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900"
                : "bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:border-gray-400 dark:hover:border-gray-500"
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
