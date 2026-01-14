import React from "react";
import { format } from "date-fns";
import { Calendar as CalendarIcon } from "lucide-react";
import { Calendar } from "@/ui/calendar";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { cn } from "@/lib/utils";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";

export const DatePickerField: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  hasError,
  allValues = {},
}) => {
  const dateValue = value ? new Date(value as string) : undefined;
  
  // Calculate min/max dates based on other fields
  const getDateConstraints = React.useMemo(() => {
    let minDate: Date | undefined;
    let maxDate: Date | undefined;

    // For endDate field, prevent selecting dates before startDate
    if (field.name === "endDate" && allValues.startDate) {
      const startDate = new Date(allValues.startDate as string);
      if (!isNaN(startDate.getTime())) {
        minDate = startDate;
      }
    }

    // For startDate field, prevent selecting dates after endDate
    if (field.name === "startDate" && allValues.endDate) {
      const endDate = new Date(allValues.endDate as string);
      if (!isNaN(endDate.getTime())) {
        maxDate = endDate;
      }
    }

    return { minDate, maxDate };
  }, [field.name, allValues.startDate, allValues.endDate]);

  const handleDateSelect = (date: Date | undefined) => {
    if (date) {
      // Format as YYYY-MM-DD for HTML date input compatibility
      const year = date.getFullYear();
      const month = String(date.getMonth() + 1).padStart(2, "0");
      const day = String(date.getDate()).padStart(2, "0");
      onChange(`${year}-${month}-${day}`);
    } else {
      onChange(undefined);
    }
  };

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className={cn(
            "w-full justify-start text-left font-normal",
            !dateValue && "text-muted-foreground",
            hasError && "border-red-500 border-2"
          )}
          data-testid={toTestId(`date-field-${field.name}`)}
        >
          <CalendarIcon className="mr-2 h-4 w-4" />
          {dateValue ? format(dateValue, "PPP") : <span>Pick a date</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <div data-slot="popover-content">
          <Calendar
            mode="single"
            selected={dateValue}
            onSelect={handleDateSelect}
            disabled={(date) => {
              if (getDateConstraints.minDate && date < getDateConstraints.minDate) {
                return true;
              }
              if (getDateConstraints.maxDate && date > getDateConstraints.maxDate) {
                return true;
              }
              return false;
            }}
            initialFocus
          />
        </div>
      </PopoverContent>
    </Popover>
  );
};
