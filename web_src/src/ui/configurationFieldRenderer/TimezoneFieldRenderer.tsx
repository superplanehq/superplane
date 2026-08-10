import React, { useEffect, useMemo, useRef } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { FieldRendererProps } from "./types";
import { toTestId } from "@/lib/testID";

// The user's current IANA timezone identifier (e.g. "America/New_York").
// This carries DST rules, unlike a bare numeric offset.
const getUserTimezone = (): string => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
};

// The full list of IANA timezone identifiers supported by the runtime.
const getTimezoneIdentifiers = (): string[] => {
  try {
    const supportedValuesOf = (Intl as unknown as { supportedValuesOf?: (key: string) => string[] })
      .supportedValuesOf;
    if (typeof supportedValuesOf === "function") {
      const zones = supportedValuesOf("timeZone");
      // Intl doesn't always include "UTC"; make sure it's selectable.
      return zones.includes("UTC") ? zones : ["UTC", ...zones];
    }
  } catch {
    // Fall through to the minimal fallback below.
  }
  return ["UTC"];
};

// Build a human-friendly label showing the identifier and its current offset,
// e.g. "America/New_York (GMT-04:00)".
const formatTimezoneLabel = (identifier: string): string => {
  try {
    const parts = new Intl.DateTimeFormat("en-US", {
      timeZone: identifier,
      timeZoneName: "shortOffset",
    }).formatToParts(new Date());
    const offset = parts.find((p) => p.type === "timeZoneName")?.value;
    const name = identifier.replace(/_/g, " ");
    return offset ? `${name} (${offset})` : name;
  } catch {
    return identifier.replace(/_/g, " ");
  }
};

export const TimezoneFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const hasSetDefault = useRef(false);
  const testId = field.name ? toTestId(`field-${field.name}-select`) : undefined;

  const timezoneOptions = useMemo(
    () => getTimezoneIdentifiers().map((identifier) => ({ label: formatTimezoneLabel(identifier), value: identifier })),
    [],
  );

  // Set the user's current timezone as the default on first render if no value is
  // present or if the value is "current" (which signals to use the user's timezone).
  useEffect(() => {
    if (!hasSetDefault.current && (value === undefined || value === null || value === "current")) {
      const userTimezone = getUserTimezone();
      // Use the user's timezone if it's one of our options, otherwise fall back to UTC.
      const defaultTimezone = timezoneOptions.some((tz) => tz.value === userTimezone) ? userTimezone : "UTC";

      onChange(defaultTimezone);
      hasSetDefault.current = true;
    }
  }, [value, field.defaultValue, onChange, timezoneOptions]);

  // Get the display value - if value is "current", show the user's timezone.
  const displayValue = (() => {
    if (value === "current") {
      const userTimezone = getUserTimezone();
      return timezoneOptions.some((tz) => tz.value === userTimezone) ? userTimezone : "UTC";
    }
    return (value as string) ?? "UTC";
  })();

  return (
    <Select value={displayValue} onValueChange={(val) => onChange(val || undefined)}>
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
