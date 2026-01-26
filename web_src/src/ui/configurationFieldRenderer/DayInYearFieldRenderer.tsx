import React, { useMemo } from "react";
import { AutoCompleteSelect } from "@/components/AutoCompleteSelect/AutoCompleteSelect";
import { FieldRendererProps } from "./types";

export const DayInYearFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
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

  const daysInMonth = [31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];

  const parseValue = (val: string): { month: number | undefined; day: number | undefined } => {
    if (!val) return { month: undefined, day: undefined };

    let month: number | undefined;
    let day: number | undefined;

    if (val.match(/^\d{2}\/\d{2}$/)) {
      const parts = val.split("/").map((n) => parseInt(n, 10));
      month = parts[0];
      day = parts[1];
    } else if (val.match(/^\d{2}-\d{2}$/)) {
      const parts = val.split("-").map((n) => parseInt(n, 10));
      month = parts[0];
      day = parts[1];
    } else if (val.match(/^\d{4}-\d{2}-\d{2}$/)) {
      const parts = val.split("-").map((n) => parseInt(n, 10));
      month = parts[1];
      day = parts[2];
    }

    return { month, day };
  };

  const { month: currentMonth, day: currentDay } = parseValue(currentValue);

  const monthOptions = useMemo(() => {
    return months.map((month) => ({
      value: month.value.toString(),
      label: month.label,
    }));
  }, []);

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

  const handleMonthChange = (monthStr: string) => {
    const month = parseInt(monthStr, 10);
    if (currentDay) {
      const maxDays = daysInMonth[month - 1];
      const day = Math.min(currentDay, maxDays);
      const formattedValue = `${month.toString().padStart(2, "0")}/${day.toString().padStart(2, "0")}`;
      onChange(formattedValue);
    } else {
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

  const errorClassName = hasError ? "border-red-500 border-2" : "";

  return (
    <div className="flex items-center gap-0 flex-1 min-w-0">
      <div className="flex-[0.5] min-w-0 w-full">
        <AutoCompleteSelect
          options={monthOptions}
          value={currentMonth ? currentMonth.toString() : ""}
          onChange={handleMonthChange}
          placeholder="Month"
          error={hasError}
          className={`${errorClassName} rounded-r-none border-r-0`}
        />
      </div>
      <div className="flex-[0.5] min-w-0 w-full">
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
