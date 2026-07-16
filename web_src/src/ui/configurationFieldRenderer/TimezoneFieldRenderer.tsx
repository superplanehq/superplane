import React, { useEffect, useRef } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { FieldRendererProps } from "./types";
import { toTestId } from "@/lib/testID";
import { resolveDefaultTimezoneValue, resolveTimezoneDisplayValue, timezoneOptions } from "./timezoneDisplayValue";
import { useSkipDefaultsAfterReadOnly } from "./useSkipDefaultsAfterReadOnly";

export const TimezoneFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, readOnly = false }) => {
  const hasSetDefault = useRef(false);
  const skipDefaultsAfterReadOnly = useSkipDefaultsAfterReadOnly(readOnly);
  const testId = field.name ? toTestId(`field-${field.name}-select`) : undefined;

  // Set user's current timezone as default on first render if no value is present
  // or if the value is "current" (which signals to use user's timezone)
  useEffect(() => {
    if (readOnly || skipDefaultsAfterReadOnly) {
      return;
    }

    if (!hasSetDefault.current && (value === undefined || value === null || value === "current")) {
      onChange(resolveDefaultTimezoneValue());
      hasSetDefault.current = true;
    }
  }, [readOnly, skipDefaultsAfterReadOnly, value, field.defaultValue, onChange]);

  // Get the display value - unset and "current" both resolve to the user's timezone
  const displayValue = resolveTimezoneDisplayValue(value);

  return (
    <Select value={displayValue} onValueChange={(val) => onChange(val || undefined)} disabled={readOnly}>
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder={`Select ${field.label || field.name}`} />
      </SelectTrigger>
      <SelectContent className="max-h-60">
        {timezoneOptions.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
