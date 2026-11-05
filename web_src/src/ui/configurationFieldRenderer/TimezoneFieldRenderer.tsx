import React, { useEffect, useRef } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { scheduleTimezones } from "@/lib/scheduleTimezones";
import { toTestId } from "@/lib/testID";
import type { FieldRendererProps } from "./types";

// Function to get user's current timezone offset as a string (e.g., "-5", "0", "5.5")
const getUserTimezoneOffset = (): string => {
  const offset = -new Date().getTimezoneOffset() / 60;
  return offset.toString();
};

export const TimezoneFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const hasSetDefault = useRef(false);
  const testId = field.name ? toTestId(`field-${field.name}-select`) : undefined;

  // Set user's current timezone as default on first render if no value is present
  // or if the value is "current" (which signals to use user's timezone)
  useEffect(() => {
    if (!hasSetDefault.current && (value === undefined || value === null || value === "current")) {
      const userTimezone = getUserTimezoneOffset();
      // Use user's timezone if it matches one of our options, otherwise fallback to "0" (UTC)
      const defaultTimezone = scheduleTimezones.find((tz) => tz.value === userTimezone) ? userTimezone : "0";

      onChange(defaultTimezone);
      hasSetDefault.current = true;
    }
  }, [value, field.defaultValue, onChange]);

  // Get the display value - if value is "current", show user's timezone
  const displayValue = (() => {
    if (value === "current") {
      const userTimezone = getUserTimezoneOffset();
      return scheduleTimezones.find((tz) => tz.value === userTimezone)?.value ?? "0";
    }
    return (value as string) ?? "0";
  })();

  return (
    <Select value={displayValue} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder={`Select ${field.label || field.name}`} />
      </SelectTrigger>
      <SelectContent className="max-h-60">
        {scheduleTimezones.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
