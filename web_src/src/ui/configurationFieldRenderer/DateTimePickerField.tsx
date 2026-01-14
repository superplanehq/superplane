import React from "react";
import { format } from "date-fns";
import { Calendar as CalendarIcon } from "lucide-react";
import { Calendar } from "@/ui/calendar";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";

export const DateTimePickerField: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  hasError,
  allValues = {},
}) => {
  const dateTimeValue = React.useMemo(() => {
    if (!value) return undefined;
    const date = new Date(value as string);
    return !isNaN(date.getTime()) ? date : undefined;
  }, [value]);

  const [selectedDate, setSelectedDate] = React.useState<Date | undefined>(dateTimeValue);
  const [selectedTime, setSelectedTime] = React.useState<string>(dateTimeValue ? format(dateTimeValue, "HH:mm") : "");

  // Sync state when value prop changes
  React.useEffect(() => {
    if (dateTimeValue) {
      setSelectedDate(dateTimeValue);
      setSelectedTime(format(dateTimeValue, "HH:mm"));
    } else {
      setSelectedDate(undefined);
      setSelectedTime("");
    }
  }, [dateTimeValue]);

  // Calculate min/max datetime based on other fields
  const getDateTimeConstraints = React.useMemo(() => {
    let minDateTime: Date | undefined;
    let maxDateTime: Date | undefined;

    // For endDateTime field, prevent selecting datetimes before startDateTime
    if (field.name === "endDateTime" && allValues.startDateTime) {
      const startDateTime = new Date(allValues.startDateTime as string);
      if (!isNaN(startDateTime.getTime())) {
        minDateTime = startDateTime;
      }
    }

    // For startDateTime field, prevent selecting datetimes after endDateTime
    if (field.name === "startDateTime" && allValues.endDateTime) {
      const endDateTime = new Date(allValues.endDateTime as string);
      if (!isNaN(endDateTime.getTime())) {
        maxDateTime = endDateTime;
      }
    }

    return { minDateTime, maxDateTime };
  }, [field.name, allValues.startDateTime, allValues.endDateTime]);

  const handleDateSelect = (date: Date | undefined) => {
    setSelectedDate(date);
    if (date && selectedTime) {
      const [hours, minutes] = selectedTime.split(":");
      const newDateTime = new Date(date);
      newDateTime.setHours(parseInt(hours, 10), parseInt(minutes, 10));
      onChange(newDateTime.toISOString());
    } else if (date) {
      // If no time is selected, use current time
      const now = new Date();
      const newDateTime = new Date(date);
      newDateTime.setHours(now.getHours(), now.getMinutes());
      setSelectedTime(format(newDateTime, "HH:mm"));
      onChange(newDateTime.toISOString());
    }
  };

  const handleTimeChange = (time: string) => {
    setSelectedTime(time);
    if (selectedDate && time) {
      const [hours, minutes] = time.split(":");
      const newDateTime = new Date(selectedDate);
      newDateTime.setHours(parseInt(hours, 10), parseInt(minutes, 10));
      onChange(newDateTime.toISOString());
    }
  };

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className={cn(
            "w-full justify-start text-left font-normal",
            !dateTimeValue && "text-muted-foreground",
            hasError && "border-red-500 border-2",
          )}
          data-testid={toTestId(`datetime-field-${field.name}`)}
        >
          <CalendarIcon className="mr-2 h-4 w-4" />
          {dateTimeValue ? <span>{format(dateTimeValue, "PPP p")}</span> : <span>Pick a date and time</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-4" align="start">
        <div className="space-y-4">
          <Calendar
            mode="single"
            selected={selectedDate}
            onSelect={handleDateSelect}
            disabled={(date) => {
              if (getDateTimeConstraints.minDateTime && date < getDateTimeConstraints.minDateTime) {
                return true;
              }
              if (getDateTimeConstraints.maxDateTime && date > getDateTimeConstraints.maxDateTime) {
                return true;
              }
              return false;
            }}
            initialFocus
          />
          <div className="border-t pt-4">
            <Input
              type="time"
              value={selectedTime}
              onChange={(e) => handleTimeChange(e.target.value)}
              className="bg-background appearance-none [&::-webkit-calendar-picker-indicator]:hidden [&::-webkit-calendar-picker-indicator]:appearance-none"
            />
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
};
