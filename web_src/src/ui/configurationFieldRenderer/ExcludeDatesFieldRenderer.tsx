import React, { useState, useMemo, useEffect } from "react";
import { Plus, X } from "lucide-react";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { FieldRendererProps } from "./types";

interface ExcludeDatesFieldRendererProps extends FieldRendererProps {
  // All props are inherited from FieldRendererProps
}

const months = [
  { value: 1, label: "Jan" },
  { value: 2, label: "Feb" },
  { value: 3, label: "Mar" },
  { value: 4, label: "Apr" },
  { value: 5, label: "May" },
  { value: 6, label: "Jun" },
  { value: 7, label: "Jul" },
  { value: 8, label: "Aug" },
  { value: 9, label: "Sep" },
  { value: 10, label: "Oct" },
  { value: 11, label: "Nov" },
  { value: 12, label: "Dec" },
];

const daysInMonth = [31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]; // Feb has 29 to account for leap years

const formatDateDisplay = (dateStr: string): string => {
  if (!dateStr) return "";
  const parts = dateStr.split("-");
  if (parts.length !== 2) return dateStr;
  const month = parseInt(parts[0], 10);
  const day = parseInt(parts[1], 10);
  const monthLabel = months.find((m) => m.value === month)?.label || "";
  return `${monthLabel} ${day}`;
};

export const ExcludeDatesFieldRenderer: React.FC<ExcludeDatesFieldRendererProps> = ({
  field,
  value,
  onChange,
  hasError = false,
}) => {
  // Normalize dates array: convert strings to objects { date: string }
  const normalizedDates = useMemo(() => {
    const datesArray = Array.isArray(value) ? value : [];
    return datesArray.map((item) => {
      if (typeof item === "string") {
        return { date: item };
      }
      if (typeof item === "object" && item !== null && "date" in item) {
        return item;
      }
      return item;
    });
  }, [value]);

  // Update parent if normalization changed the data
  useEffect(() => {
    const datesArray = Array.isArray(value) ? value : [];
    const needsNormalization = datesArray.some((item) => typeof item === "string");
    if (needsNormalization) {
      onChange(normalizedDates.length > 0 ? normalizedDates : undefined);
    }
  }, [value, normalizedDates, onChange]);

  const dates = normalizedDates;
  const [selectedMonth, setSelectedMonth] = useState<number>(12); // Default to December
  const [selectedDay, setSelectedDay] = useState<number>(31); // Default to 31
  const [isOpen, setIsOpen] = useState(false);

  // Generate day options for selected month
  const dayOptions = useMemo(() => {
    const maxDays = daysInMonth[selectedMonth - 1];
    return Array.from({ length: maxDays }, (_, i) => i + 1);
  }, [selectedMonth]);

  const handleAddDate = () => {
    const monthStr = selectedMonth.toString().padStart(2, "0");
    const dayStr = selectedDay.toString().padStart(2, "0");
    const newDate = `${monthStr}-${dayStr}`;

    // Check if date already exists
    if (!dates.some((d) => (typeof d === "object" && d !== null ? (d as { date?: string }).date : d) === newDate)) {
      const newDates = [
        ...dates,
        { date: newDate },
      ];
      onChange(newDates);
    }

    // Reset to default
    setSelectedMonth(12);
    setSelectedDay(31);
    setIsOpen(false);
  };

  const handleRemoveDate = (dateToRemove: string) => {
    const newDates = dates.filter((d) => {
      const dateValue = typeof d === "object" && d !== null ? (d as { date?: string }).date : d;
      return dateValue !== dateToRemove;
    });
    onChange(newDates.length > 0 ? newDates : undefined);
  };

  const getDateValue = (item: unknown): string => {
    if (typeof item === "string") return item;
    if (typeof item === "object" && item !== null && "date" in item) {
      return (item as { date: string }).date;
    }
    return "";
  };

  return (
    <div className="space-y-2">
      <Label className={`block text-left ${hasError ? "text-red-600 dark:text-red-400" : ""}`}>
        {field.label || field.name}
        {field.required && <span className="text-gray-800 dark:text-gray-300 ml-1">*</span>}
      </Label>
      <div className="flex items-center gap-2 flex-wrap">
        <Popover open={isOpen} onOpenChange={setIsOpen}>
          <PopoverTrigger asChild>
            <Button type="button" variant="outline" size="icon" className="h-8 w-8">
              <Plus className="h-4 w-4" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-auto p-2" align="start">
            <div className="flex items-center gap-2">
              <Select
                value={selectedMonth.toString()}
                onValueChange={(val) => {
                  const month = parseInt(val, 10);
                  setSelectedMonth(month);
                  // Adjust day if needed
                  const maxDays = daysInMonth[month - 1];
                  if (selectedDay > maxDays) {
                    setSelectedDay(maxDays);
                  }
                }}
              >
                <SelectTrigger className="w-20">
                  <SelectValue placeholder="Month" />
                </SelectTrigger>
                <SelectContent>
                  {months.map((month) => (
                    <SelectItem key={month.value} value={month.value.toString()}>
                      {month.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={selectedDay.toString()} onValueChange={(val) => setSelectedDay(parseInt(val, 10))}>
                <SelectTrigger className="w-16">
                  <SelectValue placeholder="Day" />
                </SelectTrigger>
                <SelectContent>
                  {dayOptions.map((day) => (
                    <SelectItem key={day} value={day.toString()}>
                      {day}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button type="button" onClick={handleAddDate}>
                Add
              </Button>
            </div>
          </PopoverContent>
        </Popover>
        {dates.map((dateItem, index) => {
          const dateValue = getDateValue(dateItem);
          if (!dateValue) return null;
          return (
            <Badge key={index} variant="outline" className="gap-1 h-8 py-2 text-[13px] font-normal">
              {formatDateDisplay(dateValue)}
              <button
                type="button"
                onClick={() => handleRemoveDate(dateValue)}
                className="hover:bg-gray-100 dark:hover:bg-gray-700 rounded p-0.5 -mr-1"
                aria-label="Remove date"
              >
                <X className="h-4 w-4" />
              </button>
            </Badge>
          );
        })}
      </div>
    </div>
  );
};
