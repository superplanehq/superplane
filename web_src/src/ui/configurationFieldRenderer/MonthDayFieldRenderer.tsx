import React, { useMemo } from "react";
import { AutoCompleteSelect } from "@/components/AutoCompleteSelect/AutoCompleteSelect";
import { FieldRendererProps } from "./types";

export const MonthDayFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const currentValue = (value as string) ?? (field.defaultValue as string) ?? "";

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

  const daysInMonth = [31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]; // Feb has 29 to account for leap years

  // Parse current value to extract month and day
  const parseValue = (val: string): { month: number | undefined; day: number | undefined } => {
    if (!val) return { month: undefined, day: undefined };

    let month: number | undefined;
    let day: number | undefined;

    // Try MM-DD format first
    if (val.match(/^\d{2}-\d{2}$/)) {
      const parts = val.split("-").map((n) => parseInt(n, 10));
      month = parts[0];
      day = parts[1];
    }
    // Try YYYY-MM-DD format (backward compatibility)
    else if (val.match(/^\d{4}-\d{2}-\d{2}$/)) {
      const parts = val.split("-").map((n) => parseInt(n, 10));
      month = parts[1];
      day = parts[2];
    }

    return { month, day };
  };

  const { month: currentMonth, day: currentDay } = parseValue(currentValue);

  // Generate month options
  const monthOptions = useMemo(() => {
    return months.map((month) => ({
      value: month.value.toString(),
      label: month.label,
    }));
  }, []);

  // Generate day options for selected month
  const dayOptions = useMemo(() => {
    if (!currentMonth) return [];

    const maxDays = daysInMonth[currentMonth - 1];
    return Array.from({ length: maxDays }, (_, i) => {
      const day = i + 1;
      return {
        value: day.toString(),
        label: day.toString(),
      };
    });
  }, [currentMonth]);

  const handleMonthChange = (monthValue: string) => {
    const monthNum = parseInt(monthValue, 10);
    
    // If there was a previously selected day, adjust it if needed
    if (currentDay) {
      const maxDays = daysInMonth[monthNum - 1];
      const adjustedDay = Math.min(currentDay, maxDays);
      const monthStr = monthNum.toString().padStart(2, "0");
      const dayStr = adjustedDay.toString().padStart(2, "0");
      onChange(`${monthStr}-${dayStr}`);
    } else {
      // Just set month, wait for day
      const monthStr = monthNum.toString().padStart(2, "0");
      onChange(`${monthStr}-01`);
    }
  };

  const handleDayChange = (dayValue: string) => {
    if (!currentMonth) return;

    const day = parseInt(dayValue, 10);
    const monthStr = currentMonth.toString().padStart(2, "0");
    const dayStr = day.toString().padStart(2, "0");
    onChange(`${monthStr}-${dayStr}`);
  };

  const errorClassName = hasError ? "border-red-500 border-2" : "";

  return (
    <div className="flex items-center">
      <div className="flex-1">
        <AutoCompleteSelect
          options={monthOptions}
          value={currentMonth ? currentMonth.toString() : ""}
          onChange={handleMonthChange}
          placeholder="Month"
          error={hasError}
          className={`${errorClassName} rounded-r-none border-r-0`}
        />
      </div>
      <div className="flex-1">
        <AutoCompleteSelect
          options={dayOptions}
          value={currentDay ? currentDay.toString() : ""}
          onChange={handleDayChange}
          placeholder="Day"
          error={hasError}
          disabled={!currentMonth}
          className={`${errorClassName} rounded-l-none`}
        />
      </div>
    </div>
  );
};
