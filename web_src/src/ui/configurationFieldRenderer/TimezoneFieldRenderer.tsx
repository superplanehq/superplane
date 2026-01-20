import React, { useEffect, useRef } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { FieldRendererProps } from "./types";

// Function to get user's current timezone offset as a string (e.g., "-5", "0", "5.5")
const getUserTimezoneOffset = (): string => {
  const offset = -new Date().getTimezoneOffset() / 60;
  return offset.toString();
};

// Timezone options with labels and values
const timezoneOptions = [
  { label: "GMT-12 (Baker Island)", value: "-12" },
  { label: "GMT-11 (American Samoa)", value: "-11" },
  { label: "GMT-10 (Hawaii)", value: "-10" },
  { label: "GMT-9 (Alaska)", value: "-9" },
  { label: "GMT-8 (Los Angeles, Vancouver)", value: "-8" },
  { label: "GMT-7 (Denver, Phoenix)", value: "-7" },
  { label: "GMT-6 (Chicago, Mexico City)", value: "-6" },
  { label: "GMT-5 (New York, Toronto)", value: "-5" },
  { label: "GMT-4 (Santiago, Atlantic)", value: "-4" },
  { label: "GMT-3 (São Paulo, Buenos Aires)", value: "-3" },
  { label: "GMT-2 (South Georgia)", value: "-2" },
  { label: "GMT-1 (Azores)", value: "-1" },
  { label: "GMT+0 (London, Dublin, UTC)", value: "0" },
  { label: "GMT+1 (Paris, Berlin, Rome)", value: "1" },
  { label: "GMT+2 (Cairo, Helsinki, Athens)", value: "2" },
  { label: "GMT+3 (Moscow, Istanbul, Riyadh)", value: "3" },
  { label: "GMT+4 (Dubai, Baku)", value: "4" },
  { label: "GMT+5 (Karachi, Tashkent)", value: "5" },
  { label: "GMT+5:30 (Mumbai, Delhi)", value: "5.5" },
  { label: "GMT+6 (Dhaka, Almaty)", value: "6" },
  { label: "GMT+7 (Bangkok, Jakarta)", value: "7" },
  { label: "GMT+8 (Beijing, Singapore, Perth)", value: "8" },
  { label: "GMT+9 (Tokyo, Seoul)", value: "9" },
  { label: "GMT+9:30 (Adelaide)", value: "9.5" },
  { label: "GMT+10 (Sydney, Melbourne)", value: "10" },
  { label: "GMT+11 (Solomon Islands)", value: "11" },
  { label: "GMT+12 (Auckland, Fiji)", value: "12" },
  { label: "GMT+13 (Tonga, Samoa)", value: "13" },
  { label: "GMT+14 (Kiribati)", value: "14" },
];

export const TimezoneFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const hasSetDefault = useRef(false);

  // Set user's current timezone as default on first render if no value is present
  // or if the value is "current" (which signals to use user's timezone)
  useEffect(() => {
    if (!hasSetDefault.current && (value === undefined || value === null || value === "current")) {
      const userTimezone = getUserTimezoneOffset();
      // Use user's timezone if it matches one of our options, otherwise fallback to "0" (UTC)
      const defaultTimezone = timezoneOptions.find((tz) => tz.value === userTimezone) ? userTimezone : "0";

      onChange(defaultTimezone);
      hasSetDefault.current = true;
    }
  }, [value, field.defaultValue, onChange]);

  // Get the display value - if value is "current", show user's timezone
  const displayValue = (() => {
    if (value === "current") {
      const userTimezone = getUserTimezoneOffset();
      return timezoneOptions.find((tz) => tz.value === userTimezone)?.value ?? "0";
    }
    return (value as string) ?? "0";
  })();

  return (
    <Select value={displayValue} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full">
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
