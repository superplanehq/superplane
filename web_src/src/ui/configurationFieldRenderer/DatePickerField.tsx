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
  // Parse value as MM-DD format (recurring date, no year)
  const currentYear = new Date().getFullYear();
  
  // Parse the stored value (could be YYYY-MM-DD or MM-DD format)
  const normalizedDateValue = React.useMemo(() => {
    if (!value) return undefined;
    
    const valueStr = value as string;
    // Try parsing as MM-DD first
    if (valueStr.match(/^\d{2}-\d{2}$/)) {
      const [month, day] = valueStr.split('-').map(Number);
      return new Date(currentYear, month - 1, day);
    }
    // Try parsing as YYYY-MM-DD (backward compatibility)
    if (valueStr.match(/^\d{4}-\d{2}-\d{2}$/)) {
      const date = new Date(valueStr);
      if (!isNaN(date.getTime())) {
        return new Date(currentYear, date.getMonth(), date.getDate());
      }
    }
    return undefined;
  }, [value, currentYear]);
  
  // Initialize month from dateValue or current date
  const [month, setMonth] = React.useState<Date | undefined>(() => {
    if (normalizedDateValue && !isNaN(normalizedDateValue.getTime())) {
      return new Date(currentYear, normalizedDateValue.getMonth(), 1);
    }
    return new Date(currentYear, new Date().getMonth(), 1);
  });
  
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
      // Store as MM-DD format (recurring date, no year)
      const month = String(date.getMonth() + 1).padStart(2, "0");
      const day = String(date.getDate()).padStart(2, "0");
      onChange(`${month}-${day}`);
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
            !normalizedDateValue && "text-muted-foreground",
            hasError && "border-red-500 border-2"
          )}
          data-testid={toTestId(`date-field-${field.name}`)}
        >
          <CalendarIcon className="mr-2 h-4 w-4" />
          {normalizedDateValue ? format(normalizedDateValue, "MMMM d") : <span>Pick a date</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto overflow-hidden p-0" align="start">
        {/* #region agent log */}
        {React.useEffect(() => {
          setTimeout(() => {
            const popoverEl = document.querySelector('[data-radix-popper-content-wrapper]');
            const popoverContent = popoverEl?.querySelector('[role="dialog"]') || popoverEl?.firstElementChild;
            const calendarEl = popoverEl?.querySelector('[data-slot="calendar"]');
            if (calendarEl && popoverContent) {
              // Check DOM hierarchy
              let parent = calendarEl.parentElement;
              const parentChain: string[] = [];
              while (parent && parent !== document.body) {
                parentChain.push(`${parent.tagName}${parent.className ? '.' + parent.className.split(' ')[0] : ''}${parent.getAttribute('data-slot') ? '[data-slot=' + parent.getAttribute('data-slot') + ']' : ''}`);
                parent = parent.parentElement;
              }
              
              // Check if CSS selector matches
              const testEl = document.createElement('div');
              testEl.className = calendarEl.className;
              document.body.appendChild(testEl);
              const testComputed = window.getComputedStyle(testEl);
              const testBg = testComputed.backgroundColor;
              document.body.removeChild(testEl);
              
              const calendarComputed = window.getComputedStyle(calendarEl);
              const popoverComputed = window.getComputedStyle(popoverContent as Element);
              const logData = {
                location: 'DatePickerField.tsx:82',
                message: 'DOM structure and CSS selector analysis',
                data: {
                  calendar: {
                    className: calendarEl.className,
                    bgColor: calendarComputed.backgroundColor,
                    border: calendarComputed.border,
                    borderRadius: calendarComputed.borderRadius,
                    padding: calendarComputed.padding,
                    hasDataSlot: calendarEl.getAttribute('data-slot'),
                    parentChain: parentChain
                  },
                  popover: {
                    className: (popoverContent as Element).className,
                    bgColor: popoverComputed.backgroundColor,
                    border: popoverComputed.border,
                    borderRadius: popoverComputed.borderRadius,
                    padding: popoverComputed.padding,
                    hasDataSlot: (popoverContent as Element).getAttribute('data-slot')
                  },
                  cssSelectorTest: {
                    isolatedBgColor: testBg,
                    matchesPopoverParent: parentChain.some(p => p.includes('popover-content'))
                  }
                },
                timestamp: Date.now(),
                sessionId: 'debug-session',
                runId: 'post-fix-4',
                hypothesisId: 'I'
              };
              fetch('http://127.0.0.1:7242/ingest/f719ffac-e1c8-4cef-8f17-d4bc91ac736c',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(logData)}).catch(()=>{});
            }
          }, 200);
        }, [])}
        {/* #endregion */}
        <Calendar
          mode="single"
          selected={normalizedDateValue}
          onSelect={handleDateSelect}
          captionLayout="dropdown"
          month={month}
          onMonthChange={setMonth}
          fromYear={currentYear}
          toYear={currentYear}
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
      </PopoverContent>
    </Popover>
  );
};
