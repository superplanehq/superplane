import React from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { FieldRendererProps } from "./types";

// Function to get user's current timezone offset as a string (e.g., "-5", "0", "5.5")
const getUserTimezoneOffset = (): string => {
  const offsetMinutes = new Date().getTimezoneOffset();
  // getTimezoneOffset returns minutes behind UTC, so we negate to get offset ahead
  const offsetHours = -offsetMinutes / 60;
  
  // Handle half-hour offsets (e.g., 5.5 for India)
  if (offsetHours % 1 === 0.5 || offsetHours % 1 === -0.5) {
    return offsetHours.toString();
  }
  
  // Round to nearest whole number for edge cases
  return Math.round(offsetHours).toString();
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

export const TimezoneFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  // Calculate browser timezone once
  const browserTimezone = React.useMemo(() => {
    const userTimezone = getUserTimezoneOffset();
    return timezoneOptions.find((tz) => tz.value === userTimezone)?.value || "0";
  }, []);

  // Set user's current timezone as default on mount if no value is present
  // Only run once on initial mount to avoid loops
  React.useEffect(() => {
    if (value === undefined || value === null || value === "") {
      onChange(browserTimezone);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Empty deps array - only run once on mount

  // Get the display value - use the value if set, otherwise use browser timezone
  // This ensures the Select shows the correct timezone even before state is updated
  const displayValue = React.useMemo(() => {
    const val = value !== undefined && value !== null && value !== "" ? (value as string) : browserTimezone;
    // Ensure the value matches one of our options, otherwise fallback to "0"
    const matchedOption = timezoneOptions.find((opt) => opt.value === val);
    return matchedOption ? matchedOption.value : "0";
  }, [value, browserTimezone]);

  return (
    <Select value={displayValue} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className={`w-full ${hasError ? "border-red-500 border-2" : ""}`}>
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
