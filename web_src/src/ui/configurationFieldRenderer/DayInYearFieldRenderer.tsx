import React from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { FieldRendererProps } from "./types";

export const DayInYearFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const currentValue = (value as string) ?? (field.defaultValue as string) ?? "";

  // Parse current MM/DD value
  const [currentMonth, currentDay] = currentValue
    ? currentValue.split("/").map((n) => parseInt(n, 10))
    : [undefined, undefined];

  const months = [
    { value: 1, label: "January" },
    { value: 2, label: "February" },
    { value: 3, label: "March" },
    { value: 4, label: "April" },
    { value: 5, label: "May" },
    { value: 6, label: "June" },
    { value: 7, label: "July" },
    { value: 8, label: "August" },
    { value: 9, label: "September" },
    { value: 10, label: "October" },
    { value: 11, label: "November" },
    { value: 12, label: "December" },
  ];

  // Get days for the selected month (accounting for leap year)
  const getDaysInMonth = (month: number) => {
    const daysInMonth = [31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
    return month ? daysInMonth[month - 1] : 31;
  };

  const handleMonthChange = (monthStr: string) => {
    const month = parseInt(monthStr, 10);
    if (currentDay) {
      const maxDays = getDaysInMonth(month);
      const day = Math.min(currentDay, maxDays); // Adjust day if it exceeds the month's limit
      const formattedValue = `${month.toString().padStart(2, "0")}/${day.toString().padStart(2, "0")}`;
      onChange(formattedValue);
    } else {
      // Only month selected, wait for day
      onChange(`${month.toString().padStart(2, "0")}/01`);
    }
  };

  const handleDayChange = (dayStr: string) => {
    const day = parseInt(dayStr, 10);
    if (currentMonth) {
      const formattedValue = `${currentMonth.toString().padStart(2, "0")}/${day.toString().padStart(2, "0")}`;
      onChange(formattedValue);
    }
  };

  const maxDays = getDaysInMonth(currentMonth || 1);
  const days = Array.from({ length: maxDays }, (_, i) => i + 1);

  return (
    <div className="flex gap-2">
      <Select value={currentMonth ? currentMonth.toString() : ""} onValueChange={handleMonthChange}>
        <SelectTrigger className={`flex-1 ${hasError ? "border-red-500 border-2" : ""}`}>
          <SelectValue placeholder="Month" />
        </SelectTrigger>
        <SelectContent className="max-h-60">
          {months.map((month) => (
            <SelectItem key={month.value} value={month.value.toString()}>
              {month.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select value={currentDay ? currentDay.toString() : ""} onValueChange={handleDayChange} disabled={!currentMonth}>
        <SelectTrigger className={`flex-1 ${hasError ? "border-red-500 border-2" : ""}`}>
          <SelectValue placeholder="Day" />
        </SelectTrigger>
        <SelectContent className="max-h-60">
          {days.map((day) => (
            <SelectItem key={day} value={day.toString()}>
              {day}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};
